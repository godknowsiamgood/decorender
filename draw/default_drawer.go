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

func (d *DefaultDrawer) DrawRect(w float64, h float64, c color.Color, radius utils.FourValues) {
	d.gc.SetColor(c)

	// Top-left corner
	if radius[0] > 0 {
		d.gc.DrawArc(0+radius[0], 0+radius[0], radius[0], -math.Pi, -math.Pi/2)
	} else {
		d.gc.MoveTo(0, 0)
	}

	// Top-right corner
	if radius[1] > 0 {
		d.gc.DrawArc(w-radius[1], 0+radius[1], radius[1], -math.Pi/2, 0)
	} else {
		d.gc.LineTo(w, 0)
	}

	// Bottom-right corner
	if radius[2] > 0 {
		d.gc.DrawArc(w-radius[2], h-radius[2], radius[2], 0, math.Pi/2)
	} else {
		d.gc.LineTo(w, h)
	}

	// Bottom-left corner
	if radius[3] > 0 {
		d.gc.DrawArc(0+radius[3], h-radius[3], radius[3], math.Pi/2, math.Pi)
	} else {
		d.gc.LineTo(0, h)
	}

	d.gc.Fill()
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
