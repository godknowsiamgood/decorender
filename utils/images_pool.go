package utils

import (
	"golang.org/x/image/draw"
	"image"
	"image/color"
	"sync"
)

var imagesBufPool = sync.Pool{
	New: func() any {
		return make([]byte, 4*500*500)
	},
}

func getImagesBytesBuffer(r image.Rectangle, size int) []byte {
	buf := imagesBufPool.Get().([]byte)

	requiredSize := size * r.Dx() * r.Dy()
	if cap(buf) < requiredSize {
		buf = buf[0:cap(buf)]
		buf = append(buf, make([]byte, requiredSize-cap(buf))...)
	}

	return buf[0:requiredSize]
}

func NewAlphaImageFromPool(w int, h int) *image.Alpha {
	img := image.Alpha{
		Pix:    getImagesBytesBuffer(image.Rect(0, 0, w, h), 1),
		Stride: w,
		Rect:   image.Rect(0, 0, w, h),
	}
	draw.Draw(&img, img.Bounds(), image.NewUniform(color.Alpha{}), image.Point{}, draw.Src)
	return &img
}

func NewRGBAImageFromPool(w int, h int) *image.RGBA {
	img := image.RGBA{
		Pix:    getImagesBytesBuffer(image.Rect(0, 0, w, h), 4),
		Stride: 4 * w,
		Rect:   image.Rect(0, 0, w, h),
	}
	draw.Draw(&img, img.Bounds(), image.NewUniform(color.RGBA{}), image.Point{}, draw.Src)
	return &img
}

func ReleaseImage(img image.Image) {
	switch res := img.(type) {
	case *image.Alpha:
		if len(res.Pix) == 0 {
			return
		}
		res.Pix = res.Pix[0:0]
		imagesBufPool.Put(res.Pix)
	case *image.RGBA:
		if len(res.Pix) == 0 {
			return
		}
		res.Pix = res.Pix[0:0]
		imagesBufPool.Put(res.Pix)
	case *image.NRGBA:
		if len(res.Pix) == 0 {
			return
		}
		res.Pix = res.Pix[0:0]
		imagesBufPool.Put(res.Pix)
	}
}

func UseTempImage(bounds image.Rectangle, cb func(img *image.RGBA)) {
	img := NewRGBAImageFromPool(bounds.Dx(), bounds.Dy())
	cb(img)
	ReleaseImage(img)
}
