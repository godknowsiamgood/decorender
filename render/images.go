package render

import (
	"bytes"
	"fmt"
	"github.com/godknowsiamgood/decorender/layout"
	"github.com/godknowsiamgood/decorender/resources"
	"github.com/godknowsiamgood/decorender/utils"
	"golang.org/x/image/draw"
	"image"
	"image/color"
	"math"
	"strconv"
	"time"
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

	mask := utils.NewAlphaImageFromPool(bounds.Dx(), bounds.Dy())

	drawRoundedRect(cache, mask, color.Alpha{A: 255}, 0, 0, float64(bounds.Dx()), float64(bounds.Dy()), radii)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			offset := src.PixOffset(x, y)
			alpha := mask.AlphaAt(x, y).A
			r, g, b, a := blendColors(0, 0, 0, alpha, src.Pix[offset+0], src.Pix[offset+1], src.Pix[offset+2], src.Pix[offset+3])
			src.Pix[offset+0] = r
			src.Pix[offset+1] = g
			src.Pix[offset+2] = b
			src.Pix[offset+3] = a
		}
	}

	utils.ReleaseImage(mask)
}

func getScaledImage(cache *Cache, fileName string, w, h float64, sizeType layout.BkgImageSizeType) (image.Image, error) {
	keyBuilder := cache.keysBuildersPool.Get()
	keyBuilder.WriteString(fileName)
	keyBuilder.WriteString(strconv.Itoa(int(w)))
	keyBuilder.WriteString("/")
	keyBuilder.WriteString(strconv.Itoa(int(h)))
	keyBuilder.WriteString("/")
	keyBuilder.WriteString(strconv.Itoa(int(sizeType)))
	key := keyBuilder.String()
	cache.keysBuildersPool.Release(keyBuilder)

	if cache.scaledResourceImages.Has(key) {
		v, _ := cache.scaledResourceImages.Get(key)
		img, _ := v.(image.Image)
		if img == nil {
			return nil, fmt.Errorf("cant get image %v from cache", fileName)
		}
		return img, nil
	} else {
		imageBytes, err := resources.GetResourceContent(fileName)
		if err != nil {
			return nil, fmt.Errorf("cant get image %v: %w", fileName, err)
		}

		imgReader := bytes.NewReader(imageBytes)
		srcImg, _, err := image.Decode(imgReader)
		if err != nil {
			return nil, fmt.Errorf("cant decode image %v: %w", fileName, err)
		}

		var scaledAndCroppedImage image.Image
		if srcImg.Bounds().Dx() == int(w) && srcImg.Bounds().Dy() == int(h) {
			scaledAndCroppedImage = srcImg
		} else {
			scaledAndCroppedImage = scaleAndCropImage(srcImg, w, h, sizeType)
		}
		_ = cache.scaledResourceImages.SetWithExpire(key, scaledAndCroppedImage, time.Minute*5)

		return scaledAndCroppedImage, nil
	}
}

func copyImage(dst draw.Image, src image.Image) {
	draw.Draw(dst, dst.Bounds(), src, image.Point{}, draw.Src)
}

func blendColors(sr, sg, sb, sa, dr, dg, db, da uint8) (uint8, uint8, uint8, uint8) {
	srcAlpha := float64(sa) / 255
	dstAlpha := float64(da) / 255

	finalAlpha := srcAlpha + dstAlpha*(1-srcAlpha)
	if finalAlpha == 0 {
		return 0, 0, 0, 0
	}

	r := (float64(sr)*srcAlpha + float64(dr)*dstAlpha*(1-srcAlpha)) / finalAlpha
	g := (float64(sg)*srcAlpha + float64(dg)*dstAlpha*(1-srcAlpha)) / finalAlpha
	b := (float64(sb)*srcAlpha + float64(db)*dstAlpha*(1-srcAlpha)) / finalAlpha

	a := uint8(finalAlpha * 255)

	r *= finalAlpha
	g *= finalAlpha
	b *= finalAlpha

	return uint8(r), uint8(g), uint8(b), a
}
