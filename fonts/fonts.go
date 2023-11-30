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

type loadedFontFace struct {
	family string
	style  font.Style
	weight int
	font   *opentype.Font
	uri    string
}

var loadedFaces []loadedFontFace
var loadedFacesMx sync.RWMutex

// GetFont returns font nearest by it`s weight
func GetFont(fd FaceDescription) (*opentype.Font, error) {
	loadedFacesMx.RLock()
	defer loadedFacesMx.RUnlock()

	minWeightDiff := 9999999.0
	var currentFace *loadedFontFace
	for _, f := range loadedFaces {
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
		currentFace = &loadedFaces[0] // first is the default face
	}

	return currentFace.font, nil
}

// Little dumb way to cache face for GetFontFace
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
	loadedFacesMx.Lock()
	defer loadedFacesMx.Unlock()

	if loadedFaces == nil {
		loadedFaces = make([]loadedFontFace, 0, len(faceTemplates)+1)
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
	cff := loadedFontFace{}
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

	if slices.IndexFunc(loadedFaces, func(f loadedFontFace) bool {
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

		loadedFaces = append(loadedFaces, cff)
	}

	return nil
}
