| ![Original image](lenna.png) | 
|:--:| 
| *Original image* |

| ![Dithered image](lenna_dit.png) | 
|:--:| 
| *Dithered image using only four colors* |

# Important note!
This program works for PNG files, at least. However it does not work with JPEG files.
**Pro tip!** If you're on Windows you can use MS Paint to convert your images to PNG.

# What is this program?
This Go program transforms an image by applying the Bayer dithering algorithm. (https://en.wikipedia.org/wiki/Ordered_dithering)

# How do I run this program?
Type this following line in your console.

```
go run main.go -in=lenna.png -out=lenna_dit.png -pal=4==
```

This command creates a dithered image using four colors.

These are the available flags:
- **in**:   filepath of the input image
- **ou**:  filepath of the output image
- **pal**:  palette maximum size i.e. the maximum number of colors to use in the output image.

# TODO
 - Allow users to choose the size of the Bayer dithering matrix: 2, 4 or 8.
 - There is definitely a problem whith JPEG/JPG images. The resulting images are usually a very unappealing soup of pixels, sadly.