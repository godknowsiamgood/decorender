package parsing

import "strconv"

type Node struct {
	Id                  string     `yaml:"id"`
	Size                string     `yaml:"size"`
	Width               string     `yaml:"width"`
	Height              string     `yaml:"height"`
	Absolute            string     `yaml:"absolute"`
	BkgColor            string     `yaml:"bkgColor"`
	LineHeight          string     `yaml:"lineHeight"`
	InnerDirection      string     `yaml:"innerDirection"`
	Justify             string     `yaml:"justify"`
	ChildrenColumnAlign string     `yaml:"childrenColumnAlign"`
	ChildrenWrap        string     `yaml:"childrenWrap"`
	Padding             string     `yaml:"padding"`
	Text                string     `yaml:"text"`
	Image               string     `yaml:"image"`
	FontFaces           []FontFace `yaml:"fontFaces"`
	Font                string     `yaml:"font"`
	FontFamily          string     `yaml:"fontFamily"`
	FontSize            string     `yaml:"fontSize"`
	FontWeight          string     `yaml:"fontWeight"`
	FontStyle           string     `yaml:"fontStyle"`
	FontColor           string     `yaml:"fontColor"`
	Color               string     `yaml:"color"` // same as fontColor
	BorderRadius        string     `yaml:"borderRadius"`
	InnerGap            string     `yaml:"innerGap"`
	Rotation            string     `yaml:"rotate"`
	DebugOnly           string     `yaml:"only"`
	BkgImageSize        string     `yaml:"bkgImageSize"`
	Border              string     `yaml:"border"`
	Scale               string     `yaml:"scale"`

	ForEach string `yaml:"forEach"`
	Inner   []Node `yaml:"inner"`
}

func (n *Node) GetScale() float64 {
	scale, err := strconv.ParseFloat(n.Scale, 64)
	if err != nil || scale < 0.001 {
		scale = 1.0
	}
	return scale
}

type FontFace struct {
	Family string `yaml:"family"`
	Style  string `yaml:"style"`
	Weight string `yaml:"weight"`
	File   string `yaml:"file"`
}
