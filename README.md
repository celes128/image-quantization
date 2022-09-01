# What is this program?
This Go program transforms an image by applying the Bayer dithering algorithm. (https://en.wikipedia.org/wiki/Ordered_dithering)

# How do I run this program?
Type this following line in your console.
==go run main.go -in=image.png -out=image_out.png -pal=4==

This command creates a dithered image using four colors.

These are the available flags:
    - in:   filepath of the input image
    - out:   filepath of the output image
    - pal:  the maximum number of colors to use in the output image (pal is short for palette).

# TODO
 - Allow users to choose the size of the Bayer dithering matrix: 2, 4 or 8.
 - There is definitely a problem whith JPEG/JPG images. The resulting images are usually a very unappealing soup of pixels.