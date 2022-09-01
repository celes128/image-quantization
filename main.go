package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"math"
	"os"
	"sort"
)

func main() {
	// Setup the command line flags and retrieve their values.
	srcFilepath := flag.String("in", "", "input image filepath")
	outFilepath := flag.String("out", "", "output image filepath")
	paletteMaxSize := flag.Int("pal", 4, "maximum size of the palette")
	bayerMatSize := flag.Int("bay", 4, "Bayer dithering matrix size (2, 4 or 8)")
	flag.Parse()

	// Open the source image file.
	img, err := GetImageFromFilePath(*srcFilepath)
	if err != nil {
		fmt.Printf("%v", err)
		return
	}

	// Process the image; apply color quantization.
	outImage := QuantizeImage(img, *paletteMaxSize, *bayerMatSize)

	// Write the resulting image to a file.
	err = WriteImageToFile(outImage, *outFilepath)
	if err != nil {
		fmt.Printf("%v", err)
		return
	}
}

//
// 			Image processing functions.
//

// QuantizeImage creates an image similar to the one given as input.
// Its number of colors is reduced to approximately <paletteMaxSize> and a Bayer dithering
// is also applied to approximate the original image.
func QuantizeImage(img image.Image, paletteMaxSize int, bayerMatSize int) image.Image {
	// Create a color palette from the input image.
	palette := PaletteFromImage(img, paletteMaxSize)

	// Create the resulting image; undefined pixel colors for now.
	outImage := image.NewRGBA(img.Bounds())

	// Compute its pixels by applying dithering to the source image.
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			// Get the pixel color in the source image.
			c := PixelColor(img, x, y)

			// Apply Bayer dithering.
			ditheredColor := BayerDithering(c, x, y, len(palette), bayerMatSize)

			// Select a color from the palette.
			outColor := NearestColor(ditheredColor, palette)

			// Finally we can write the pixel color in the result image.
			outImage.Set(x, y, outColor)
		}
	}

	return outImage
}

// Only three sizes for the Bayer matrix are supported: 2, 4 or 8.
// If the parameter <bayerMatSize> is different from all these values then a size of 8 is selected.
func BayerCoefficient(x, y int, bayerMatSize int) float64 {
	if bayerMatSize != 2 && bayerMatSize != 4 && bayerMatSize != 8 {
		bayerMatSize = 8
	}

	// The Bayer matrices are stored in an array.
	// This integer is the array index of the matrix coefficient.
	i := (y%bayerMatSize)*bayerMatSize + (x % bayerMatSize)

	// Retrieve the coefficient.
	coef := 0.
	switch bayerMatSize {
	case 2:
		mat := [...]float64{0., 2., 3., 1.}
		coef = mat[i]
	case 4:
		mat := [...]float64{
			0., 8., 2., 10.,
			12., 4., 14., 6.,
			3., 11., 1., 9.,
			15., 7., 13., 5.,
		}
		coef = mat[i]
	default:
		// Use a matrix size of 8.
		mat := [...]float64{
			0., 32., 8., 40., 2., 34., 10., 42.,
			48., 16., 56., 24., 50., 18., 58., 26.,
			12., 44., 4., 36., 14., 46., 6., 38.,
			60., 28., 52., 20., 62., 30., 54., 22.,
			3., 35., 11., 43., 1., 33., 9., 41.,
			51., 19., 59., 27., 49., 17., 57., 25.,
			15., 47., 7., 39., 13., 45., 5., 37.,
			63., 31., 55., 23., 61., 29., 53., 21.,
		}
		coef = mat[i]
	}

	coef /= float64(bayerMatSize * bayerMatSize)
	coef -= 0.5

	return coef
}

// BayerDithering transforms a pixel color using Bayer dithering of order 4, for now.
// TODO: add a function parameter N (N = 2, 4 or 8) so that client can choose the matrix size.
func BayerDithering(c color.RGBA, x, y int, paletteSize int, bayerMatSize int) color.RGBA {
	coef := BayerCoefficient(x, y, bayerMatSize)
	R := 255. / (float64(paletteSize))
	k := R * coef

	// Manually add the color offset to each channel value.
	// We work with floats because the offset can be negative.
	// Do not work with uint8!
	r := float64(c.R) + k
	g := float64(c.G) + k
	b := float64(c.B) + k

	return color.RGBA{
		uint8(ClampF64(r, 0., 255.)),
		uint8(ClampF64(g, 0., 255.)),
		uint8(ClampF64(b, 0., 255.)),
		255,
	}
}

//
// 			Image functions.
//

// PixelColor returns the color of the pixel located at column x and row y in a given image.
func PixelColor(img image.Image, x, y int) color.RGBA {
	r, g, b, a := img.At(x, y).RGBA()

	return color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
}

//
// 			Palette functions.
//

// PaletteFromImage generates a color palette from a given iamge.
// The number of colors in the palette is at most paletteMaxSize.
// The palette can contain duplicated colors however.
// The algorithm is described here: https://en.wikipedia.org/wiki/Median_cut
func PaletteFromImage(img image.Image, paletteMaxSize int) []color.RGBA {
	// Adjust some input here.
	paletteMaxSize = ClampBelowInt(paletteMaxSize, 2)
	fmt.Printf("paletteMaxSize: %d\n", paletteMaxSize)

	// Sort the pixels according to the red color channel.
	pixels := RedSortedImagePixels(img)

	// If the image is very very small, its number of pixels may be less than the
	// input parameter paletteMaxSize. In this case we must adjust the palette size.
	estimatedPaletteSize := ClampAboveInt(paletteMaxSize, len(pixels))
	fmt.Printf("estimatedPaletteSize: %d\n", estimatedPaletteSize)

	// Determine the palette colors. Each color is defined as the mean value of the pixels colors in a bucket.
	// A bucket is a range of pixels. All buckets have the same size except for the last one which has, most of the time,
	// a smaller size.
	bucketSize := len(pixels) / estimatedPaletteSize
	fmt.Printf("bucketSize: %d\n", bucketSize)
	var palette []color.RGBA
	for i := 0; i < estimatedPaletteSize; i++ {
		// Compute the mean color of bucket #i.
		begin := i * bucketSize
		end := ClampAboveInt(begin+bucketSize, len(pixels))
		c := MeanColorOfRange(pixels, begin, end)

		// Note here that this "append" may add a duplicated color in the palette.
		// For now we allow the palette to contain the same color more than once.
		palette = append(palette, c)
	}

	return palette
}

// MeanColorOfRange computes the mean color of a range of colors stored in a slice.
func MeanColorOfRange(pixels []color.RGBA, begin, end int) color.RGBA {
	r, g, b := 0., 0., 0.
	for j := begin; j < end; j++ {
		r += float64(pixels[j].R)
		g += float64(pixels[j].G)
		b += float64(pixels[j].B)
	}

	n := float64(end - begin)

	return color.RGBA{
		uint8(r / n),
		uint8(g / n),
		uint8(b / n),
		255,
	}
}

// RedSortedImagePixels collects and sorts all the pixels colors in a given image.
// The colors are sorted in ascending order with respect to the red channel.
func RedSortedImagePixels(img image.Image) []color.RGBA {
	var pixels []color.RGBA
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			c := color.RGBA{
				uint8(r),
				uint8(g),
				uint8(b),
				uint8(a),
			}

			pixels = append(pixels, c)
		}
	}

	sort.SliceStable(pixels, func(i, j int) bool { return pixels[i].R < pixels[j].R })

	return pixels
}

// NearestColor returns the palette color that is the closest to the given color c.
// The distance in the color space is the Euclidean distance.
func NearestColor(c color.RGBA, palette []color.RGBA) color.RGBA {
	// Assert(len(palette) >= 1)
	minD := ColorDistance(c, palette[0])
	nearest := palette[0]

	for i := 1; i < len(palette); i++ {
		d := ColorDistance(c, palette[i])
		if d < minD {
			minD = d
			nearest = palette[i]
		}
	}

	return nearest
}

//
// 			Color manipulation functions.
//

// ColorDistance computes the Euclidean distance between two colors.
// Note however that the alpha channel is ignored.
func ColorDistance(c1, c2 color.RGBA) float64 {
	// Euclidean distance
	dr := float64(c1.R) - float64(c2.R)
	dr *= dr

	dg := float64(c1.G) - float64(c2.G)
	dg *= dg

	db := float64(c1.B) - float64(c2.B)
	db *= db

	return math.Sqrt(float64(dr + dg + db))
}

// LinearGradient computes the following linear combination of colors c1 and c2: s * c1 + t * c2.
// The resulting alpha channel is set to 255.
func LinearGradient(s float64, c1 color.RGBA, t float64, c2 color.RGBA) color.RGBA {
	sc1 := ScalMult(s, c1)
	tc2 := ScalMult(t, c2)

	return Add(sc1, tc2)
}

// Add adds the three color channels of two colors.
// The resulting alpha channel is set to 255.
func Add(c1, c2 color.RGBA) color.RGBA {
	return color.RGBA{
		ClampU8(c1.R+c2.R, 0, 255),
		ClampU8(c1.G+c2.G, 0, 255),
		ClampU8(c1.B+c2.B, 0, 255),
		255,
	}
}

// ScalMult multiplies the RGB channels of a color by a scalar lambda in [0.0, 1.0].
// The alpha channel remains unchanged.
func ScalMult(lambda float64, c color.RGBA) color.RGBA {
	// Clamp lambda to [0, 1]
	if lambda < 0. {
		lambda = 0.
	} else if lambda > 1. {
		lambda = 1.
	}

	return color.RGBA{
		uint8(lambda * float64(c.R)),
		uint8(lambda * float64(c.G)),
		uint8(lambda * float64(c.B)),
		c.A,
	}
}

//
// 			Image file read/write functions.
//

// GetImageFromFilePath returns an image.Image object from an image filepath.
func GetImageFromFilePath(filePath string) (image.Image, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	image, _, err := image.Decode(f)
	return image, err
}

// WriteImageToFile saves an image to a file.
func WriteImageToFile(img image.Image, filepath string) error {
	outputFile, err := os.Create(filepath)
	if err != nil {
		return err
	}

	// Encode takes a writer interface and an image interface
	// We pass it the File and the RGBA
	png.Encode(outputFile, img)

	// Don't forget to close files
	outputFile.Close()

	return nil
}

//
// 			General functions.
//

// ClampU8 clamps an uint8 <x> inside the range [low; high].
func ClampU8(x, low, high uint8) uint8 {
	if x < low {
		return low
	} else if x > high {
		return high
	} else {
		return x
	}
}

// ClampF64 clamps a float <x> inside the range [low; high].
func ClampF64(x, low, high float64) float64 {
	if x < low {
		return low
	} else if x > high {
		return high
	} else {
		return x
	}
}

// ClampAboveInt clamps an integer <x> if it is below the value <low>.
func ClampBelowInt(x, low int) int {
	if x < low {
		return low
	} else {
		return x
	}
}

// ClampAboveInt clamps an integer <x> if it is above the value <high>.
func ClampAboveInt(x, high int) int {
	if x > high {
		return high
	} else {
		return x
	}
}
