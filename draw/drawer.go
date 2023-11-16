package draw

import (
	"github.com/godknowsiamgood/decorender/fonts"
	"github.com/godknowsiamgood/decorender/utils"
	"image"
	"image/color"
)

type Drawer interface {
	InitImage(width int, height int)
	RetrieveImage() image.Image
	DrawRect(w float64, h float64, color color.Color, border utils.Border, radius utils.FourValues)
	DrawText(text string, fd fonts.FaceDescription, fontColor color.Color)
	GetTextWidth(text string, fd fonts.FaceDescription) float64
	DrawImage(img image.Image)
	SaveState()
	RestoreState()
	SetRotation(deg float64)
	SetTranslation(x float64, y float64)
}
