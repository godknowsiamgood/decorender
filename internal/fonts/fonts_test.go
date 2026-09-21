package fonts

import (
	"encoding/binary"
	"os"
	"strings"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"image"
)

const testFontFile = "default.ttf"

// Registries used to share a package-level slice of loaded faces, so a face
// declared by one renderer leaked into every other renderer in the process.
func TestRegistriesAreIsolated(t *testing.T) {
	fsys := os.DirFS(".")

	withFace, err := NewRegistry([]FaceTemplate{
		{Family: "Custom", Style: "normal", Weight: "400", File: testFontFile},
	}, fsys)
	if err != nil {
		t.Fatalf("building registry with a declared face: %v", err)
	}

	withoutFace, err := NewRegistry(nil, fsys)
	if err != nil {
		t.Fatalf("building empty registry: %v", err)
	}

	fd := FaceDescription{Family: "Custom", Size: 16, Weight: 400, Style: font.StyleNormal}

	if _, err := withFace.GetFont(fd); err != nil {
		t.Errorf("declared face should resolve in its own registry, got %v", err)
	}
	if _, err := withoutFace.GetFont(fd); err == nil {
		t.Error("a face declared in another registry must not be visible here")
	}
}

// Declaring the same family/style/weight twice used to be silently ignored,
// so the second file never took effect.
func TestDuplicateFaceDeclarationIsRejected(t *testing.T) {
	_, err := NewRegistry([]FaceTemplate{
		{Family: "Custom", Style: "normal", Weight: "400", File: testFontFile},
		{Family: "Custom", Style: "normal", Weight: "400", File: testFontFile},
	}, os.DirFS("."))

	if err == nil {
		t.Fatal("expected an error when the same face is declared twice")
	}
}

// A face the user declares under the default family must win over the embedded font.
func TestDeclaredFaceOverridesEmbeddedDefault(t *testing.T) {
	reg, err := NewRegistry([]FaceTemplate{
		{Family: DefaultFamily, Style: "normal", Weight: "400", File: testFontFile},
	}, os.DirFS("."))
	if err != nil {
		t.Fatalf("building registry: %v", err)
	}

	fd := FaceDescription{Family: DefaultFamily, Size: 16, Weight: 400, Style: font.StyleNormal}
	got, err := reg.GetFont(fd)
	if err != nil {
		t.Fatalf("resolving default family: %v", err)
	}
	if got == reg.defaultFont {
		t.Error("expected the declared face, got the embedded default")
	}
}

// An unknown family is an error; an unknown weight falls back to the nearest one.
func TestUnknownFamilyAndNearestWeight(t *testing.T) {
	reg, err := NewRegistry([]FaceTemplate{
		{Family: "Custom", Style: "normal", Weight: "400", File: testFontFile},
	}, os.DirFS("."))
	if err != nil {
		t.Fatalf("building registry: %v", err)
	}

	if _, err := reg.GetFont(FaceDescription{Family: "Nope", Size: 16, Weight: 400}); err == nil {
		t.Error("expected an error for an undeclared family")
	}

	if _, err := reg.GetFont(FaceDescription{Family: "Custom", Size: 16, Weight: 900}); err != nil {
		t.Errorf("expected nearest-weight fallback within a declared family, got %v", err)
	}

	// The default family always resolves, even with no declared faces.
	empty, err := NewRegistry(nil, os.DirFS("."))
	if err != nil {
		t.Fatalf("building empty registry: %v", err)
	}
	if _, err := empty.GetFont(FaceDescription{Family: DefaultFamily, Size: 16, Weight: 400}); err != nil {
		t.Errorf("default family must always resolve, got %v", err)
	}
}

// Each acquired FaceSet must hand out its own font.Face: Face values own a
// mutable glyph buffer and corrupt each other when shared across goroutines.
func TestFaceSetsDoNotShareFaces(t *testing.T) {
	reg, err := NewRegistry(nil, os.DirFS("."))
	if err != nil {
		t.Fatalf("building registry: %v", err)
	}

	fd := FaceDescription{Family: DefaultFamily, Size: 16, Weight: 400, Style: font.StyleNormal}

	first := reg.AcquireFaceSet()
	second := reg.AcquireFaceSet()

	faceA, err := first.Face(fd)
	if err != nil {
		t.Fatalf("first face: %v", err)
	}
	faceB, err := second.Face(fd)
	if err != nil {
		t.Fatalf("second face: %v", err)
	}

	if faceA == faceB {
		t.Error("two concurrently held face sets must not share a font.Face")
	}

	// Within one set the face is reused, so faces are not rebuilt per call.
	again, err := first.Face(fd)
	if err != nil {
		t.Fatalf("re-reading first face: %v", err)
	}
	if again != faceA {
		t.Error("a face set should cache its realized faces")
	}

	reg.ReleaseFaceSet(first)
	reg.ReleaseFaceSet(second)
}

// variableFont builds the smallest file the axis reader looks at: a table
// directory with an fvar table holding one axis. Real variable fonts cannot be
// checked in here, so the bytes are made by hand.
func variableFont(axis string, defaultValue float64) []byte {
	const (
		headerSize = 12
		recordSize = 16
		axisRecord = 20
	)

	fvar := make([]byte, 16+axisRecord)
	binary.BigEndian.PutUint32(fvar[0:], 0x00010000) // version
	binary.BigEndian.PutUint16(fvar[4:], 16)         // axes start right after the header
	binary.BigEndian.PutUint16(fvar[8:], 1)          // one axis
	binary.BigEndian.PutUint16(fvar[10:], axisRecord)

	copy(fvar[16:], axis)
	binary.BigEndian.PutUint32(fvar[16+4:], uint32(int32(100*65536)))          // min
	binary.BigEndian.PutUint32(fvar[16+8:], uint32(int32(defaultValue*65536))) // default
	binary.BigEndian.PutUint32(fvar[16+12:], uint32(int32(900*65536)))         // max

	file := make([]byte, headerSize+recordSize)
	binary.BigEndian.PutUint32(file[0:], 0x00010000)
	binary.BigEndian.PutUint16(file[4:], 1) // one table
	copy(file[headerSize:], "fvar")
	binary.BigEndian.PutUint32(file[headerSize+8:], uint32(len(file)))
	binary.BigEndian.PutUint32(file[headerSize+12:], uint32(len(fvar)))

	return append(file, fvar...)
}

func TestVariableAxisDefault(t *testing.T) {
	if _, ok := variableAxisDefault(variableFont("wght", 400), "wght"); !ok {
		t.Error("the wght axis of a variable font was not found")
	}
	if got, _ := variableAxisDefault(variableFont("wght", 400), "wght"); got != 400 {
		t.Errorf("wght default = %v, want 400", got)
	}
	if _, ok := variableAxisDefault(variableFont("wdth", 100), "wght"); ok {
		t.Error("a font without a wght axis reported one")
	}

	static, err := os.ReadFile(testFontFile)
	if err != nil {
		t.Fatalf("reading %v: %v", testFontFile, err)
	}
	if _, ok := variableAxisDefault(static, "wght"); ok {
		t.Error("a static font reported a wght axis")
	}
}

// Only the default instance of a variable font is rendered, so a weight it
// cannot produce has to be refused rather than quietly drawn at 400.
func TestVariableFontRefusesAWeightItCannotRender(t *testing.T) {
	r := &Registry{}

	err := r.loadFont(FaceTemplate{Family: "Var", Weight: "700", File: "var.ttf"}, variableFont("wght", 400), nil)
	if err == nil {
		t.Fatal("declaring weight 700 against a variable font was accepted")
	}
	if !strings.Contains(err.Error(), "variable font") {
		t.Errorf("error does not explain the problem: %v", err)
	}

	// The weight the file does render is fine - it parses no further here,
	// so only the absence of the variable font error matters.
	err = r.loadFont(FaceTemplate{Family: "Var", Weight: "400", File: "var.ttf"}, variableFont("wght", 400), nil)
	if err != nil && strings.Contains(err.Error(), "variable font") {
		t.Errorf("the default weight was refused: %v", err)
	}
}

// TestMeasurementIsAdditive pins the property that lets the layout add up a
// row of measured words instead of measuring the joined line a second time:
// widths are a plain sum of glyph advances, with nothing between one glyph and
// the next.
//
// Introducing kerning would break this, and the layout would have to measure
// joined rows again rather than adding their parts.
func TestMeasurementIsAdditive(t *testing.T) {
	registry, err := NewRegistry(nil, os.DirFS("."))
	if err != nil {
		t.Fatalf("building registry: %v", err)
	}

	faces := registry.AcquireFaceSet()
	defer registry.ReleaseFaceSet(faces)

	description := FaceDescription{Family: DefaultFamily, Size: 16, Weight: 400}
	space, err := faces.MeasureTextWidth(" ", description)
	if err != nil {
		t.Fatalf("measuring a space: %v", err)
	}

	for _, line := range []string{
		"the quick brown fox jumps over",
		"AVATAR To Wave Yearn",
		"Ünïcödé wörds hëre",
	} {
		words := strings.Fields(line)

		var summed float64
		for _, w := range words {
			width, err := faces.MeasureTextWidth(w, description)
			if err != nil {
				t.Fatalf("measuring %q: %v", w, err)
			}
			summed += width
		}
		summed += float64(len(words)-1) * space

		joined, err := faces.MeasureTextWidth(line, description)
		if err != nil {
			t.Fatalf("measuring %q: %v", line, err)
		}
		if summed != joined {
			t.Errorf("%q: the words add up to %v but the line measures %v", line, summed, joined)
		}
	}
}

// TestAdvancesAreWholePixels pins the property that lets a rasterized glyph be
// kept and redrawn: hinted faces advance the pen by whole pixels, so a pen
// starting on a pixel never lands between two, and a glyph's mask does not
// depend on where it is drawn.
//
// Dropping to font.HintingNone would break this - advances become fractional
// and a glyph has to be rasterized for its sub-pixel position - so the face
// options and this test have to change together.
func TestAdvancesAreWholePixels(t *testing.T) {
	registry, err := NewRegistry(nil, os.DirFS("."))
	if err != nil {
		t.Fatalf("building registry: %v", err)
	}

	faces := registry.AcquireFaceSet()
	defer registry.ReleaseFaceSet(faces)

	for _, size := range []float64{9, 13, 16, 23.5, 44, 88} {
		face, err := faces.Face(FaceDescription{Family: DefaultFamily, Size: size, Weight: 400})
		if err != nil {
			t.Fatalf("realizing a face at size %v: %v", size, err)
		}

		for r := rune(' '); r < 127; r++ {
			advance, ok := face.GlyphAdvance(r)
			if ok && advance%64 != 0 {
				t.Fatalf("at size %v, %q advances %v, which is not a whole pixel", size, r, advance)
			}
		}
	}
}

// TestCachedGlyphMatchesAFreshOne draws every printable ASCII character from
// the cache and from a face that has never seen it, and compares the masks.
func TestCachedGlyphMatchesAFreshOne(t *testing.T) {
	registry, err := NewRegistry(nil, os.DirFS("."))
	if err != nil {
		t.Fatalf("building registry: %v", err)
	}

	faces := registry.AcquireFaceSet()
	defer registry.ReleaseFaceSet(faces)

	description := FaceDescription{Family: DefaultFamily, Size: 24, Weight: 400}

	for r := rune(' '); r < 127; r++ {
		cached, err := faces.Glyph(description, r)
		if err != nil {
			t.Fatalf("caching %q: %v", r, err)
		}

		// A face of its own, so the comparison is against a rasterization that
		// no cache took part in.
		fresh := registry.AcquireFaceSet()
		face, err := fresh.Face(description)
		if err != nil {
			t.Fatalf("realizing a face: %v", err)
		}
		bounds, mask, maskPoint, advance, ok := face.Glyph(fixed.P(0, 0), r)
		registry.ReleaseFaceSet(fresh)

		if ok != cached.Found {
			t.Errorf("%q: cache says found=%v, a fresh face says %v", r, cached.Found, ok)
			continue
		}
		if !ok {
			continue
		}
		if advance != cached.Advance {
			t.Errorf("%q: cached advance %v, fresh advance %v", r, cached.Advance, advance)
		}
		if bounds.Min != cached.Offset {
			t.Errorf("%q: cached offset %v, fresh offset %v", r, cached.Offset, bounds.Min)
		}

		for y := 0; y < bounds.Dy(); y++ {
			for x := 0; x < bounds.Dx(); x++ {
				want := mask.(*image.Alpha).AlphaAt(maskPoint.X+x, maskPoint.Y+y)
				if got := cached.Mask.AlphaAt(x, y); got != want {
					t.Fatalf("%q: cached mask differs at (%d,%d): %v, want %v", r, x, y, got, want)
				}
			}
		}
	}
}

// TestGlyphCacheStaysWithinItsMemoryBudget rasterizes far more glyph than the
// budget allows and checks that the cache gives memory back as it evicts.
func TestGlyphCacheStaysWithinItsMemoryBudget(t *testing.T) {
	registry, err := NewRegistry(nil, os.DirFS("."))
	if err != nil {
		t.Fatalf("building registry: %v", err)
	}

	faces := registry.AcquireFaceSet()
	defer registry.ReleaseFaceSet(faces)

	// Large glyphs at many sizes, which is the shape that grows the cache
	// fastest: one size's alphabet alone is a sizeable fraction of the budget.
	for size := 200; size < 260; size++ {
		description := FaceDescription{Family: DefaultFamily, Size: float64(size), Weight: 400}
		for r := rune('A'); r <= 'Z'; r++ {
			if _, err := faces.Glyph(description, r); err != nil {
				t.Fatalf("caching %q at size %d: %v", r, size, err)
			}
		}
	}

	if faces.glyphBytes > maxCachedGlyphBytes {
		t.Errorf("cached masks occupy %d bytes, over the %d byte budget", faces.glyphBytes, maxCachedGlyphBytes)
	}
	if faces.glyphBytes <= 0 {
		t.Errorf("cached masks occupy %d bytes, so the accounting has drifted", faces.glyphBytes)
	}
}
