package fonts

import (
	_ "embed"
	"fmt"
	"io"
	"io/fs"
	"math"
	"slices"
	"strconv"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed default.ttf
var defaultFontFile []byte

const DefaultFamily = "Roboto"

// maxCachedFaces caps how many realized faces a single FaceSet keeps.
// Layouts with expression-driven font sizes can otherwise grow it without bound.
const maxCachedFaces = 64

// FaceTemplate describes a font face to load, as declared in a layout.
type FaceTemplate struct {
	Family string
	Style  string
	Weight string
	File   string
}

type FaceDescription struct {
	Family string
	Size   float64
	Weight int
	Style  font.Style
}

type loadedFontFace struct {
	family string
	style  font.Style
	weight int
	font   *opentype.Font
}

// Registry holds the parsed fonts of a single renderer.
//
// *opentype.Font is safe to share between goroutines (sfnt documents that Font
// methods are concurrent-safe as long as each call uses a different Buffer), so
// the registry itself is immutable once built and can be shared freely.
// font.Face is NOT safe to share, because it owns the glyph buffer - realized
// faces live in a FaceSet instead, one per in-flight render.
type Registry struct {
	faces []loadedFontFace
	// defaultFont is the embedded fallback, used whenever no declared face
	// matches a requested DefaultFamily description.
	defaultFont *opentype.Font
	pool        sync.Pool
}

// NewRegistry parses the embedded default font plus every declared face.
func NewRegistry(templates []FaceTemplate, fsys fs.FS) (*Registry, error) {
	defaultFont, err := opentype.Parse(defaultFontFile)
	if err != nil {
		return nil, fmt.Errorf("can't parse embedded default font: %w", err)
	}

	r := &Registry{
		faces:       make([]loadedFontFace, 0, len(templates)),
		defaultFont: defaultFont,
	}
	r.pool.New = func() any {
		return &FaceSet{
			reg:   r,
			faces: make(map[FaceDescription]font.Face, 8),
		}
	}

	for _, t := range templates {
		if err := r.loadFont(t, nil, fsys); err != nil {
			return nil, fmt.Errorf("failed loading font faces: %w", err)
		}
	}

	return r, nil
}

func (r *Registry) loadFont(template FaceTemplate, content []byte, fsys fs.FS) error {
	var loaded loadedFontFace

	switch template.Style {
	case "", "normal":
		loaded.style = font.StyleNormal
	case "italic":
		loaded.style = font.StyleItalic
	default:
		return fmt.Errorf("wrong style %v for font %v", template.Style, template.Family)
	}

	if template.Family == "" {
		return fmt.Errorf("font family not specified")
	}
	loaded.family = template.Family

	if template.Weight != "" {
		weight, err := strconv.Atoi(template.Weight)
		if err != nil {
			return fmt.Errorf("font weight not valid")
		}
		loaded.weight = weight
	}

	if slices.IndexFunc(r.faces, func(f loadedFontFace) bool {
		return f.family == loaded.family && f.style == loaded.style && f.weight == loaded.weight
	}) != -1 {
		return fmt.Errorf("font face %v (style %v, weight %v) is declared more than once",
			loaded.family, template.Style, loaded.weight)
	}

	if content == nil {
		f, err := fsys.Open(template.File)
		if err != nil {
			return fmt.Errorf("can't open font file %v", template.File)
		}
		defer func() { _ = f.Close() }()

		content, err = io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("can't read font file %v", template.File)
		}
	}

	fnt, err := opentype.Parse(content)
	if err != nil {
		return fmt.Errorf("can't parse font file %v", template.File)
	}
	loaded.font = fnt

	r.faces = append(r.faces, loaded)

	return nil
}

// GetFont returns the font nearest by weight within the requested family and style.
func (r *Registry) GetFont(fd FaceDescription) (*opentype.Font, error) {
	minWeightDiff := math.Inf(1)
	var found *opentype.Font

	for i := range r.faces {
		f := &r.faces[i]
		if fd.Family != f.family || fd.Style != f.style {
			continue
		}
		weightDiff := math.Abs(float64(f.weight - fd.Weight))
		if weightDiff < minWeightDiff {
			found = f.font
			minWeightDiff = weightDiff
		}
	}

	if found == nil {
		if fd.Family != DefaultFamily {
			return nil, fmt.Errorf("font face (%v) not found", fd.Family)
		}
		found = r.defaultFont
	}

	return found, nil
}

// AcquireFaceSet returns a face set for the duration of one render. The returned
// set belongs to the calling goroutine until it is released.
func (r *Registry) AcquireFaceSet() *FaceSet {
	return r.pool.Get().(*FaceSet)
}

func (r *Registry) ReleaseFaceSet(fs *FaceSet) {
	if fs == nil {
		return
	}
	if len(fs.faces) > maxCachedFaces {
		clear(fs.faces)
	}
	r.pool.Put(fs)
}

// FaceSet caches realized font.Face values for a single render.
//
// It is NOT safe for concurrent use: font.Face reuses an internal glyph mask
// buffer, so a face handed to two goroutines corrupts both. Acquire one set per
// render via Registry.AcquireFaceSet.
type FaceSet struct {
	reg   *Registry
	faces map[FaceDescription]font.Face
}

// Face returns the realized face for fd, creating it on first use.
func (fs *FaceSet) Face(fd FaceDescription) (font.Face, error) {
	if face, ok := fs.faces[fd]; ok {
		return face, nil
	}

	f, err := fs.reg.GetFont(fd)
	if err != nil {
		return nil, err
	}

	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    fd.Size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("can't create face for %v: %w", fd.Family, err)
	}

	fs.faces[fd] = face

	return face, nil
}

func (fs *FaceSet) MeasureTextWidth(text string, fd FaceDescription) float64 {
	face, err := fs.Face(fd)
	if err != nil {
		return 0.0
	}

	var width float64
	for _, runeValue := range text {
		advance, _ := face.GlyphAdvance(runeValue)
		width += float64(advance)
	}

	return width / 64 // Convert from 26.6 fixed-point to float64
}

func GetFontFaceBaseLineOffset(face font.Face, lineHeight float64) float64 {
	metrics := face.Metrics()
	ascent := float64(metrics.Ascent.Ceil())
	descent := float64(metrics.Descent.Ceil())
	baselineOffset := (lineHeight - (ascent + descent)) / 2
	return ascent + baselineOffset
}
