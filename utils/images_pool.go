package utils

import (
	"image"
	"sync"
)

var imagesBufPool = sync.Pool{
	New: func() any {
		return make([]byte, 4*1000*1000)
	},
}

func NewRGBAImageFromPool(r image.Rectangle) image.RGBA {
	buf := imagesBufPool.Get().([]byte)

	for i := range buf {
		buf[i] = 0x0
	}

	requiredSize := 4 * r.Dx() * r.Dy()
	if cap(buf) < requiredSize {
		buf = append(buf, make([]byte, requiredSize-cap(buf))...)
	}

	return image.RGBA{
		Pix:    buf[0:requiredSize],
		Stride: 4 * r.Dx(),
		Rect:   r,
	}
}

func ReleaseImage(img image.RGBA) {
	imagesBufPool.Put(img.Pix)
}
