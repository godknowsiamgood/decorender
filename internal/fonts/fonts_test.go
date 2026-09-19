package fonts

import (
	"encoding/binary"
	"os"
	"strings"
	"testing"

	"golang.org/x/image/font"
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
