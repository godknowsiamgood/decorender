package decorender

import (
	"image"
	"image/color"
	"testing"
)

// renderLayout lays out a template and hands back the image, already released by the
// time the test reads it - the pixels are copied out first.
func renderLayout(t *testing.T, layout string) image.Image {
	t.Helper()

	r, err := NewRendererWithTemplate([]byte(layout), nil)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	img, release, err := r.Render(nil, &RenderOptions{UseSample: true})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	defer release()

	copied := image.NewRGBA(img.Bounds())
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			copied.Set(x, y, img.At(x, y))
		}
	}
	return copied
}

func rgb(c color.Color) (uint8, uint8, uint8) {
	r, g, b, _ := c.RGBA()
	return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)
}

// A row is as tall as its tallest child. Measuring it by the first child
// instead made a container shorter than what it draws, and everything past
// that height was clipped - a legend that wrapped onto a second line simply
// disappeared.
func TestRowHeightFollowsTallestChild(t *testing.T) {
	cases := []struct {
		name   string
		layout string
	}{
		{
			name: "tallest child last",
			layout: `width: 100
bkgColor: white
innerDirection: row
inner:
  - size: 20 20
    bkgColor: 0x0000ff
  - size: 20 60
    bkgColor: 0x00ff00
`,
		},
		{
			name: "tallest child first",
			layout: `width: 100
bkgColor: white
innerDirection: row
inner:
  - size: 20 60
    bkgColor: 0x00ff00
  - size: 20 20
    bkgColor: 0x0000ff
`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			img := renderLayout(t, c.layout)
			if got := img.Bounds().Dy(); got != 60 {
				t.Fatalf("row height = %v, want 60", got)
			}
		})
	}
}

// Wrapped rows stack by their own heights, so a container around them has to
// be tall enough for every line.
func TestWrappedRowsAreNotClipped(t *testing.T) {
	// Three 40 wide children in a 100 wide row: two on the first line, one on
	// the second. The tall one is last, so it only fits if both the wrap and
	// the row height are measured properly.
	const layout = `width: 100
bkgColor: white
innerDirection: row
innerWrap: wrap
inner:
  - size: 40 20
    bkgColor: 0x0000ff
  - size: 40 20
    bkgColor: 0x0000ff
  - size: 40 50
    bkgColor: 0x00ff00
`

	img := renderLayout(t, layout)
	if got := img.Bounds().Dy(); got != 70 {
		t.Fatalf("two rows of 20 and 50 measured %v, want 70", got)
	}

	// The wrapped child is drawn, not cut off at the edge of the container.
	if r, g, b := rgb(img.At(20, 65)); r != 0 || g != 255 || b != 0 {
		t.Errorf("wrapped child missing at its last row: got %v %v %v", r, g, b)
	}
}

// innerRowAlign moves children across the row, the way innerColumnAlign moves
// them across a column.
func TestInnerRowAlign(t *testing.T) {
	cases := []struct {
		align string
		top   int // where the short child is expected to start
	}{
		{align: "", top: 0}, // absent: children sit on the top edge
		{align: "top", top: 0},
		{align: "center", top: 20},
		{align: "bottom", top: 40},
	}

	for _, c := range cases {
		name := c.align
		if name == "" {
			name = "absent"
		}

		t.Run(name, func(t *testing.T) {
			layout := "width: 100\nbkgColor: white\ninnerDirection: row\n"
			if c.align != "" {
				layout += "innerRowAlign: " + c.align + "\n"
			}
			layout += `inner:
  - size: 20 60
    bkgColor: 0x00ff00
  - size: 20 20
    bkgColor: 0x0000ff
`

			img := renderLayout(t, layout)

			// The short child spans x 20..40; find where its blue starts.
			top := -1
			for y := 0; y < img.Bounds().Dy(); y++ {
				if r, g, b := rgb(img.At(30, y)); r == 0 && g == 0 && b == 255 {
					top = y
					break
				}
			}
			if top != c.top {
				t.Errorf("innerRowAlign %q put the child at %v, want %v", c.align, top, c.top)
			}
		})
	}
}

// A property the layout got wrong is reported rather than swallowed. Every
// property but color used to fall back to a default, so a misquoted anchor
// turned an absolutely positioned node back into a node in the flow with no
// diagnostic at all.
func TestBrokenExpressionIsReported(t *testing.T) {
	cases := []struct {
		name   string
		layout string
	}{
		{
			// The quotes belong around the whole expression. Written this way
			// they reach expr as the quotes of a string literal, so it happily
			// evaluates to the text of the expression, and `absolute` is left
			// holding a direction it cannot read.
			name: "anchor whose quotes are in the wrong place",
			layout: `size: 100 100
sample:
  y: 10
inner:
  - size: 10 10
    bkgColor: salmon
    absolute: ~ '"top/" + string(y)'
`,
		},
		{
			name: "size reading a field that is not there",
			layout: `size: 100 100
sample:
  w: 10
inner:
  - width: ~ missing.field
    height: 10
    bkgColor: salmon
`,
		},
		{
			name: "text of a node",
			layout: `size: 100 100
sample:
  title: hello
inner:
  - text: ~ title +
`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := NewRendererWithTemplate([]byte(c.layout), nil)
			if err != nil {
				return // a parse error is a report too
			}
			if err := r.RenderAndWrite(nil, EncodeFormatNone, nil, &RenderOptions{UseSample: true}); err == nil {
				t.Error("a broken expression rendered without an error")
			}
		})
	}
}

// A border drawn inside a node thinner than twice its own width has nothing
// left to cut out of the middle. The empty inner rectangle used to come back
// from image.Rect the right way round, covering the whole node, so the border
// vanished - which is every horizontal rule, the thinnest node there is.
func TestThinInsetBorderIsDrawn(t *testing.T) {
	const layout = `size: 40 10
bkgColor: white
inner:
  - absolute: left/0 top/4
    width: 40
    height: 1
    border: 1 0xff0000 top inset
`

	img := renderLayout(t, layout)

	drawn := 0
	for x := 0; x < 40; x++ {
		if r, g, b := rgb(img.At(x, 4)); r == 255 && g == 0 && b == 0 {
			drawn++
		}
	}
	if drawn != 40 {
		t.Errorf("a one pixel rule drew %v of 40 pixels", drawn)
	}
}

// The dash pattern runs along the side, in the lengths it was given.
func TestDashedBorderAlternates(t *testing.T) {
	const layout = `size: 40 10
bkgColor: white
inner:
  - absolute: left/0 top/4
    width: 40
    height: 1
    border: 1 0xff0000 dashed/4/4 top inset
`

	img := renderLayout(t, layout)

	for x := 0; x < 40; x++ {
		r, g, b := rgb(img.At(x, 4))
		isDash := r == 255 && g == 0 && b == 0
		if want := (x/4)%2 == 0; isDash != want {
			t.Errorf("at x=%v the rule is drawn=%v, want %v", x, isDash, want)
		}
	}
}

// Wrapped rows are spaced by innerGap. They used to advance by whatever gap
// justify had spread the row with, so a row distributed with space-between
// pushed the next one down by the free space left on the line.
func TestWrappedRowsAreSpacedByInnerGap(t *testing.T) {
	// Two 40 wide children fit on a 100 wide line and are pushed apart by 20;
	// the third wraps. Its top must follow the height of the first line, not
	// that horizontal spread.
	const layout = `width: 100
bkgColor: white
innerDirection: row
innerWrap: wrap
justify: space-between
inner:
  - size: 40 20
    bkgColor: 0x0000ff
  - size: 40 20
    bkgColor: 0x0000ff
  - size: 40 20
    bkgColor: 0x00ff00
`

	img := renderLayout(t, layout)

	top := -1
	for y := 0; y < img.Bounds().Dy(); y++ {
		if r, g, b := rgb(img.At(20, y)); r == 0 && g == 255 && b == 0 {
			top = y
			break
		}
	}
	if top != 20 {
		t.Errorf("the wrapped row starts at %v, want 20", top)
	}
}
