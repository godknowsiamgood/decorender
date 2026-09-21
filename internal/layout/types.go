package layout

import (
	"github.com/godknowsiamgood/decorender/internal/fonts"
	"github.com/godknowsiamgood/decorender/internal/utils"
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
	ChildrenRowAlign       string
	IsWrappingEnabled      bool
	Padding                utils.TopRightBottomLeft
	LineHeight             float64
	BorderRadius           utils.FourValues
	AbsolutePosition       utils.AbsolutePosition
	InnerRowGap            float64
	InnerColumnGap         float64
	Rotation               float64
	BkgImageSize           BkgImageSizeType
	Border                 utils.Border
	Offset                 utils.TopRightBottomLeft
}

// Node represents positioned and prepared element to render after layout phase
type Node struct {
	Id string

	Pos   utils.Pos
	Size  utils.Size
	Props CalculatedProperties
	Text  string
	Image string
	// JoinsNextToken marks a text token that the following token follows
	// directly, with no whitespace between them: the halves of a hyphenated
	// word, or of one split at a zero-width break.
	JoinsNextToken bool
	Level          int

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

// RowsTotalHeight measures rows by their tallest child.
//
// It used to take the height of whichever child came first in the row, while
// positioning advances to the next row by the tallest one. A row whose first
// child was the shortest therefore reported a height smaller than what it
// draws, and a container sized around it clipped its own content.
func (nodes Nodes) RowsTotalHeight(level int, from int, gap float64) (height float64, count int) {
	rowIndex := -1
	var rowHeight float64

	for i := len(nodes) - 1; i >= from; i-- {
		n := &nodes[i]
		if n.Level != level || n.IsAbsolutePositioned() {
			continue
		}

		if rowIndex != n.RowIndex {
			height += rowHeight
			rowHeight = 0
			rowIndex = n.RowIndex
			count += 1
		}

		if n.Size.H > rowHeight {
			rowHeight = n.Size.H
		}
	}
	height += rowHeight

	if count == 0 {
		return 0, 0
	}

	return height + float64(count-1)*gap, count
}

func (nodes Nodes) RowTotalWidth(level int, from int, rowIndex int, textWhitespaceWidth float64, gap float64) (float64, int) {
	var total float64
	var whitespace float64

	// Whitespace goes between nodes, so it is counted when the next node
	// arrives rather than after each one. Counting it per node instead would
	// charge the row for a space after its last node, which the earlier form
	// of this did: a row ending in a joining token - a hyphenated fragment, or
	// one split at a zero-width break - came out a space too narrow.
	count := 0
	var previous *Node
	nodes.IterateRow(level, from, rowIndex, func(cn *Node) {
		if cn.IsAbsolutePositioned() {
			return
		}

		if previous != nil && !previous.JoinsNextToken {
			whitespace += textWhitespaceWidth
		}

		total += cn.Size.W
		count += 1
		previous = cn
	})

	if count == 0 {
		return 0, 0
	}

	return total + whitespace + gap*float64(count-1), count
}
