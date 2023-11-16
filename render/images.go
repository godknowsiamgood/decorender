package render

import (
	draw2 "github.com/godknowsiamgood/decorender/draw"
	"github.com/godknowsiamgood/decorender/utils"
	"golang.org/x/image/draw"
	"image"
	"image/color"
	"math"
)

func scaleAndCropImage(src image.Image, w, h float64, isContains bool) image.Image {
	imgWidth, imgHeight := src.Bounds().Dx(), src.Bounds().Dy()
	dstAspectRatio := w / h
	srcAspectRatio := float64(imgWidth) / float64(imgHeight)

	var scaledImg image.Image

	if isContains {
		var scaleFactor float64
		if srcAspectRatio > dstAspectRatio {
			scaleFactor = w / float64(imgWidth)
		} else {
			scaleFactor = h / float64(imgHeight)
		}
		newWidth := float64(imgWidth) * scaleFactor
		newHeight := float64(imgHeight) * scaleFactor

		scaledAndCropped := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
		dstRect := image.Rect(int((w-newWidth)/2), int((h-newHeight)/2), int((w+newWidth)/2), int((h+newHeight)/2))
		draw.CatmullRom.Scale(scaledAndCropped, dstRect, src, src.Bounds(), draw.Over, nil)

		scaledImg = scaledAndCropped
	} else {
		srcX, srcY, srcW, srcH := 0, 0, imgWidth, imgHeight

		if srcAspectRatio > dstAspectRatio {
			srcW = int(math.Round(float64(srcH) * dstAspectRatio))
			srcX = (imgWidth - srcW) / 2
		} else {
			srcH = int(math.Round(float64(srcW) / dstAspectRatio))
			srcY = (imgHeight - srcH) / 2
		}

		scaledAndCropped := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
		draw.CatmullRom.Scale(scaledAndCropped, scaledAndCropped.Bounds(), src, image.Rect(srcX, srcY, srcX+srcW, srcY+srcH), draw.Over, nil)

		scaledImg = scaledAndCropped
	}

	return scaledImg
}

func applyBorderRadius(src image.Image, radius utils.FourValues) image.Image {
	bounds := src.Bounds()
	d := draw2.DefaultDrawer{}
	d.InitImage(bounds.Size().X, bounds.Size().Y)

	d.DrawRect(float64(src.Bounds().Dx()), float64(src.Bounds().Dy()), color.White, utils.Border{}, radius)
	mask := d.RetrieveImage()

	dst := image.NewRGBA(bounds)
	draw.DrawMask(dst, src.Bounds(), src, image.Point{}, mask, image.Point{}, draw.Over)

	return dst
}
