# How to use
go run main.go -in=image.png -out=image_out.png -pal=4

This command creates a dithered image using four colors.

# TODO
 - Allow users to choose the size of the Bayer dithering matrix: 2, 4 or 8.
 - There is definitely a problem whith JPEG/JPG images. The resulting images are usually a very unappealing soup of pixels.