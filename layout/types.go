package layout

import (
	"github.com/godknowsiamgood/decorender/fonts"
	"github.com/godknowsiamgood/decorender/utils"
	"golang.org/x/image/font"
	"image/color"
)

type BkgImageSizeType int

const (
	BkgImageSizeCover BkgImageSizeType = iota
	BkgImageSizeContain
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
	Padding                utils.TopRightBottomLeft
	LineHeight             float64
	BorderRadius           utils.FourValues
	Anchors                utils.Anchors
	InnerGap               float64
	Rotation               float64
	BkgImageSize           BkgImageSizeType
	Border                 utils.Border
}

type Node struct {
	Id string

	Pos                utils.Pos
	Size               utils.Size
	Props              CalculatedProperties
	Text               string
	Image              string
	TextHasHyphenAtEnd bool
	Level              int
	Face               font.Face

	RowIndex   int
	InRowIndex int
}

func (n *Node) HasAnchors() bool {
	return n.Props.Anchors.Has()
}

type Nodes []Node

func (nodes Nodes) GetRootNode() *Node {
	if len(nodes) == 0 {
		return nil
	} else {
		return &nodes[len(nodes)-1]
	}
}

func (nodes Nodes) IterateNodes(cb func(node *Node)) {
	for i := range nodes {
		cb(&nodes[i])
	}
}

func (nodes Nodes) IterateRows(level int, from int, cb func(rowIndex int, firstInRowNode *Node)) {
	rowIndex := -1
	for i := len(nodes) - 1; i >= from; i-- {
		n := &nodes[i]
		if level == n.Level && rowIndex != n.RowIndex {
			rowIndex = n.RowIndex
			cb(rowIndex, n)
		}
	}
}

func (nodes Nodes) IterateRow(level int, from int, rowIndex int, cb func(cn *Node)) {
	for i := len(nodes) - 1; i >= from; i-- {
		if nodes[i].Level == level && rowIndex == nodes[i].RowIndex {
			cb(&nodes[i])
		}
	}
}

func (nodes Nodes) IterateChildNodes(level int, from int, cb func(cn *Node)) {
	for i := len(nodes) - 1; i >= from; i-- {
		if nodes[i].Level == level {
			cb(&nodes[i])
		}
	}
}

func (nodes Nodes) RowsTotalHeight(level int, from int, gap float64) (height float64, count int) {
	nodes.IterateRows(level, from, func(rowIndex int, node *Node) {
		if node.HasAnchors() {
			return
		}
		count += 1
		height += node.Size.H
	})
	return height + float64(count-1)*gap, count
}

func (nodes Nodes) RowTotalWidth(level int, from int, rowIndex int, textWhitespaceWidth float64, gap float64) (float64, int) {
	var total float64

	hyphensCount := 0
	count := 0
	nodes.IterateRow(level, from, rowIndex, func(cn *Node) {
		if cn.HasAnchors() {
			return
		}

		total += cn.Size.W
		if cn.TextHasHyphenAtEnd {
			hyphensCount += 1
		}
		count += 1
	})

	return total + textWhitespaceWidth*float64(count-hyphensCount-1) + gap*float64(count-1), count
}
