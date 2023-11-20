package draw

import (
	"github.com/fogleman/gg"
	"github.com/godknowsiamgood/decorender/fonts"
	"github.com/godknowsiamgood/decorender/utils"
	"golang.org/x/image/draw"
	"image"
	"image/color"
	_ "image/jpeg"
	"math"
)

type DefaultDrawer struct {
	gc  *gg.Context
	img *image.RGBA
}

func (d *DefaultDrawer) InitImage(width int, height int) {
	d.img = image.NewRGBA(image.Rect(0, 0, width, height))
}

func (d *DefaultDrawer) ReleaseImage() {
	//utils.ReleaseImage(*d.gc.Image().(*image.RGBA))
}

func (d *DefaultDrawer) DrawRect(w float64, h float64, c color.RGBA, border utils.Border, radius utils.FourValues) {
	// Draw filled rect
	if c.A > 0 {
		if border.Width == 0 {
			draw.Draw(d.img, image.Rect(0, 0, int(w), int(h)), &image.Uniform{C: c}, image.Point{}, draw.Src)
		}
	}

	if !radius.HasValues() {
		if border.Width == 0 {
			//draw.Draw(d.img, image.Rect(0, 00, int(w), int(h)), &image.Uniform{C: c}, image.Point{}, draw.Src)
		} else {

			//points := []Point{{100, 80}, {200, 100}, {220, 150}, {200, 200}, {100, 200}}
			//cc := color.RGBA{255, 0, 0, 255} // Red
			//FillPolygon(d.img, points, cc)
			//
			//drawWuLine(d.img, 40, 20, 140, 40, cc)

			//drawAntiAliasedLine(d.img, image.Point{
			//	X: 0,
			//	Y: 0,
			//}, image.Point{
			//	X: 100,
			//	Y: 555,
			//}, c)
			//
			//d := &font.Drawer{
			//	Dst: d.img,
			//	Src: &image.Uniform{color.RGBA{
			//		R: 255,
			//		G: 0,
			//		B: 0,
			//		A: 0,
			//	}},
			//	Face: fonts.GetFontFace(fonts.FaceDescription{
			//		Family: "Roboto",
			//		Size:   50,
			//		Weight: 0,
			//		Style:  0,
			//	}),
			//	Dot: fixed.Point26_6{X: 2000, Y: 8000},
			//}
			//d.DrawString("Hello, World!")

			//drawBorderedRectangle(d.img, w, h, c, border)
		}
	}

	//d.gc.SetColor(c)
	//
	//d.drawRoundedRect(0, 0, w, h, radius)
	//d.gc.Fill()
	//
	//if border.Width > 0 {
	//	d.gc.SetLineWidth(border.Width)
	//	d.gc.SetColor(border.Color)
	//
	//	switch border.Type {
	//	case utils.BorderTypeOutset:
	//		d.drawRoundedRect(-border.Width/2, -border.Width/2, w+border.Width, h+border.Width, increaseRadius(radius, border.Width/2))
	//	case utils.BorderTypeInset:
	//		d.drawRoundedRect(border.Width/2, border.Width/2, w-border.Width, h-border.Width, increaseRadius(radius, -border.Width/2))
	//	case utils.BorderTypeCenter:
	//		d.drawRoundedRect(0, 0, w, h, radius)
	//	}
	//	d.gc.Stroke()
	//}
}

type Point struct {
	X, Y float64
}

func (d *DefaultDrawer) drawRoundedRect(x, y, w, h float64, radius utils.FourValues) {
	radius[0] = min(radius[0], h/2, w/2)
	radius[1] = min(radius[1], h/2, w/2)
	radius[2] = min(radius[2], h/2, w/2)
	radius[3] = min(radius[3], h/2, w/2)

	// Top-left corner
	if radius[0] > 0 {
		d.gc.DrawArc(x+radius[0], y+radius[0], radius[0], -math.Pi, -math.Pi/2)
	} else {
		d.gc.MoveTo(x, y)
	}

	// Top-right corner
	if radius[1] > 0 {
		d.gc.DrawArc(x+w-radius[1], y+radius[1], radius[1], -math.Pi/2, 0)
	} else {
		d.gc.LineTo(x+w, y)
	}

	// Bottom-right corner
	if radius[2] > 0 {
		d.gc.DrawArc(x+w-radius[2], y+h-radius[2], radius[2], 0, math.Pi/2)
	} else {
		d.gc.LineTo(x+w, y+h)
	}

	// Bottom-left corner
	if radius[3] > 0 {
		d.gc.DrawArc(x+radius[3], y+h-radius[3], radius[3], math.Pi/2, math.Pi)
	} else {
		d.gc.LineTo(x, y+h)
	}

	d.gc.ClosePath()
}

func increaseRadius(r utils.FourValues, delta float64) utils.FourValues {
	for i := range r {
		if r[i] > 0 {
			r[i] = max(0, r[i]+delta)
		}
	}
	return r
}

func (d *DefaultDrawer) DrawText(text string, fd fonts.FaceDescription, fontColor color.Color) {
	d.gc.SetFontFace(fonts.GetFontFace(fd))
	d.gc.SetColor(fontColor)
	d.gc.DrawString(text, 0, 0)
}

func (d *DefaultDrawer) GetTextWidth(text string, fd fonts.FaceDescription) float64 {
	d.gc.SetFontFace(fonts.GetFontFace(fd))
	w, _ := d.gc.MeasureString(text)
	return w
}

func (d *DefaultDrawer) RetrieveImage() image.Image {
	return d.img
}

func (d *DefaultDrawer) DrawImage(img image.Image) {
	d.gc.DrawImage(img, 0, 0)
}

func (d *DefaultDrawer) SetTranslation(x float64, y float64) {
	d.gc.Translate(x, y)
}

func (d *DefaultDrawer) SetRotation(deg float64) {
	d.gc.Rotate(deg * 0.01745)
}

func (d *DefaultDrawer) SaveState() {
	d.gc.Push()
}

func (d *DefaultDrawer) RestoreState() {
	d.gc.Pop()
}

func drawBorderedRectangle(dst draw.Image, w, h float64, rectColor color.Color, border utils.Border) {
	return
	//width, height := 200, 200
	//img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Define the colors with alpha for semi-transparency
	red := color.RGBA{255, 0, 0, 255}   // Semi-transparent red
	green := color.RGBA{0, 255, 0, 255} // Semi-transparent green

	// Draw the first rectangle (red)
	draw.Draw(dst, image.Rect(20, 20, 120, 120), &image.Uniform{red}, image.Point{}, draw.Over)

	// Draw the second rectangle (green), overlapping the first
	//draw.Draw(dst, image.Rect(80, 80, 180, 180), &image.Uniform{green}, image.Point{}, draw.Src)
	draw.DrawMask(dst, image.Rect(80, 80, 180, 180), &image.Uniform{green}, image.Point{}, image.NewUniform(color.RGBA{
		R: 0,
		G: 0,
		B: 0,
		A: 100,
	}), image.Point{}, draw.Over)

	draw.DrawMask(dst, image.Rect(100, 100, 200, 200), &image.Uniform{green}, image.Point{}, image.NewUniform(color.RGBA{
		R: 0,
		G: 0,
		B: 0,
		A: 100,
	}), image.Point{}, draw.Over)
	//
	//var totalWidth, totalHeight int
	//var filledXOffset, filledYOffset int
	//
	//if border.Type == utils.BorderTypeCenter {
	//	totalWidth = int(w + border.Width)
	//	totalHeight = int(h + border.Width)
	//} else if border.Type == utils.BorderTypeOutset {
	//	totalWidth = int(w + 2*border.Width)
	//	totalHeight = int(h + 2*border.Width)
	//} else { // BorderTypeInset
	//	totalWidth = int(w)
	//	totalHeight = int(h)
	//}
	//
	//rectImg := image.NewRGBA(image.Rect(0, 0, totalWidth, totalHeight))
	//
	//// Draw the filled rectangle
	//draw.Draw(rectImg, image.Rect(filledXOffset, filledXOffset, totalWidth-filledXOffset, totalHeight-filledYOffset), &image.Uniform{C: rectColor}, image.Point{}, draw.Src)
	//
	//borderColor := &image.Uniform{C: border.Color}
	//
	//borderWidth := int(border.Width)
	//// Top border
	//draw.Draw(rectImg, image.Rect(0, 0, totalWidth, borderWidth), image.NewUniform(color.Alpha{128}), image.Point{}, draw.Src)
	//// Bottom border
	//draw.Draw(rectImg, image.Rect(0, totalHeight-borderWidth, totalWidth, totalHeight), borderColor, image.Point{}, draw.Src)
	//// Left border
	//draw.Draw(rectImg, image.Rect(0, borderWidth, borderWidth, totalHeight-borderWidth), borderColor, image.Point{}, draw.Src)
	//// Right border
	//draw.Draw(rectImg, image.Rect(totalWidth-borderWidth, borderWidth, totalWidth, totalHeight-borderWidth), borderColor, image.Point{}, draw.Src)
	//
	//draw.Draw(dst, image.Rect(0, 0, totalWidth, totalHeight), rectImg, image.Point{}, draw.Over)
}

func premultiplyColor(c color.RGBA) color.RGBA {
	a := c.A
	r := uint8(uint16(c.R) * uint16(a) / 255)
	g := uint8(uint16(c.G) * uint16(a) / 255)
	b := uint8(uint16(c.B) * uint16(a) / 255)
	return color.RGBA{R: r, G: g, B: b, A: a}
}

func drawAntiAliasedLine(img draw.Image, start, end image.Point, fill color.Color) {
	x0, x1 := start.X, end.X
	y0, y1 := start.Y, end.Y
	Δx := math.Abs(float64(x1 - x0))
	Δy := math.Abs(float64(y1 - y0))
	if Δx >= Δy { // shallow slope
		if x0 > x1 {
			x0, y0, x1, y1 = x1, y1, x0, y0
		}
		y := y0
		yStep := 1
		if y0 > y1 {
			yStep = -1
		}
		remainder := float64(int(Δx/2)) - Δx
		for x := x0; x <= x1; x++ {
			img.Set(x, y, fill)
			remainder += Δy
			if remainder >= 0.0 {
				remainder -= Δx
				y += yStep
			}
		}
	} else { // steep slope
		if y0 > y1 {
			x0, y0, x1, y1 = x1, y1, x0, y0
		}
		x := x0
		xStep := 1
		if x0 > x1 {
			xStep = -1
		}
		remainder := float64(int(Δy/2)) - Δy
		for y := y0; y <= y1; y++ {
			img.Set(x, y, fill)
			remainder += Δx
			if remainder >= 0.0 {
				remainder -= Δy
				x += xStep
			}
		}
	}
}
