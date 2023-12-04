package render

import (
	"github.com/godknowsiamgood/decorender/layout"
	"github.com/godknowsiamgood/decorender/utils"
	"golang.org/x/image/draw"
	"image"
	"image/color"
	"math"
)

func scaleAndCropImage(src image.Image, w, h float64, sizeType layout.BkgImageSizeType) *image.RGBA {
	imgWidth, imgHeight := src.Bounds().Dx(), src.Bounds().Dy()
	dstAspectRatio := w / h
	srcAspectRatio := float64(imgWidth) / float64(imgHeight)

	var scaledImg *image.RGBA

	if sizeType == layout.BkgImageSizeContain {
		var scaleFactor float64
		if srcAspectRatio > dstAspectRatio {
			scaleFactor = w / float64(imgWidth)
		} else {
			scaleFactor = h / float64(imgHeight)
		}
		newWidth := float64(imgWidth) * scaleFactor
		newHeight := float64(imgHeight) * scaleFactor

		scaledImg = utils.NewRGBAImageFromPool(int(w), int(h))

		dstRect := image.Rect(int((w-newWidth)/2), int((h-newHeight)/2), int((w+newWidth)/2), int((h+newHeight)/2))
		draw.BiLinear.Scale(scaledImg, dstRect, src, src.Bounds(), draw.Over, nil)
	} else {
		srcX, srcY, srcW, srcH := 0, 0, imgWidth, imgHeight

		if srcAspectRatio > dstAspectRatio {
			srcW = int(math.Round(float64(srcH) * dstAspectRatio))
			srcX = (imgWidth - srcW) / 2
		} else {
			srcH = int(math.Round(float64(srcW) / dstAspectRatio))
			srcY = (imgHeight - srcH) / 2
		}

		scaledImg = utils.NewRGBAImageFromPool(int(w), int(h))
		draw.BiLinear.Scale(scaledImg, scaledImg.Bounds(), src, image.Rect(srcX, srcY, srcX+srcW, srcY+srcH), draw.Over, nil)
	}

	return scaledImg
}

func applyBorderRadius(cache *Cache, src *image.RGBA, radii utils.FourValues) {
	if !radii.HasValues() {
		return
	}

	bounds := src.Bounds()

	useRoundedRectMaskImage(cache, float64(bounds.Dx()), float64(bounds.Dy()), radii, func(mask *image.Alpha) {
		bounds = mask.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				maskAlpha := mask.At(x, y).(color.Alpha).A
				srcIdx := src.PixOffset(x, y)

				existingAlpha := src.Pix[srcIdx+3]
				var alphaRatio float64
				if existingAlpha > 0 {
					alphaRatio = float64(maskAlpha) / float64(existingAlpha)
				} else {
					alphaRatio = 0
				}

				if alphaRatio < 1 {
					for c := 0; c < 3; c++ {
						src.Pix[srcIdx+c] = uint8(float64(src.Pix[srcIdx+c]) * alphaRatio)
					}
				}

				src.Pix[srcIdx+3] = minUint8(maskAlpha, existingAlpha)
			}
		}
	})
}

func minUint8(a, b uint8) uint8 {
	if a < b {
		return a
	} else {
		return b
	}
}

func copyImage(dst draw.Image, src image.Image) {
	draw.Draw(dst, dst.Bounds(), src, image.Point{}, draw.Src)
}
