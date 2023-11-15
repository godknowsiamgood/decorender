package fonts

import (
	"decorender/parsing"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"math"
	"os"
	"slices"
	"strconv"
	"sync"
)

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

func GetFont(fd FaceDescription) *opentype.Font {
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
		currentFace = &faces[0] // first is the default face
	}

	return currentFace.font
}

func GetFontFace(fd FaceDescription) font.Face {
	f := GetFont(fd)
	face, _ := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    fd.Size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	return face
}

func LoadFaces(faceTemplates []parsing.FontFace) error {
	facesMx.Lock()
	defer facesMx.Unlock()

	if faces == nil {
		faces = make([]cachedFontFace, 0, len(faceTemplates)+1)
	}

	loadFont(parsing.FontFace{
		Family: "default",
		Style:  "normal",
		Weight: "400",
		File:   "./Roboto-Regular.ttf",
	})

	for _, ft := range faceTemplates {
		loadFont(ft)
	}

	return nil
}

func loadFont(faceTemplate parsing.FontFace) {
	cff := cachedFontFace{}
	if faceTemplate.Style != "" {
		switch faceTemplate.Style {
		case "normal":
			cff.style = font.StyleNormal
		case "italic":
			cff.style = font.StyleItalic
		default:
			return
		}
	} else {
		cff.style = font.StyleNormal
	}

	if faceTemplate.Family == "" {
		return
	}
	cff.family = faceTemplate.Family

	if faceTemplate.Weight != "" {
		weight, err := strconv.Atoi(faceTemplate.Weight)
		if err != nil {
			return
		}
		cff.weight = weight
	}

	cff.uri = faceTemplate.File

	if slices.IndexFunc(faces, func(f cachedFontFace) bool {
		return f.family == cff.family && f.style == cff.style && f.weight == cff.weight
	}) == -1 {
		data, err := os.ReadFile(faceTemplate.File)
		if err != nil {
			return
		}

		fnt, err := opentype.Parse(data)
		if err != nil {
			return
		}

		cff.font = fnt

		faces = append(faces, cff)
	}
}
