package draw

import (
	"image"
	"image/color"
	"math"
)

func ipart(x float64) float64 {
	return math.Floor(x)
}

func round(x float64) float64 {
	return ipart(x + 0.5)
}

func fpart(x float64) float64 {
	return x - math.Floor(x)
}

func rfpart(x float64) float64 {
	return 1 - fpart(x)
}

func plot(dst *image.RGBA, x, y int, c color.RGBA, alpha float64) {
	oc := dst.RGBAAt(x, y)

	// Calculate the final alpha value as a combination of the old and new alpha values.
	finalAlpha := uint8(alpha*float64(c.A) + (1-alpha)*float64(oc.A))

	// Blend the new color with the existing color based on alpha.
	dst.SetRGBA(x, y, color.RGBA{
		R: uint8(alpha*float64(c.R) + (1-alpha)*float64(oc.R)),
		G: uint8(alpha*float64(c.G) + (1-alpha)*float64(oc.G)),
		B: uint8(alpha*float64(c.B) + (1-alpha)*float64(oc.B)),
		A: finalAlpha,
	})
}

func drawWuLine(dst *image.RGBA, x0, y0, x1, y1 float64, c color.RGBA) {
	steep := math.Abs(float64(y1-y0)) > math.Abs(float64(x1-x0))

	if steep {
		x0, y0 = y0, x0
		x1, y1 = y1, x1
	}
	if x0 > x1 {
		x0, x1 = x1, x0
		y0, y1 = y1, y0
	}

	dx := float64(x1 - x0)
	dy := float64(y1 - y0)
	gradient := dy / dx
	if dx == 0.0 {
		gradient = 1.0
	}

	xend := round(float64(x0))
	yend := float64(y0) + gradient*(xend-float64(x0))
	xgap := rfpart(float64(x0) + 0.5)
	xpxl1 := int(xend)
	ypxl1 := int(ipart(yend))
	if steep {
		plot(dst, ypxl1, xpxl1, c, rfpart(yend)*xgap)
		plot(dst, ypxl1+1, xpxl1, c, fpart(yend)*xgap)
	} else {
		plot(dst, xpxl1, ypxl1, c, rfpart(yend)*xgap)
		plot(dst, xpxl1, ypxl1+1, c, fpart(yend)*xgap)
	}
	intery := yend + gradient

	xend = round(float64(x1))
	yend = float64(y1) + gradient*(xend-float64(x1))
	xgap = fpart(float64(x1) + 0.5)
	xpxl2 := int(xend)
	ypxl2 := int(ipart(yend))
	if steep {
		plot(dst, ypxl2, xpxl2, c, rfpart(yend)*xgap)
		plot(dst, ypxl2+1, xpxl2, c, fpart(yend)*xgap)
	} else {
		plot(dst, xpxl2, ypxl2, c, rfpart(yend)*xgap)
		plot(dst, xpxl2, ypxl2+1, c, fpart(yend)*xgap)
	}

	if steep {
		for x := xpxl1 + 1; x <= xpxl2-1; x++ {
			plot(dst, int(ipart(intery)), x, c, rfpart(intery))
			plot(dst, int(ipart(intery))+1, x, c, fpart(intery))
			intery = intery + gradient
		}
	} else {
		for x := xpxl1 + 1; x <= xpxl2-1; x++ {
			plot(dst, x, int(ipart(intery)), c, rfpart(intery))
			plot(dst, x, int(ipart(intery))+1, c, fpart(intery))
			intery = intery + gradient
		}
	}
}
