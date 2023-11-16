package draw

import (
	"github.com/fogleman/gg"
	"github.com/godknowsiamgood/decorender/fonts"
	"github.com/godknowsiamgood/decorender/utils"
	"image"
	"image/color"
	_ "image/jpeg"
	"math"
)

type DefaultDrawer struct {
	gc *gg.Context
}

func (d *DefaultDrawer) InitImage(width int, height int) {
	d.gc = gg.NewContext(width, height)
}

func (d *DefaultDrawer) DrawRect(w float64, h float64, c color.Color, border utils.Border, radius utils.FourValues) {
	d.gc.SetColor(c)

	d.drawRoundedRect(0, 0, w, h, radius)
	d.gc.Fill()

	if border.Width > 0 {
		d.gc.SetLineWidth(border.Width)
		d.gc.SetColor(border.Color)

		switch border.Type {
		case utils.BorderTypeOutset:
			d.drawRoundedRect(-border.Width/2, -border.Width/2, w+border.Width, h+border.Width, increaseRadius(radius, border.Width/2))
		case utils.BorderTypeInset:
			d.drawRoundedRect(border.Width/2, border.Width/2, w-border.Width, h-border.Width, increaseRadius(radius, -border.Width/2))
		case utils.BorderTypeCenter:
			d.drawRoundedRect(0, 0, w, h, radius)
		}
		d.gc.Stroke()
	}
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
	return d.gc.Image()
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
