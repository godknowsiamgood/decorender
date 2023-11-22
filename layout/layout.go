package layout

import (
	"fmt"
	"github.com/godknowsiamgood/decorender/fonts"
	"github.com/godknowsiamgood/decorender/parsing"
	"github.com/godknowsiamgood/decorender/resources"
	"github.com/godknowsiamgood/decorender/utils"
	"github.com/samber/lo"
	"golang.org/x/image/font"
	"image/color"
	"math"
	"sync"
)

type layoutPhaseContext struct {
	size  utils.Size
	pos   utils.Pos
	props CalculatedProperties
	level int
}

var nodesPool = sync.Pool{
	New: func() any {
		return make(Nodes, 0, 10)
	},
}

func Do(pn parsing.Node, userData any) (Nodes, error) {
	nodes := nodesPool.Get().(Nodes)

	err := doLayoutNode(pn, &nodes, layoutPhaseContext{
		size: utils.Size{},
		props: CalculatedProperties{
			FontColor:  color.RGBA{A: 255},
			LineHeight: -1,
			FontDescription: fonts.FaceDescription{
				Family: fonts.DefaultFamily,
				Size:   16,
				Weight: 400,
				Style:  font.StyleNormal,
			},
		},
		level: -1,
	}, userData, nil)

	if err != nil {
		return nil, err
	}

	root := nodes.GetRootNode()
	if root == nil {
		return nil, fmt.Errorf("no nodes to render")
	}

	if pn.GetScale() != 1.0 {
		nodes.IterateNodes(func(node *Node) {
			utils.ScaleAllValues(node, pn.GetScale())
		})
	}

	return nodes, nil
}

func Release(nodes Nodes) {
	nodes = nodes[0:0]
	nodesPool.Put(nodes)
}

func doLayoutNode(pn parsing.Node, nodes *Nodes, context layoutPhaseContext, value any, parentValue any) error {
	nodeLevel := context.level + 1

	forEach, err := utils.ReplaceWithValues(pn.ForEach, value, parentValue)
	if err != nil {
		return err
	}

	return utils.RunForEach(value, forEach, func(currentValue any, iteratorValue any) error {
		if iteratorValue == nil {
			iteratorValue = parentValue
		}

		props := calculateProperties(pn, context, currentValue, iteratorValue)

		newContext := context
		newContext.props = props
		newContext.level = nodeLevel

		// Setup context size

		if props.Size.W != -1 {
			newContext.size.W = props.Size.W
		}
		newContext.size.W -= props.Padding.Right() + props.Padding.Left()

		if props.Size.H != -1 {
			newContext.size.H = props.Size.H
		}
		newContext.size.H -= props.Padding.Left() + props.Padding.Bottom()

		// Retrieve child nodes

		var textWhitespaceWidth float64

		var text string
		if pn.Text != "" {
			text, err = utils.ReplaceWithValues(pn.Text, currentValue, iteratorValue)
			if err != nil {
				return err
			}
		}

		from := len(*nodes)

		// All nodes are stored in linear slice for efficiency,
		// and for traversing reasons later at render phase,
		// all children in slice are in reverse order.

		if text != "" {
			textWhitespaceWidth = spitTextToNodes(nodes, text, newContext)
		} else {
			for i := len(pn.Inner) - 1; i >= 0; i-- {
				if err = doLayoutNode(pn.Inner[i], nodes, newContext, currentValue, iteratorValue); err != nil {
					return err
				}
			}
		}

		childrenNodesLevel := nodeLevel + 1

		childCount := 0
		nodes.IterateChildNodes(childrenNodesLevel, from, func(_ *Node) {
			childCount += 1
		})

		isDirectionRow := props.IsChildrenDirectionRow

		if childCount > 1 {

			// do child wrapping
			if isDirectionRow {
				var currentRowIndex int
				var currentInRowIndex int
				var currentWidth float64

				var prevInRow *Node
				nodes.IterateChildNodes(childrenNodesLevel, from, func(cn *Node) {
					if !cn.HasAnchors() {
						if props.IsWrappingEnabled && currentWidth+cn.Size.W > newContext.size.W && cn.Size.W < newContext.size.W {
							currentWidth = 0
							currentRowIndex += 1
							currentInRowIndex = 0

							// Maybe we can wrap whole-hyphened word to look it better
							if prevInRow != nil && prevInRow.TextHasHyphenAtEnd {
								wholeWidth := prevInRow.Size.W + cn.Size.W
								if wholeWidth <= newContext.size.W {
									prevInRow.InRowIndex = currentRowIndex
									prevInRow.RowIndex = 0
									currentInRowIndex = 1
								}
							}

							prevInRow = nil
						}
						currentWidth += cn.Size.W + lo.Ternary(cn.TextHasHyphenAtEnd, 0, textWhitespaceWidth) + props.InnerGap
					}

					cn.RowIndex = currentRowIndex
					cn.InRowIndex = currentInRowIndex
					currentInRowIndex += 1
					prevInRow = cn
				})
			} else {
				i := 0
				nodes.IterateChildNodes(childrenNodesLevel, from, func(cn *Node) {
					cn.RowIndex = i
					i++
				})
			}

			// do justify and vertical position for rows

			if props.IsChildrenDirectionRow {
				var top float64
				nodes.IterateRows(childrenNodesLevel, from, func(rowIndex int, _ *Node) {
					totalRowSize, countInRow := nodes.RowTotalWidth(childrenNodesLevel, from, rowIndex, textWhitespaceWidth, props.InnerGap)
					offset, gap := getJustifyOffsetAndGap(props.Justify, props.InnerGap, totalRowSize, newContext.size.W, countInRow)

					var maxHeight float64
					nodes.IterateRow(childrenNodesLevel, from, rowIndex, func(cn *Node) {
						if cn.HasAnchors() {
							return
						}
						cn.Pos.Left = offset
						cn.Pos.Top = top
						offset += cn.Size.W + lo.Ternary(cn.TextHasHyphenAtEnd, 0, textWhitespaceWidth) + gap
						maxHeight = math.Max(maxHeight, cn.Size.H)
					})

					top += maxHeight + gap
				})
			} else {
				totalHeight, count := nodes.RowsTotalHeight(childrenNodesLevel, from, props.InnerGap)
				offset, gap := getJustifyOffsetAndGap(props.Justify, props.InnerGap, totalHeight, newContext.size.H, count)
				nodes.IterateRows(childrenNodesLevel, from, func(_ int, node *Node) {
					if node.HasAnchors() {
						return
					}
					node.Pos.Top = offset
					offset += node.Size.H + gap
				})
			}

			// do horizontal align for column children

			if !isDirectionRow {
				nodes.IterateRow(childrenNodesLevel, from, 0, func(cn *Node) {
					if cn.HasAnchors() {
						return
					}

					if props.ChildrenColumnAlign == "center" {
						cn.Pos.Left = newContext.size.W/2 - cn.Size.W/2
					} else if props.ChildrenColumnAlign == "right" {
						cn.Pos.Left = newContext.size.W - cn.Size.W
					}
				})
			}
		}

		if props.Size.W == -1 {
			nodes.IterateRows(childrenNodesLevel, from, func(rowIndex int, _ *Node) {
				rowWidth, _ := nodes.RowTotalWidth(childrenNodesLevel, from, rowIndex, textWhitespaceWidth, props.InnerGap)
				props.Size.W = math.Max(props.Size.W, rowWidth)
			})
			props.Size.W = math.Max(0, props.Size.W+props.Padding.Right()+props.Padding.Bottom())
		}

		if props.Size.H == -1 {
			height, _ := nodes.RowsTotalHeight(childrenNodesLevel, from, props.InnerGap)
			props.Size.H = math.Max(0, height+props.Padding.Top()+props.Padding.Bottom())
		}

		imageVal, err := utils.ReplaceWithValues(pn.Image, currentValue, iteratorValue)
		if err != nil {
			return err
		}

		ln := Node{
			Id:    pn.Id,
			Size:  props.Size,
			Props: props,
			Image: imageVal,
			Level: nodeLevel,
		}

		if ln.Image != "" {
			resources.PrefetchResource(ln.Image)
		}

		*nodes = append(*nodes, ln)

		return nil
	})
}

func getJustifyOffsetAndGap(justifyProp string, gapProp float64, totalSize float64, parentSize float64, count int) (offset float64, gap float64) {
	switch justifyProp {
	case "center":
		offset = parentSize/2 - totalSize/2
	case "end":
		offset = parentSize - totalSize
	case "space-between":
		gap = (parentSize - totalSize) / float64(count-1)
	case "space-evenly":
		gap = (parentSize - totalSize) / float64(count+1)
		offset = gap
	}
	gap = math.Max(gap, gapProp)
	return offset, gap
}
