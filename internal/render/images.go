package render

import (
	"github.com/godknowsiamgood/decorender/internal/layout"
	"github.com/godknowsiamgood/decorender/internal/utils"
	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
	"image"
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
		mb := mask.Bounds()
		// Indexing Pix directly avoids an interface boxing plus a type assertion
		// per pixel, which dominated this loop.
		for y := mb.Min.Y; y < mb.Max.Y; y++ {
			maskRow := mask.Pix[mask.PixOffset(mb.Min.X, y):]
			srcRow := src.Pix[src.PixOffset(bounds.Min.X, bounds.Min.Y+y-mb.Min.Y):]

			for x := 0; x < mb.Dx(); x++ {
				maskAlpha := maskRow[x]
				srcIdx := x * 4

				existingAlpha := srcRow[srcIdx+3]
				var alphaRatio float64
				if existingAlpha > 0 {
					alphaRatio = float64(maskAlpha) / float64(existingAlpha)
				}

				if alphaRatio < 1 {
					srcRow[srcIdx+0] = uint8(float64(srcRow[srcIdx+0]) * alphaRatio)
					srcRow[srcIdx+1] = uint8(float64(srcRow[srcIdx+1]) * alphaRatio)
					srcRow[srcIdx+2] = uint8(float64(srcRow[srcIdx+2]) * alphaRatio)
				}

				srcRow[srcIdx+3] = minUint8(maskAlpha, existingAlpha)
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

// rotateImage rotates src counter-clockwise by angle degrees onto a new pooled
// image large enough to hold the result, with a transparent background.
//
// This replaces imaging.Rotate, which spawned GOMAXPROCS goroutines per call -
// a large scheduler cost for the small images rotation is typically used on.
func rotateImage(src *image.RGBA, angle float64) *image.RGBA {
	srcW, srcH := src.Bounds().Dx(), src.Bounds().Dy()

	// Transform samples the source and clips at its edge, which leaves the
	// rotated outline hard and jagged. Copying into a one pixel transparent
	// border lets the interpolator fade out across it, which antialiases the
	// edge the way the previous imaging.Rotate did.
	const pad = 1
	padded := utils.NewRGBAImageFromPool(srcW+pad*2, srcH+pad*2)
	defer utils.ReleaseImage(padded)
	draw.Draw(padded, image.Rect(pad, pad, pad+srcW, pad+srcH), src, src.Bounds().Min, draw.Src)

	paddedW, paddedH := srcW+pad*2, srcH+pad*2

	sin, cos := math.Sincos(angle * math.Pi / 180)
	absSin, absCos := math.Abs(sin), math.Abs(cos)

	dstW := int(math.Ceil(float64(paddedW)*absCos + float64(paddedH)*absSin))
	dstH := int(math.Ceil(float64(paddedW)*absSin + float64(paddedH)*absCos))

	dst := utils.NewRGBAImageFromPool(dstW, dstH)

	// Screen coordinates have y pointing down, so a visually counter-clockwise
	// rotation is [cos, sin; -sin, cos]. The translation terms map the centre of
	// the source onto the centre of the (larger) destination.
	srcCx, srcCy := float64(paddedW)/2, float64(paddedH)/2
	dstCx, dstCy := float64(dstW)/2, float64(dstH)/2

	s2d := f64.Aff3{
		cos, sin, dstCx - (cos*srcCx + sin*srcCy),
		-sin, cos, dstCy - (-sin*srcCx + cos*srcCy),
	}

	draw.BiLinear.Transform(dst, s2d, padded, padded.Bounds(), draw.Over, nil)

	return dst
}
