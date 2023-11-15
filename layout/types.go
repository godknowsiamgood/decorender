package layout

import (
	"github.com/godknowsiamgood/decorender/fonts"
	"github.com/godknowsiamgood/decorender/utils"
	"image/color"
)

type CalculatedProperties struct {
	Size                   utils.Size
	BkgColor               color.RGBA
	FontColor              color.RGBA
	FontDescription        fonts.FaceDescription
	ChildAlign             string
	IsChildrenDirectionRow bool
	Justify                string
	ChildrenColumnAlign    string
	IsWrappingEnabled      bool
	Padding                utils.FourValues
	LineHeight             float64
	BorderRadius           utils.FourValues
	Anchors                utils.Anchors
	InnerGap               float64
	Rotation               float64
}

type Node struct {
	Pos                utils.Pos
	Size               utils.Size
	Props              CalculatedProperties
	Text               string
	Image              string
	TextHasHyphenAtEnd bool
	Children           []Node
}

func (n *Node) HasAnchors() bool {
	return n.Props.Anchors.Has()
}
