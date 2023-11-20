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
	BkgImageSize           string
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
	ParentId           int

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

func (nodes Nodes) IterateRows(parentId int, cb func(rowIndex int, node *Node)) {
	rowIndex := -1
	for i := range nodes {
		n := &nodes[i]
		if parentId == n.ParentId && rowIndex != n.RowIndex {
			rowIndex = n.RowIndex
			cb(rowIndex, n)
		}
	}
}

func (nodes Nodes) IterateRow(parentId int, rowIndex int, cb func(cn *Node)) {
	for i := range nodes {
		if nodes[i].ParentId == parentId && rowIndex == nodes[i].RowIndex {
			cb(&nodes[i])
		}
	}
}

func (nodes Nodes) IterateChildNodes(parentId int, cb func(cn *Node)) {
	for i := range nodes {
		if nodes[i].ParentId == parentId {
			cb(&nodes[i])
		}
	}
}

func (nodes Nodes) RowsTotalHeight(parentId int, gap float64) (height float64, count int) {
	nodes.IterateRows(parentId, func(rowIndex int, node *Node) {
		if node.HasAnchors() {
			return
		}
		count += 1
		height += node.Size.H
	})
	return height + float64(count-1)*gap, count
}

func (nodes Nodes) RowTotalWidth(parentId int, rowIndex int, textWhitespaceWidth float64, gap float64) (float64, int) {
	var total float64

	hyphensCount := 0
	count := 0
	nodes.IterateRow(parentId, rowIndex, func(cn *Node) {
		if cn.HasAnchors() {
			return
		}

		total += cn.Size.W
		if cn.TextHasHyphenAtEnd {
			hyphensCount += 1
		}
		count += 1
	})

	return total + textWhitespaceWidth*float64(count-hyphensCount) + gap*float64(count-1), count
}

func (nodes Nodes) IterateDepthFirst(cb func(node *Node)) {
	for i := len(nodes) - 1; i >= 0; i-- {
		cb(&nodes[i])
	}
}
