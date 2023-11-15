package utils

import "image/color"

var PredefinedColors = []struct {
	Name  string
	Color color.RGBA
}{
	{"black", color.RGBA{0, 0, 0, 255}},
	{"white", color.RGBA{255, 255, 255, 255}},
	{"red", color.RGBA{255, 0, 0, 255}},
	{"lime", color.RGBA{0, 255, 0, 255}},
	{"blue", color.RGBA{0, 0, 255, 255}},
	{"yellow", color.RGBA{255, 255, 0, 255}},
	{"cyan", color.RGBA{0, 255, 255, 255}},
	{"magenta", color.RGBA{255, 0, 255, 255}},
	{"silver", color.RGBA{192, 192, 192, 255}},
	{"gray", color.RGBA{128, 128, 128, 255}},
	{"maroon", color.RGBA{128, 0, 0, 255}},
	{"olive", color.RGBA{128, 128, 0, 255}},
	{"green", color.RGBA{0, 128, 0, 255}},
	{"purple", color.RGBA{128, 0, 128, 255}},
	{"teal", color.RGBA{0, 128, 128, 255}},
	{"navy", color.RGBA{0, 0, 128, 255}},
	{"orange", color.RGBA{255, 165, 0, 255}},
	{"pink", color.RGBA{255, 192, 203, 255}},
	{"coral", color.RGBA{255, 127, 80, 255}},
	{"salmon", color.RGBA{250, 128, 114, 255}},
	{"gold", color.RGBA{255, 215, 0, 255}},
	{"khaki", color.RGBA{240, 230, 140, 255}},
	{"violet", color.RGBA{238, 130, 238, 255}},
	{"plum", color.RGBA{221, 160, 221, 255}},
	{"orchid", color.RGBA{218, 112, 214, 255}},
	{"beige", color.RGBA{245, 245, 220, 255}},
	{"mint", color.RGBA{189, 252, 201, 255}},
	{"lavender", color.RGBA{230, 230, 250, 255}},
	{"ivory", color.RGBA{255, 255, 240, 255}},
	{"azure", color.RGBA{240, 255, 255, 255}},
}
