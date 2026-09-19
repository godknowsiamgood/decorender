package decorender

import (
	"fmt"
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

// innerGap takes the pair "<row> <column>", as CSS writes gap: the first value
// spaces rows from each other, the second one spaces children within a row.
// One value still applies to both axes. Until they could be told apart, a
// legend that wrapped had to choose between lines close together and items
// far enough apart to be read as separate.
func TestInnerGapTakesRowAndColumnValues(t *testing.T) {
	cases := []struct {
		name   string
		gap    string
		column int
		row    int
	}{
		{name: "row and column", gap: "2 20", column: 20, row: 2},
		{name: "one value for both axes", gap: "6", column: 6, row: 6},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Two 40 wide children fit on a 100 wide line, the third wraps.
			layout := fmt.Sprintf(`width: 100
bkgColor: white
innerDirection: row
innerWrap: wrap
innerGap: %s
inner:
  - size: 40 20
    bkgColor: 0x0000ff
  - size: 40 20
    bkgColor: 0x0000ff
  - size: 40 20
    bkgColor: 0x00ff00
`, c.gap)

			img := renderLayout(t, layout)

			second := -1
			for x := 40; x < img.Bounds().Dx(); x++ {
				if r, g, b := rgb(img.At(x, 10)); r == 0 && g == 0 && b == 255 {
					second = x
					break
				}
			}
			if want := 40 + c.column; second != want {
				t.Errorf("the second child starts at %v, want %v", second, want)
			}

			top := -1
			for y := 0; y < img.Bounds().Dy(); y++ {
				if r, g, b := rgb(img.At(20, y)); r == 0 && g == 255 && b == 0 {
					top = y
					break
				}
			}
			if want := 20 + c.row; top != want {
				t.Errorf("the wrapped row starts at %v, want %v", top, want)
			}
		})
	}
}

// countAcross returns how many pixels of the given color a row of the image
// holds, which is how these tests count the nodes a forEach produced.
func countAcross(img image.Image, y int, want color.RGBA) int {
	n := 0
	for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
		if r, g, b := rgb(img.At(x, y)); r == want.R && g == want.G && b == want.B {
			n++
		}
	}
	return n
}

// forEach used to take the name of a field and nothing else: an expression was
// evaluated, but its result was turned into text and looked up as a name. It
// now takes what the expression evaluated to - a list to walk, a count to
// repeat, or a flag that decides whether the node is drawn at all.
func TestForEachTakesAnExpressionResult(t *testing.T) {
	const sample = `sample:
  rows: [a, b, c]
  empty: []
  flag: true
  off: false
  count: 2
`

	cases := []struct {
		forEach string
		nodes   int
	}{
		{forEach: "rows", nodes: 3},            // a name, as before
		{forEach: "2", nodes: 2},               // a count, as before
		{forEach: "~ rows", nodes: 3},          // the list itself
		{forEach: "~ empty", nodes: 0},         // a list with nothing in it
		{forEach: "~ missing", nodes: 0},       // not there at all
		{forEach: "~ flag", nodes: 1},          // drawn
		{forEach: "~ off", nodes: 0},           // not drawn
		{forEach: "~ count", nodes: 2},         // a count
		{forEach: "~ len(rows) - 1", nodes: 2}, // and any expression giving one
	}

	for _, c := range cases {
		t.Run(c.forEach, func(t *testing.T) {
			layout := "size: 100 10\nbkgColor: white\ninnerDirection: row\n" + sample +
				"inner:\n  - forEach: '" + c.forEach + "'\n    size: 10 10\n    bkgColor: 0x0000ff\n"

			img := renderLayout(t, layout)

			if got := countAcross(img, 5, color.RGBA{B: 255}); got != c.nodes*10 {
				t.Errorf("forEach %q drew %v nodes, want %v", c.forEach, got/10, c.nodes)
			}
		})
	}
}

// A name means the same thing in a forEach as in an expression: expr renames a
// field by its `expr` tag, and forEach looks it up by exact field name, so a
// layout written against tagged data had to spell one of them differently.
func TestForEachFollowsExprTags(t *testing.T) {
	type row struct {
		Color string `expr:"color"`
	}
	type data struct {
		Rows   []row `expr:"rows"`
		Hidden []row `expr:"-"`
	}

	const layout = `size: 100 10
bkgColor: white
innerDirection: row
inner:
  - forEach: rows
    size: 10 10
    bkgColor: ~ value.color
`

	r, err := NewRendererWithTemplate([]byte(layout), nil)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	img, release, err := r.Render(data{Rows: []row{{Color: "0x0000ff"}, {Color: "0x0000ff"}}}, nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	defer release()

	if got := countAcross(img, 5, color.RGBA{B: 255}); got != 20 {
		t.Errorf("forEach over a tagged field drew %v nodes, want 2", got/10)
	}

	// A field the tag hides is not reachable by its Go name either.
	hidden := `size: 100 10
inner:
  - forEach: Hidden
    size: 10 10
    bkgColor: salmon
`
	hr, err := NewRendererWithTemplate([]byte(hidden), nil)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := hr.RenderAndWrite(data{}, EncodeFormatNone, nil, nil); err == nil {
		t.Error("a field hidden by its tag was still found by forEach")
	}
}
