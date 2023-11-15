package render

import (
	draw2 "decorender/draw"
	"decorender/utils"
	"golang.org/x/image/draw"
	"image"
	"image/color"
	"math"
)

func scaleAndCropImage(src image.Image, w, h float64) image.Image {
	dstAspectRatio := w / h
	srcAspectRatio := float64(src.Bounds().Dx()) / float64(src.Bounds().Dy())

	srcX, srcY, srcW, srcH := 0, 0, src.Bounds().Dx(), src.Bounds().Dy()

	if srcAspectRatio > dstAspectRatio {
		// Source is wider than destination
		srcW = int(math.Round(float64(srcH) * dstAspectRatio))
		srcX = (src.Bounds().Dx() - srcW) / 2
	} else {
		// Source is taller than destination
		srcH = int(math.Round(float64(srcW) / dstAspectRatio))
		srcY = (src.Bounds().Dy() - srcH) / 2
	}

	scaledAndCropped := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
	draw.CatmullRom.Scale(scaledAndCropped, scaledAndCropped.Bounds(), src, image.Rect(srcX, srcY, srcX+srcW, srcY+srcH), draw.Over, nil)

	return scaledAndCropped
}

func applyBorderRadius(src image.Image, radius utils.FourValues) image.Image {
	bounds := src.Bounds()
	d := draw2.DefaultDrawer{}
	d.InitImage(bounds.Size().X, bounds.Size().Y)

	d.DrawRect(float64(src.Bounds().Dx()), float64(src.Bounds().Dy()), color.White, radius)
	mask := d.RetrieveImage()

	dst := image.NewRGBA(bounds)
	draw.DrawMask(dst, src.Bounds(), src, image.Point{}, mask, image.Point{}, draw.Over)

	return dst
}
