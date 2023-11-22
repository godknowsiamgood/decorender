package utils

import (
	"image"
	"sync"
)

var imagesBufPool = sync.Pool{
	New: func() any {
		return make([]byte, 4*200*200)
	},
}

func getImagesBytesBuffer(r image.Rectangle, size int) []byte {
	buf := imagesBufPool.Get().([]byte)

	requiredSize := size * r.Dx() * r.Dy()
	if cap(buf) < requiredSize {
		buf = buf[0:cap(buf)]
		buf = append(buf, make([]byte, requiredSize-cap(buf))...)
	} else {
		buf = buf[0:requiredSize]
	}

	for i := 0; i < requiredSize; i++ {
		buf[i] = 0
	}

	return buf[0:requiredSize]
}

func NewAlphaImageFromPool(w int, h int) *image.Alpha {
	return &image.Alpha{
		Pix:    getImagesBytesBuffer(image.Rect(0, 0, w, h), 1),
		Stride: w,
		Rect:   image.Rect(0, 0, w, h),
	}
}

func NewRGBAImageFromPool(w int, h int) *image.RGBA {
	return &image.RGBA{
		Pix:    getImagesBytesBuffer(image.Rect(0, 0, w, h), 4),
		Stride: 4 * w,
		Rect:   image.Rect(0, 0, w, h),
	}
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

func UseTempImage(w, h int, cb func(img *image.RGBA)) {
	img := NewRGBAImageFromPool(w, h)
	cb(img)
	ReleaseImage(img)
}
