package fonts

import (
	_ "embed"
	"fmt"
	"github.com/godknowsiamgood/decorender/parsing"
	"golang.org/x/exp/slices"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"io"
	"io/fs"
	"math"
	"strconv"
	"sync"
)

//go:embed default.ttf
var defaultFontFile []byte

const DefaultFamily = "Roboto"

type FaceDescription struct {
	Family string
	Size   float64
	Weight int
	Style  font.Style
}

type cachedFontFace struct {
	family string
	style  font.Style
	weight int
	font   *opentype.Font
	uri    string
}

var faces []cachedFontFace
var facesMx sync.RWMutex

func GetFont(fd FaceDescription) (*opentype.Font, error) {
	facesMx.RLock()
	defer facesMx.RUnlock()

	minWeightDiff := 9999999.0
	var currentFace *cachedFontFace
	for _, f := range faces {
		f := f
		weightDiff := math.Abs(float64(f.weight - fd.Weight))
		if weightDiff < minWeightDiff && fd.Family == f.family && fd.Style == f.style {
			currentFace = &f
			minWeightDiff = weightDiff
		}
	}

	if currentFace == nil {
		if fd.Family != DefaultFamily {
			return nil, fmt.Errorf("font face (%v) not found", fd.Family)
		}
		currentFace = &faces[0] // first is the default face
	}

	return currentFace.font, nil
}

var prevFaceCache = struct {
	fd   FaceDescription
	face font.Face
	mx   sync.Mutex
}{}

func GetFontFace(fd FaceDescription) (font.Face, error) {
	prevFaceCache.mx.Lock()
	defer prevFaceCache.mx.Unlock()

	if prevFaceCache.fd == fd {
		return prevFaceCache.face, nil
	}

	f, err := GetFont(fd)
	if err != nil {
		return nil, err
	}

	face, _ := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    fd.Size,
		DPI:     72,
		Hinting: font.HintingFull,
	})

	prevFaceCache.face = face
	prevFaceCache.fd = fd

	return face, nil
}

func MeasureTextWidth(text string, fd FaceDescription) float64 {
	face, err := GetFontFace(fd)
	if err != nil {
		return 0.0
	}

	var width float64

	for _, runeValue := range []rune(text) {
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

func LoadFaces(faceTemplates []parsing.FontFace, fs fs.FS) error {
	facesMx.Lock()
	defer facesMx.Unlock()

	if faces == nil {
		faces = make([]cachedFontFace, 0, len(faceTemplates)+1)
	}

	if err := loadFont(parsing.FontFace{
		Family: "default",
		Style:  "normal",
		Weight: "400",
	}, defaultFontFile, fs); err != nil {
		return err
	}

	for _, ft := range faceTemplates {
		if err := loadFont(ft, nil, fs); err != nil {
			return fmt.Errorf("failed loading font faces: %w", err)
		}
	}

	return nil
}

func loadFont(faceTemplate parsing.FontFace, content []byte, fs fs.FS) error {
	cff := cachedFontFace{}
	if faceTemplate.Style != "" {
		switch faceTemplate.Style {
		case "normal":
			cff.style = font.StyleNormal
		case "italic":
			cff.style = font.StyleItalic
		default:
			return fmt.Errorf("wrong style %v for font %v", faceTemplate.Style, faceTemplate.Family)
		}
	} else {
		cff.style = font.StyleNormal
	}

	if faceTemplate.Family == "" {
		return fmt.Errorf("font family not specified")
	}
	cff.family = faceTemplate.Family

	if faceTemplate.Weight != "" {
		weight, err := strconv.Atoi(faceTemplate.Weight)
		if err != nil {
			return fmt.Errorf("font weight not valid")
		}
		cff.weight = weight
	}

	cff.uri = faceTemplate.File

	if slices.IndexFunc(faces, func(f cachedFontFace) bool {
		return f.family == cff.family && f.style == cff.style && f.weight == cff.weight
	}) == -1 {
		if content == nil {
			f, err := fs.Open(faceTemplate.File)
			if err != nil {
				return fmt.Errorf("can't open font file %v", faceTemplate.File)
			}
			content, err = io.ReadAll(f)
			if err != nil {
				return fmt.Errorf("can't read font file %v", faceTemplate.File)
			}
		}

		fnt, err := opentype.Parse(content)
		if err != nil {
			return fmt.Errorf("can't parse font file %v", faceTemplate.File)
		}

		cff.font = fnt

		faces = append(faces, cff)
	}

	return nil
}
