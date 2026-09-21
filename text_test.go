package decorender

import (
	"image"
	"image/color"
	"os"
	"testing"
)

// renderTemplate renders an inline layout. Local files resolve from the repo
// root, matching the golden layout's convention.
func renderTemplate(t testing.TB, template string) (*image.RGBA, func()) {
	t.Helper()

	r, err := NewRendererWithTemplate([]byte(template), &Options{LocalFiles: os.DirFS(".")})
	if err != nil {
		t.Fatalf("parsing template: %v", err)
	}

	img, release, err := r.Render(nil, nil)
	if err != nil {
		t.Fatalf("rendering template: %v", err)
	}

	rgba, ok := img.(*image.RGBA)
	if !ok {
		release()
		t.Fatalf("render returned %T, want *image.RGBA", img)
	}

	return rgba, release
}

// mostCoveredPixel finds the pixel in a band that the glyph masks cover most
// fully, which is where the text color shows with the least background mixed in.
func mostCoveredPixel(img *image.RGBA, minY, maxY int, background color.RGBA) (color.RGBA, int, int) {
	var best color.RGBA
	var bestX, bestY int
	furthest := -1

	bounds := img.Bounds()
	for y := minY; y < maxY && y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.RGBAAt(x, y)
			distance := abs(int(c.R)-int(background.R)) +
				abs(int(c.G)-int(background.G)) +
				abs(int(c.B)-int(background.B))
			if distance > furthest {
				furthest, best, bestX, bestY = distance, c, x, y
			}
		}
	}

	return best, bestX, bestY
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// TestTranslucentTextColorMatchesSameColorFill renders text and a filled
// rectangle in one translucent color. Both composite over the same white
// background, so the fully covered pixels of each must agree.
//
// Before font colors were premultiplied, the fill came out correct while the
// text came out very nearly black.
func TestTranslucentTextColorMatchesSameColorFill(t *testing.T) {
	const template = `
width: 420
height: 140
bkgColor: white
padding: 10
innerGap: 8
inner:
  - id: fill
    size: 400 40
    bkgColor: rgba(128, 128, 255, 0.5)
  - text: MMMMMMMMMMMM
    font: Roboto 48 400
    color: rgba(128, 128, 255, 0.5)
`

	img, release := renderTemplate(t, template)
	defer release()

	white := color.RGBA{255, 255, 255, 255}
	fill := img.RGBAAt(200, 30)
	text, x, y := mostCoveredPixel(img, 62, 100, white)

	// 50% of (128,128,255) over white is (191,191,255).
	if fill != (color.RGBA{191, 191, 255, 255}) {
		t.Fatalf("filled rectangle is %v, want {191 191 255 255} - the test's own premise is wrong", fill)
	}

	for _, ch := range []struct {
		name      string
		got, want uint8
	}{
		{"red", text.R, fill.R},
		{"green", text.G, fill.G},
		{"blue", text.B, fill.B},
	} {
		// Antialiasing means no text pixel is quite fully covered, so allow a
		// small gap rather than demanding an exact match.
		if abs(int(ch.got)-int(ch.want)) > 12 {
			t.Errorf("text %s channel is %d at (%d,%d), want about %d (the fill of the same color)",
				ch.name, ch.got, x, y, ch.want)
		}
	}
}
