package decorender

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

var updateGolden = flag.Bool("update-golden", false, "rewrite testdata/golden.png from the current renderer output")

const (
	goldenLayout = "testdata/golden.yaml"
	goldenImage  = "testdata/golden.png"
)

// renderGolden renders the reference layout. Local files resolve from the repo
// root, so paths inside the layout are written relative to it.
func renderGolden(t testing.TB) []byte {
	t.Helper()

	r, err := NewRenderer(goldenLayout, &Options{LocalFiles: os.DirFS(".")})
	if err != nil {
		t.Fatalf("parsing %s: %v", goldenLayout, err)
	}

	var buf bytes.Buffer
	if err := r.RenderAndWrite(nil, EncodeFormatPNG, &buf, &RenderOptions{UseSample: true}); err != nil {
		t.Fatalf("rendering %s: %v", goldenLayout, err)
	}

	return buf.Bytes()
}

func decodePNG(t testing.TB, what string, data []byte) image.Image {
	t.Helper()

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decoding %s: %v", what, err)
	}
	return img
}

// TestGoldenRender renders the reference layout and compares it to the stored
// golden image. Pixels are compared rather than encoded bytes, so a change in
// the PNG encoder between Go releases does not fail the test.
//
// Run with -update-golden after an intentional rendering change, and inspect
// the resulting image before committing it.
func TestGoldenRender(t *testing.T) {
	got := renderGolden(t)

	if *updateGolden {
		if err := os.WriteFile(goldenImage, got, 0o644); err != nil {
			t.Fatalf("writing %s: %v", goldenImage, err)
		}
		t.Logf("wrote %s (%d bytes)", goldenImage, len(got))
		return
	}

	wantBytes, err := os.ReadFile(goldenImage)
	if err != nil {
		t.Fatalf("reading %s (run: go test -run TestGoldenRender -update-golden): %v", goldenImage, err)
	}

	gotImg := decodePNG(t, "rendered output", got)
	wantImg := decodePNG(t, goldenImage, wantBytes)

	if gotImg.Bounds() != wantImg.Bounds() {
		t.Fatalf("size changed: got %v, golden %v", gotImg.Bounds(), wantImg.Bounds())
	}

	differing, maxDelta := comparePixels(gotImg, wantImg)
	if differing == 0 {
		return
	}

	// Keep the actual output around so the difference can be looked at.
	actual := filepath.Join(t.TempDir(), "golden_actual.png")
	if err := os.WriteFile(actual, got, 0o644); err != nil {
		t.Logf("could not save actual output: %v", err)
	}

	bounds := wantImg.Bounds()
	total := bounds.Dx() * bounds.Dy()
	t.Errorf("render differs from %s: %d of %d pixels (%.4f%%), max channel delta %d\nactual output written to %s",
		goldenImage, differing, total, 100*float64(differing)/float64(total), maxDelta, actual)
}

func comparePixels(a, b image.Image) (differing int, maxDelta int) {
	bounds := b.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			ar, ag, ab, aa := a.At(x, y).RGBA()
			br, bg, bb, ba := b.At(x, y).RGBA()
			if ar == br && ag == bg && ab == bb && aa == ba {
				continue
			}
			differing++
			for _, d := range []int{
				int(ar>>8) - int(br>>8),
				int(ag>>8) - int(bg>>8),
				int(ab>>8) - int(bb>>8),
				int(aa>>8) - int(ba>>8),
			} {
				if d < 0 {
					d = -d
				}
				if d > maxDelta {
					maxDelta = d
				}
			}
		}
	}
	return differing, maxDelta
}

// The reference layout is the widest exercise of the renderer available, so it
// doubles as the concurrency and allocation benchmark.
func TestGoldenRenderIsConcurrencySafe(t *testing.T) {
	r, err := NewRenderer(goldenLayout, &Options{LocalFiles: os.DirFS(".")})
	if err != nil {
		t.Fatalf("parsing %s: %v", goldenLayout, err)
	}

	const goroutines, iterations = 8, 5

	errs := make(chan error, goroutines*iterations)
	done := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < iterations; j++ {
				if err := r.RenderAndWrite(nil, EncodeFormatNone, nil, &RenderOptions{UseSample: true}); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	for i := 0; i < goroutines; i++ {
		<-done
	}
	close(errs)

	for err := range errs {
		t.Errorf("concurrent render of the reference layout failed: %v", err)
	}
}

func BenchmarkGoldenRender(b *testing.B) {
	r, err := NewRenderer(goldenLayout, &Options{LocalFiles: os.DirFS(".")})
	if err != nil {
		b.Fatalf("parsing %s: %v", goldenLayout, err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := r.RenderAndWrite(nil, EncodeFormatNone, nil, &RenderOptions{UseSample: true}); err != nil {
			b.Fatal(err)
		}
	}
}

// The two layout keys the reference image cannot express are asserted here
// instead: `scale` would double the golden's size, and `only` discards every
// other node by design, so it cannot coexist with the rest of the layout.

func TestScaleOption(t *testing.T) {
	cases := []struct {
		scale         string
		width, height int
	}{
		{scale: "1", width: 200, height: 100},
		{scale: "2", width: 400, height: 200},
		{scale: "0.5", width: 100, height: 50},
		{scale: "", width: 200, height: 100},     // absent: defaults to 1
		{scale: "junk", width: 200, height: 100}, // unparseable: defaults to 1
	}

	for _, c := range cases {
		layout := "size: 200 100\nbkgColor: salmon\n"
		if c.scale != "" {
			layout += "scale: " + c.scale + "\n"
		}

		r, err := NewRendererWithTemplate([]byte(layout), nil)
		if err != nil {
			t.Errorf("scale %q: parse: %v", c.scale, err)
			continue
		}

		img, release, err := r.Render(nil, nil)
		if err != nil {
			t.Errorf("scale %q: render: %v", c.scale, err)
			continue
		}

		if got := img.Bounds(); got.Dx() != c.width || got.Dy() != c.height {
			t.Errorf("scale %q: got %dx%d, want %dx%d", c.scale, got.Dx(), got.Dy(), c.width, c.height)
		}
		release()
	}
}

func TestDebugOnlyKeepsSingleNode(t *testing.T) {
	// `only` marks the one subtree to render; the root auto-sizes around it, so
	// the 300x300 sibling must not contribute to the result.
	layout := "bkgColor: gray\ninner:\n" +
		"  - size: 300 300\n    bkgColor: red\n" +
		"  - size: 40 25\n    bkgColor: blue\n    only: \"1\"\n"

	r, err := NewRendererWithTemplate([]byte(layout), nil)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	img, release, err := r.Render(nil, nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	defer release()

	if got := img.Bounds(); got.Dx() != 40 || got.Dy() != 25 {
		t.Errorf("only: got %dx%d, want 40x25 (the marked node alone)", got.Dx(), got.Dy())
	}
}

// innerColumnAlign positions every child of a column, not just the first one.
func TestInnerColumnAlign(t *testing.T) {
	// Three children of decreasing width stacked in a column. With left align
	// each starts at 0; with center and right they start at increasing offsets,
	// so a single mis-aligned child is visible in the span it covers.
	const containerW = 120
	widths := []int{120, 60, 20}

	layout := "size: " + itoa(containerW) + " 30\nbkgColor: white\ninnerDirection: column\n" +
		"innerColumnAlign: %s\ninner:\n"
	for _, w := range widths {
		layout += "  - size: " + itoa(w) + " 10\n    bkgColor: black\n"
	}

	for _, c := range []struct {
		align string
		want  []int // expected left edge of each child
	}{
		{"left", []int{0, 0, 0}},
		{"center", []int{0, 30, 50}},
		{"right", []int{0, 60, 100}},
	} {
		src := fmt.Sprintf(layout, c.align)

		r, err := NewRendererWithTemplate([]byte(src), nil)
		if err != nil {
			t.Errorf("align %q: parse: %v", c.align, err)
			continue
		}

		img, release, err := r.Render(nil, nil)
		if err != nil {
			t.Errorf("align %q: render: %v", c.align, err)
			continue
		}

		for row, want := range c.want {
			y := row*10 + 5 // middle of the row, away from any edge blending
			got := firstDarkPixel(img, y)
			if got != want {
				t.Errorf("align %q: child %d (width %d) starts at x=%d, want %d",
					c.align, row, widths[row], got, want)
			}
		}
		release()
	}
}

// firstDarkPixel returns the x of the leftmost non-white pixel on row y, or -1.
func firstDarkPixel(img image.Image, y int) int {
	b := img.Bounds()
	for x := b.Min.X; x < b.Max.X; x++ {
		r, g, bl, _ := img.At(x, y).RGBA()
		if r>>8 < 128 && g>>8 < 128 && bl>>8 < 128 {
			return x - b.Min.X
		}
	}
	return -1
}

func itoa(v int) string { return strconv.Itoa(v) }

// A descendant of a forEach node sees the loop counter of the iteration it is
// part of. It used to see 0, because re-entering the loop machinery for a node
// with no forEach of its own reset the index.
func TestForEachIndexReachesDescendants(t *testing.T) {
	// Each repeated row holds a bar whose width is driven by index, so the
	// counter each descendant saw can be read straight off the image.
	const layout = `size: 200 60
bkgColor: white
innerDirection: column
inner:
  - forEach: Items
    width: 200
    height: 20
    inner:
      - height: 20
        width: ~ (index + 1) * 50
        bkgColor: black
`

	r, err := NewRendererWithTemplate([]byte(layout), nil)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	img, release, err := r.Render(struct{ Items []string }{Items: []string{"a", "b", "c"}}, nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	defer release()

	for row, want := range []int{50, 100, 150} {
		if got := darkRunWidth(img, row*20+10); got != want {
			t.Errorf("row %d: bar is %dpx wide, want %dpx (index %d)", row, got, want, row)
		}
	}
}

// darkRunWidth counts the dark pixels on row y.
func darkRunWidth(img image.Image, y int) int {
	b := img.Bounds()
	n := 0
	for x := b.Min.X; x < b.Max.X; x++ {
		r, g, bl, _ := img.At(x, y).RGBA()
		if r>>8 < 128 && g>>8 < 128 && bl>>8 < 128 {
			n++
		}
	}
	return n
}

// A hex color written plainly in a sample is an integer as far as YAML is
// concerned. It used to reach the color parser as its decimal spelling, fail,
// and render transparent without a word.
func TestSampleHexColorRenders(t *testing.T) {
	// Quoted and unquoted spellings of the same color must render identically,
	// and neither may come out transparent.
	const layout = `size: 40 40
bkgColor: white
sample:
  plain: 0x4fc3f7
  quoted: '0x4fc3f7'
  withAlpha: 0xffd54fff
  notAColor: 0xff
inner:
  - size: 40 10
    bkgColor: ~ plain
  - size: 40 10
    bkgColor: ~ quoted
  - size: 40 10
    bkgColor: ~ withAlpha
  - size: 40 10
    bkgColor: ~ string(notAColor)
`

	r, err := NewRendererWithTemplate([]byte(layout), nil)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	img, release, err := r.Render(nil, &RenderOptions{UseSample: true})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	defer release()

	at := func(row int) (uint32, uint32, uint32) {
		cr, cg, cb, _ := img.At(20, row*10+5).RGBA()
		return cr >> 8, cg >> 8, cb >> 8
	}

	wantR, wantG, wantB := uint32(0x4f), uint32(0xc3), uint32(0xf7)
	for _, row := range []int{0, 1} {
		if cr, cg, cb := at(row); cr != wantR || cg != wantG || cb != wantB {
			t.Errorf("row %d: got #%02x%02x%02x, want #%02x%02x%02x",
				row, cr, cg, cb, wantR, wantG, wantB)
		}
	}
	if cr, cg, cb := at(2); cr != 0xff || cg != 0xd5 || cb != 0x4f {
		t.Errorf("row 2 (8-digit hex): got #%02x%02x%02x, want #ffd54f", cr, cg, cb)
	}
	// 0xff is not a color, so it stays an integer and the expression above
	// stringifies it to "255" - which the color parser rejects, leaving the
	// bar transparent over the white background.
	if cr, cg, cb := at(3); cr != 0xff || cg != 0xff || cb != 0xff {
		t.Errorf("row 3: a short hex scalar must stay an integer, got #%02x%02x%02x", cr, cg, cb)
	}
}
