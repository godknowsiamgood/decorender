package layout

import (
	"github.com/godknowsiamgood/decorender/internal/fonts"
	"github.com/godknowsiamgood/decorender/internal/utils"
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
	AbsolutePosition       utils.AbsolutePosition
	InnerGap               float64
	Rotation               float64
	BkgImageSize           BkgImageSizeType
	Border                 utils.Border
	Offset                 utils.TopRightBottomLeft
}

// Node represents positioned and prepared element to render after layout phase
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

func (n *Node) IsAbsolutePositioned() bool {
	return n.Props.AbsolutePosition.Has()
}

// Nodes represents hierarchy for nodes. It uses linear slice for efficiency.
// E.g. if A has children B and C, and C has child D
// they will be located in this slice in order DCBA
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

// IterateRows here and below are helpers to iterate through children.
// level is level of children and from is a hint where iteration should end.
// Iteration can be performed only for children that added recently.
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

func (nodes Nodes) IterateRowsReverse(level int, from int, cb func(rowIndex int)) {
	rowIndex := -1
	for i := from; i < len(nodes); i++ {
		n := &nodes[i]
		if level == n.Level && rowIndex != n.RowIndex {
			rowIndex = n.RowIndex
			cb(rowIndex)
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
		if node.IsAbsolutePositioned() {
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
		if cn.IsAbsolutePositioned() {
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
