package fonts

import (
	_ "embed"
	"encoding/binary"
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

	// A variable font carries its weight on an axis, and golang.org/x/image
	// has no notion of axes: it renders the default instance and nothing
	// else. Declaring 700 against such a file therefore drew the same glyphs
	// as 400, with nothing said about it.
	if def, ok := variableAxisDefault(content, "wght"); ok && template.Weight != "" && int(def) != loaded.weight {
		return fmt.Errorf("font file %v is a variable font, of which only the default instance is rendered - that is weight %v, not the declared %v. Declare that weight for this file, or use a static instance of the weight you want",
			template.File, int(def), loaded.weight)
	}

	fnt, err := opentype.Parse(content)
	if err != nil {
		return fmt.Errorf("can't parse font file %v", template.File)
	}
	loaded.font = fnt

	r.faces = append(r.faces, loaded)

	return nil
}

const (
	sfntHeaderSize     = 12
	tableRecordSize    = 16
	fvarHeaderSize     = 16
	fvarAxisRecordSize = 20
)

// findTable returns the contents of one table of a font file.
func findTable(content []byte, tag string) ([]byte, bool) {
	if len(content) < sfntHeaderSize {
		return nil, false
	}

	numTables := int(binary.BigEndian.Uint16(content[4:]))
	if len(content) < sfntHeaderSize+numTables*tableRecordSize {
		return nil, false
	}

	for i := 0; i < numTables; i++ {
		record := content[sfntHeaderSize+i*tableRecordSize:]
		if string(record[:4]) != tag {
			continue
		}

		offset := int(binary.BigEndian.Uint32(record[8:]))
		length := int(binary.BigEndian.Uint32(record[12:]))
		if offset < 0 || length < 0 || offset+length > len(content) {
			return nil, false
		}

		return content[offset : offset+length], true
	}

	return nil, false
}

// variableAxisDefault reports the default value of one variation axis, and
// whether the file has that axis at all. A font without an fvar table is not
// variable and has none.
func variableAxisDefault(content []byte, axis string) (float64, bool) {
	fvar, ok := findTable(content, "fvar")
	if !ok || len(fvar) < fvarHeaderSize {
		return 0, false
	}

	axesOffset := int(binary.BigEndian.Uint16(fvar[4:]))
	axisCount := int(binary.BigEndian.Uint16(fvar[8:]))
	axisSize := int(binary.BigEndian.Uint16(fvar[10:]))
	if axisSize < fvarAxisRecordSize {
		return 0, false
	}

	for i := 0; i < axisCount; i++ {
		start := axesOffset + i*axisSize
		if start < 0 || start+fvarAxisRecordSize > len(fvar) {
			break
		}

		record := fvar[start:]
		if string(record[:4]) != axis {
			continue
		}

		// Axis values are 16.16 fixed point.
		return float64(int32(binary.BigEndian.Uint32(record[8:]))) / 65536, true
	}

	return 0, false
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

// MeasureTextWidth reports how wide text is in the given face.
//
// The error is worth propagating rather than measuring as zero: a font that
// cannot be resolved makes every string zero-width, which collapses a
// content-sized layout to nothing and is reported as there being nothing to
// render, with no mention of the font.
func (fs *FaceSet) MeasureTextWidth(text string, fd FaceDescription) (float64, error) {
	face, err := fs.Face(fd)
	if err != nil {
		return 0, err
	}

	var width float64
	for _, runeValue := range text {
		advance, _ := face.GlyphAdvance(runeValue)
		width += float64(advance)
	}

	return width / 64, nil // Convert from 26.6 fixed-point to float64
}

func GetFontFaceBaseLineOffset(face font.Face, lineHeight float64) float64 {
	metrics := face.Metrics()
	ascent := float64(metrics.Ascent.Ceil())
	descent := float64(metrics.Descent.Ceil())
	baselineOffset := (lineHeight - (ascent + descent)) / 2
	return ascent + baselineOffset
}
