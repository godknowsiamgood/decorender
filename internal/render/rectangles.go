package render

import (
	"github.com/godknowsiamgood/decorender/internal/utils"
	"golang.org/x/image/draw"
	"golang.org/x/image/vector"
	"image"
	"image/color"
	"math"
	"sync"
)

var rasterizerPool = sync.Pool{
	New: func() any {
		return &vector.Rasterizer{}
	},
}

// image.Uniform is only read for the duration of a draw call, so the sources
// can be pooled instead of allocating one per fill.
var uniformPool = sync.Pool{
	New: func() any {
		return &image.Uniform{}
	},
}

func getUniform(c color.Color) *image.Uniform {
	u := uniformPool.Get().(*image.Uniform)
	u.C = c
	return u
}

// opaqueAlpha is read-only and shared: the rasterizer only samples from it.
var opaqueAlpha = image.NewUniform(color.Alpha{A: 255})

func alphaPremultiply(c color.RGBA) color.RGBA {
	alpha := float64(c.A) / 255
	return color.RGBA{
		R: uint8(float64(c.R) * alpha),
		G: uint8(float64(c.G) * alpha),
		B: uint8(float64(c.B) * alpha),
		A: c.A,
	}
}

func drawRoundedBorder(cache *Cache, dst *image.RGBA, x, y, w, h float64, radii utils.FourValues, border utils.Border) {
	if border.Width < 0.0001 || border.Color.A == 0 {
		return
	}

	outerRadii := radii
	innerRadii := radii

	var outerRect utils.FourValues
	var innerRect utils.FourValues

	if border.Type == utils.BorderTypeOutset {
		if radii.HasValues() {
			for i := range outerRadii {
				outerRadii[i] = math.Max(0, outerRadii[i]+border.Width)
			}
		}

		outerRect = utils.FourValues{0, 0, w + border.Width*2, h + border.Width*2}
		innerRect = utils.FourValues{border.Width, border.Width, w, h}
		x -= border.Width
		y -= border.Width
	} else if border.Type == utils.BorderTypeInset {
		if radii.HasValues() {
			for i := range innerRadii {
				innerRadii[i] = math.Max(0, innerRadii[i]-border.Width)
			}
		}
		outerRect = utils.FourValues{0, 0, w, h}
		innerRect = utils.FourValues{border.Width, border.Width, w - border.Width*2, h - border.Width*2}
	} else {
		if radii.HasValues() {
			for i := range outerRadii {
				outerRadii[i] = math.Max(0, outerRadii[i]+border.Width/2)
			}
			for i := range innerRadii {
				innerRadii[i] = math.Max(0, innerRadii[i]-border.Width/2)
			}
		}
		outerRect = utils.FourValues{0, 0, w + border.Width, h + border.Width}
		innerRect = utils.FourValues{border.Width, border.Width, w - border.Width, h - border.Width}
		x -= border.Width / 2
		y -= border.Width / 2
	}

	outerImage := utils.NewAlphaImageFromPool(int(outerRect[2]), int(outerRect[3]))
	drawRoundedRect(cache, outerImage, color.Alpha{A: 255}, 0, 0, outerRect[2], outerRect[3], outerRadii)

	innerImage := utils.NewAlphaImageFromPool(int(outerRect[2]), int(outerRect[3]))
	drawRoundedRect(cache, innerImage, color.Alpha{A: 255}, innerRect[0], innerRect[1], innerRect[2], innerRect[3], innerRadii)

	bounds := outerImage.Bounds()

	// Both masks share bounds and stride, so the ring is just a saturating
	// subtraction over the raw alpha bytes.
	outerPix, innerPix := outerImage.Pix, innerImage.Pix
	if len(innerPix) < len(outerPix) {
		outerPix = outerPix[:len(innerPix)]
	}
	for i, inner := range innerPix[:len(outerPix)] {
		if outerPix[i] > inner {
			outerPix[i] -= inner
		} else {
			outerPix[i] = 0
		}
	}

	borderUniform := getUniform(alphaPremultiply(border.Color))
	draw.DrawMask(dst, image.Rect(int(x), int(y), int(x)+bounds.Dx(), int(y)+bounds.Dy()), borderUniform, image.Point{}, outerImage, image.Point{}, draw.Over)
	uniformPool.Put(borderUniform)

	utils.ReleaseImage(outerImage)
	utils.ReleaseImage(innerImage)
}

func useRoundedRectMaskImage(cache *Cache, w float64, h float64, radii utils.FourValues, onUse func(mask *image.Alpha)) {
	cache.useRoundedMaskImage(w, h, radii, func(mask *image.Alpha) {
		for i := range radii {
			radii[i] = math.Min(radii[i], math.Min(w/2, h/2))
		}

		x32, y32, w32, h32 := float32(0.0), float32(0.0), float32(w), float32(h)

		r := rasterizerPool.Get().(*vector.Rasterizer)
		r.Reset(int(w), int(h))

		// top left
		rad, rad32 := radii[0], float32(radii[0])
		if rad > 0 {
			r.MoveTo(x32, y32+rad32)
			drawEllipticalArc(rad, rad, rad, rad, radians(180), radians(270), r)
		} else {
			r.MoveTo(x32, y32)
		}

		// top right
		rad, rad32 = radii[1], float32(radii[1])
		if rad > 0 {
			r.LineTo(x32+w32-rad32, y32)
			drawEllipticalArc(w-rad, rad, rad, rad, radians(270), radians(360), r)
		} else {
			r.LineTo(x32+w32, y32)
		}

		// bottom right
		rad, rad32 = radii[2], float32(radii[2])
		if rad > 0 {
			r.LineTo(x32+w32, y32+h32-rad32)
			drawEllipticalArc(w-rad, h-rad, rad, rad, radians(0), radians(90), r)
		} else {
			r.LineTo(x32+w32, y32+h32)
		}

		// bottom left
		rad, rad32 = radii[3], float32(radii[3])
		if rad > 0 {
			r.LineTo(x32+rad32, y32+h32)
			drawEllipticalArc(rad, h-rad, rad, rad, radians(90), radians(180), r)
		} else {
			r.LineTo(x32, y32+h32)
		}

		r.ClosePath()

		r.Draw(mask, mask.Bounds(), opaqueAlpha, image.Point{})

		rasterizerPool.Put(r)
	}, onUse)
}

func drawRoundedRect(cache *Cache, dst draw.Image, c color.Color, x, y, w, h float64, radii utils.FourValues) {
	u := getUniform(c)
	defer uniformPool.Put(u)

	if !radii.HasValues() { // Short path for drawing simple rectangle
		draw.Draw(dst, image.Rect(int(x), int(y), int(x+w), int(y+h)), u, image.Point{}, draw.Over)
		return
	}

	useRoundedRectMaskImage(cache, w, h, radii, func(mask *image.Alpha) {
		bounds := image.Rect(int(x), int(y), int(x+w), int(y+h))
		draw.DrawMask(dst, bounds, u, image.Point{}, mask, image.Point{}, draw.Over)
	})
}

func drawEllipticalArc(x, y, rx, ry, angle1, angle2 float64, path *vector.Rasterizer) {
	const n = 8
	for i := 0; i < n; i++ {
		p1 := float64(i+0) / n
		p2 := float64(i+1) / n
		a1 := angle1 + (angle2-angle1)*p1
		a2 := angle1 + (angle2-angle1)*p2
		x0 := x + rx*math.Cos(a1)
		y0 := y + ry*math.Sin(a1)
		x1 := x + rx*math.Cos((a1+a2)/2)
		y1 := y + ry*math.Sin((a1+a2)/2)
		x2 := x + rx*math.Cos(a2)
		y2 := y + ry*math.Sin(a2)
		cx := 2*x1 - x0/2 - x2/2
		cy := 2*y1 - y0/2 - y2/2
		path.CubeTo(float32(x0), float32(y0), float32(cx), float32(cy), float32(x2), float32(y2))
	}
}

func radians(degrees float64) float64 {
	return degrees * math.Pi / 180
}
