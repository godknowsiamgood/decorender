package fonts

import (
	"os"
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
