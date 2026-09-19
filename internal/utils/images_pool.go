package utils

import (
	"image"
	"math/bits"
	"sync"
)

// Image buffers are pooled in power-of-two size classes. A single pool for
// every size does not work here: renders mix a multi-megabyte root image with
// small masks and temporaries, so a shared pool keeps handing out a buffer that
// is too small and each call has to grow it, allocating the full size again.
const (
	minBufClass = 12 // 4 KiB - smaller requests round up to this
	maxBufClass = 28 // 256 MiB - larger requests are not pooled
	numBufPools = maxBufClass - minBufClass + 1
)

var imagesBufPools [numBufPools]sync.Pool

func init() {
	for i := range imagesBufPools {
		size := 1 << (minBufClass + i)
		imagesBufPools[i].New = func() any {
			buf := make([]byte, size)
			return &buf
		}
	}
}

// bufClassFor returns the index of the pool holding buffers of at least n bytes,
// and whether such a pool exists.
func bufClassFor(n int) (int, bool) {
	if n <= 0 {
		return 0, false
	}
	class := bits.Len(uint(n - 1)) // ceil(log2(n))
	if class < minBufClass {
		class = minBufClass
	}
	if class > maxBufClass {
		return 0, false
	}
	return class - minBufClass, true
}

func getImagesBytesBuffer(r image.Rectangle, bytesPerPixel int) []byte {
	required := bytesPerPixel * r.Dx() * r.Dy()
	if required <= 0 {
		return nil
	}

	idx, ok := bufClassFor(required)
	if !ok {
		// Too large to pool; the caller gets a buffer sized exactly to the request.
		return make([]byte, required)
	}

	buf := *imagesBufPools[idx].Get().(*[]byte)
	buf = buf[:required]
	clear(buf) // pooled buffers still hold the previous image
	return buf
}

func putImagesBytesBuffer(buf []byte) {
	capacity := cap(buf)
	if capacity < 1<<minBufClass {
		return
	}

	// Buffers are allocated at exactly 1<<class, so the floor of log2 of the
	// capacity recovers the class they came from.
	class := bits.Len(uint(capacity)) - 1
	if class > maxBufClass {
		class = maxBufClass
	}

	buf = buf[:capacity]
	imagesBufPools[class-minBufClass].Put(&buf)
}

func NewAlphaImageFromPool(w int, h int) *image.Alpha {
	rect := image.Rect(0, 0, w, h)
	return &image.Alpha{
		Pix:    getImagesBytesBuffer(rect, 1),
		Stride: w,
		Rect:   rect,
	}
}

func NewRGBAImageFromPool(w int, h int) *image.RGBA {
	rect := image.Rect(0, 0, w, h)
	return &image.RGBA{
		Pix:    getImagesBytesBuffer(rect, 4),
		Stride: 4 * w,
		Rect:   rect,
	}
}

func ReleaseImage(img image.Image) {
	switch res := img.(type) {
	case *image.Alpha:
		putImagesBytesBuffer(res.Pix)
		res.Pix = nil
	case *image.RGBA:
		putImagesBytesBuffer(res.Pix)
		res.Pix = nil
	case *image.NRGBA:
		putImagesBytesBuffer(res.Pix)
		res.Pix = nil
	}
}

func UseTempImage(bounds image.Rectangle, cb func(img *image.RGBA)) {
	img := NewRGBAImageFromPool(bounds.Dx(), bounds.Dy())
	cb(img)
	ReleaseImage(img)
}
