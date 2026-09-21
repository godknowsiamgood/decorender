package decorender

import (
	"image"
	"image/color"
	"os"
	"strings"
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

// lowestInkRow reports the bottom-most row within a column range that carries
// any ink, which for baseline-resting glyphs is the baseline itself.
func lowestInkRow(img *image.RGBA, minX, maxX, minY, maxY int, background color.RGBA) int {
	lowest := -1

	bounds := img.Bounds()
	for y := minY; y < maxY && y < bounds.Max.Y; y++ {
		for x := minX; x < maxX && x < bounds.Max.X; x++ {
			c := img.RGBAAt(x, y)
			if c != background {
				lowest = y
				break
			}
		}
	}

	return lowest
}

// TestColonSitsOnTheBaseline draws a colon and a full stop, which rest on the
// baseline together in every normal font, and checks that the renderer puts
// them there together.
//
// Colons used to be lifted off the baseline by a hand-tuned fraction of the
// font size, attributed to the lack of a shaping engine. Vertical placement is
// not a shaping concern, and the shift only moved a correct glyph out of place.
func TestColonSitsOnTheBaseline(t *testing.T) {
	// A size large enough that the old shift (3*size/44, rounded down) was
	// several pixels rather than none.
	const template = `
width: 300
height: 140
bkgColor: white
color: black
padding: 20
inner:
  - text: ".:."
    font: Roboto 88 400
`

	img, release := renderTemplate(t, template)
	defer release()

	white := color.RGBA{255, 255, 255, 255}

	// ".:." is three glyphs of roughly equal advance, so thirds of the inked
	// width separate them well enough to measure each one alone.
	inkStart, inkEnd := -1, -1
	for x := 0; x < img.Bounds().Max.X; x++ {
		if lowestInkRow(img, x, x+1, 0, img.Bounds().Max.Y, white) >= 0 {
			if inkStart < 0 {
				inkStart = x
			}
			inkEnd = x
		}
	}
	if inkStart < 0 {
		t.Fatal("nothing was drawn")
	}

	third := (inkEnd + 1 - inkStart) / 3
	stop := lowestInkRow(img, inkStart, inkStart+third, 0, img.Bounds().Max.Y, white)
	colon := lowestInkRow(img, inkStart+third, inkStart+2*third, 0, img.Bounds().Max.Y, white)

	if stop < 0 || colon < 0 {
		t.Fatalf("could not find both glyphs: full stop bottom %d, colon bottom %d", stop, colon)
	}

	// Both glyphs end on the baseline, so their lowest inked rows coincide.
	// One row of slack covers antialiasing at the very edge.
	if abs(colon-stop) > 1 {
		t.Errorf("colon bottom is row %d and full stop bottom is row %d; they should share the baseline", colon, stop)
	}
}

// textNodeWidth renders one unwrapped line of text and reports the width the
// layout gave it.
func textNodeWidth(t testing.TB, text string) int {
	t.Helper()

	template := "bkgColor: white\ncolor: black\npadding: 0\ninner:\n" +
		"  - innerDirection: row\n    innerWrap: none\n    font: Roboto 40 400\n" +
		"    text: \"" + text + "\"\n"

	img, release := renderTemplate(t, template)
	defer release()

	return img.Bounds().Dx()
}

// TestZeroWidthBreakTakesNoWidth checks that a zero-width space is what its
// name says: a place the line may break, drawn as nothing.
//
// It used to be rewritten to an ordinary space when drawing but measured as
// the original rune, so the text was laid out at one width and painted at
// another.
func TestZeroWidthBreakTakesNoWidth(t *testing.T) {
	plain := textNodeWidth(t, "abcd")

	for _, r := range []struct{ name, text string }{
		{"zero width space", "ab​cd"},
		{"mongolian vowel separator", "ab᠎cd"},
	} {
		if got := textNodeWidth(t, r.text); got != plain {
			t.Errorf("%s: text measures %dpx, want %dpx - the same as %q without it",
				r.name, got, plain, "abcd")
		}
	}

	// It is still a break opportunity: given room for only one half, the text
	// wraps there and the node is no wider than the wider half.
	const wrapping = `
width: 60
bkgColor: white
color: black
padding: 0
inner:
  - innerDirection: row
    innerWrap: wrap
    font: Roboto 40 400
    text: "ab` + "​" + `cd"
`
	img, release := renderTemplate(t, wrapping)
	defer release()

	if h := img.Bounds().Dy(); h < 80 {
		t.Errorf("text is %dpx tall, want two lines of 48px - it did not break at the zero-width space", h)
	}
}

// TestTrailingJoinIsNotChargedAWhitespace renders a line ending in a hyphen.
// Whitespace falls between tokens, so the last one on a line is followed by
// none, whether or not it is the kind of token the next one joins.
//
// The row width used to subtract a whitespace for every joining token in the
// row, including the last, leaving such a line one space too narrow for the
// text actually drawn into it.
func TestTrailingJoinIsNotChargedAWhitespace(t *testing.T) {
	for _, c := range []struct{ withJoin, without string }{
		{"abc-", "abc"},
		{"a b-", "a b"},
		{"ab​", "ab"},
	} {
		got := textNodeWidth(t, c.withJoin)
		bare := textNodeWidth(t, c.without)

		// A trailing hyphen adds its own advance; a trailing zero-width break
		// adds nothing. Neither may make the line narrower than the text
		// without it.
		if got < bare {
			t.Errorf("%q measures %dpx, narrower than %q at %dpx", c.withJoin, got, c.without, bare)
		}
	}
}

// TestMissingFontFaceIsReported checks that a layout naming a font it never
// declared says so.
//
// Measuring used to report zero width for a font it could not resolve. Every
// word then measured as nothing, a content-sized layout collapsed to nothing,
// and the render failed with NothingToRenderErr without mentioning the font.
// The real message appeared only when some unrelated sibling happened to carry
// an explicit size, which kept the layout large enough to reach the point
// where drawing reports the same failure.
func TestMissingFontFaceIsReported(t *testing.T) {
	templates := map[string]string{
		"text alone": `
bkgColor: white
inner:
  - text: hello world
    font: Nope 20 400
`,
		"text beside a sized box": `
bkgColor: white
inner:
  - size: 50 50
    bkgColor: red
  - text: hello world
    font: Nope 20 400
`,
	}

	for name, template := range templates {
		t.Run(name, func(t *testing.T) {
			r, err := NewRendererWithTemplate([]byte(template), &Options{LocalFiles: os.DirFS(".")})
			if err != nil {
				t.Fatalf("parsing template: %v", err)
			}

			_, release, err := r.Render(nil, nil)
			if err == nil {
				release()
				t.Fatal("rendering with an undeclared font face succeeded, want an error naming the face")
			}

			if !strings.Contains(err.Error(), "Nope") {
				t.Errorf("error is %q, want it to name the missing face Nope", err)
			}
		})
	}
}

// BenchmarkTextRender exercises a page of wrapped body text.
//
// The golden benchmark spends most of its time scaling images, so it barely
// moves when text rendering changes; this one is almost entirely text.
func BenchmarkTextRender(b *testing.B) {
	const template = `
width: 800
bkgColor: white
color: 0x222222
font: Roboto 16 400
padding: 20
inner:
  - innerDirection: row
    innerWrap: wrap
    text: ~ text
`

	r, err := NewRendererWithTemplate([]byte(template), &Options{LocalFiles: os.DirFS(".")})
	if err != nil {
		b.Fatalf("parsing template: %v", err)
	}

	words := strings.Fields("the quick brown fox jumps over a lazy dog while " +
		"typographic waves of Avast Wonder Yearn Toward Various Anchored " +
		"Layouts keep flowing onward")

	var sb strings.Builder
	for i := 0; i < 40; i++ {
		sb.WriteString(words[i%len(words)])
		sb.WriteByte(' ')
	}
	data := map[string]any{"text": sb.String()}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, release, err := r.Render(data, nil)
		if err != nil {
			b.Fatalf("rendering: %v", err)
		}
		release()
	}
}
