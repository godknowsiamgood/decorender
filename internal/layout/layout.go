package layout

import (
	"fmt"
	"github.com/godknowsiamgood/decorender/internal/fonts"
	"github.com/godknowsiamgood/decorender/internal/parsing"
	"github.com/godknowsiamgood/decorender/internal/utils"
	"github.com/godknowsiamgood/decorender/resources"
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

	externalImage resources.ExternalImage
	cache         *Cache
	faces         *fonts.FaceSet
}

var nodesPool = sync.Pool{
	New: func() any {
		return make(Nodes, 0, 10)
	},
}

func Do(pn parsing.Node, userData any, externalImage resources.ExternalImage, cache *Cache, faces *fonts.FaceSet) (Nodes, error) {
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
		level:         -1,
		externalImage: externalImage,
		cache:         cache,
		faces:         faces,
	}, userData, nil, 0)

	if err != nil {
		return nil, err
	}

	root := nodes.GetRootNode()
	if root == nil {
		return nil, fmt.Errorf("no nodes to render")
	}

	if pn.GetScale() != 1.0 {
		nodes.IterateNodes(func(node *Node) {
			ScaleAllValues(node, pn.GetScale())
		})
	}

	return nodes, nil
}

func Release(nodes Nodes) {
	nodes = nodes[0:0]
	nodesPool.Put(nodes)
}

func doLayoutNode(pn parsing.Node, nodes *Nodes, context layoutPhaseContext, value any, parentValue any, currentValueIndex int) error {
	nodeLevel := context.level + 1

	var forEach any = pn.ForEach
	if IsExpression(pn.ForEach) {
		evaluated, err := evaluate(pn.ForEach, value, parentValue, currentValueIndex, context.cache)
		if err != nil {
			return fmt.Errorf("forEach %q: %w", pn.ForEach, err)
		}
		forEach = evaluated
	}

	return RunForEach(value, forEach, currentValueIndex, func(currentValue any, iteratorValue any, currentValueIndex int) error {
		if iteratorValue == nil {
			iteratorValue = parentValue
		}

		props, err := calculateProperties(pn, context, currentValue, iteratorValue, currentValueIndex)
		if err != nil {
			return fmt.Errorf("%s: %w", nodeRef(pn), err)
		}

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
		newContext.size.H -= props.Padding.Top() + props.Padding.Bottom()

		// Retrieve child nodes

		var textWhitespaceWidth float64

		var text string
		if pn.Text != "" {
			text, err = replaceWithValues(pn.Text, currentValue, iteratorValue, currentValueIndex, context.cache)
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
				if err = doLayoutNode(pn.Inner[i], nodes, newContext, currentValue, iteratorValue, currentValueIndex); err != nil {
					return err
				}
			}
		}

		childrenNodesLevel := nodeLevel + 1

		childCount := 0
		nodes.IterateChildNodes(childrenNodesLevel, from, func(cn *Node) {
			if !cn.IsAbsolutePositioned() {
				childCount += 1
			}
		})

		// Apply wrapping and aligning. All of this can be applied only for not absolute positioned elements
		if childCount > 0 {
			isDirectionRow := props.IsChildrenDirectionRow

			// do child wrapping
			if isDirectionRow {
				var currentRowIndex int
				var currentInRowIndex int
				var currentWidth float64

				var prevNodeInRow *Node
				nodes.IterateChildNodes(childrenNodesLevel, from, func(node *Node) {
					if !node.IsAbsolutePositioned() {
						if props.IsWrappingEnabled && currentWidth+node.Size.W > newContext.size.W {
							currentWidth = 0
							currentRowIndex += 1
							currentInRowIndex = 0

							// Maybe we can wrap whole-hyphened word to look it better
							if prevNodeInRow != nil && prevNodeInRow.JoinsNextToken {
								wholeWidth := prevNodeInRow.Size.W + node.Size.W
								if wholeWidth <= newContext.size.W {
									prevNodeInRow.InRowIndex = 0
									prevNodeInRow.RowIndex = currentRowIndex
									currentInRowIndex = 1
								}
							}

							prevNodeInRow = nil
						}
						currentWidth += node.Size.W + whitespaceAfter(node, textWhitespaceWidth) + props.InnerColumnGap
					}

					node.RowIndex = currentRowIndex
					node.InRowIndex = currentInRowIndex
					currentInRowIndex += 1
					prevNodeInRow = node
				})

				if text != "" {
					mergeTextNodes(nodes, childrenNodesLevel, from, textWhitespaceWidth)
				}
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
					totalRowSize, countInRow := nodes.RowTotalWidth(childrenNodesLevel, from, rowIndex, textWhitespaceWidth, props.InnerColumnGap)
					offset, gap := getJustifyOffsetAndGap(props.Justify, props.InnerColumnGap, totalRowSize, newContext.size.W, countInRow)

					var maxHeight float64
					nodes.IterateRow(childrenNodesLevel, from, rowIndex, func(cn *Node) {
						if cn.IsAbsolutePositioned() {
							return
						}
						cn.Pos.Left = offset
						cn.Pos.Top = top
						offset += cn.Size.W + whitespaceAfter(cn, textWhitespaceWidth) + gap
						maxHeight = math.Max(maxHeight, cn.Size.H)
					})

					// Children of a row are laid out along the top edge; this
					// moves them within the row once its height is known.
					if props.ChildrenRowAlign != "top" {
						nodes.IterateRow(childrenNodesLevel, from, rowIndex, func(cn *Node) {
							if cn.IsAbsolutePositioned() {
								return
							}
							free := maxHeight - cn.Size.H
							if props.ChildrenRowAlign == "center" {
								free /= 2
							}
							cn.Pos.Top = top + free
						})
					}

					// Rows stack by the row gap the layout asked for. The
					// gap getJustifyOffsetAndGap returns is the horizontal one
					// it spread the row with, which under space-between grows
					// with the free space left on the line.
					top += maxHeight + props.InnerRowGap
				})
			} else {
				totalHeight, count := nodes.RowsTotalHeight(childrenNodesLevel, from, props.InnerRowGap)
				offset, gap := getJustifyOffsetAndGap(props.Justify, props.InnerRowGap, totalHeight, newContext.size.H, count)
				nodes.IterateRows(childrenNodesLevel, from, func(_ int, node *Node) {
					if node.IsAbsolutePositioned() {
						return
					}
					node.Pos.Top = offset
					offset += node.Size.H + gap
				})
			}

			// do horizontal align for column children

			if !isDirectionRow {
				// Every child of a column is its own row, so this must walk all
				// children: iterating row 0 alone would align only the first.
				nodes.IterateChildNodes(childrenNodesLevel, from, func(cn *Node) {
					if cn.IsAbsolutePositioned() {
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
				rowWidth, _ := nodes.RowTotalWidth(childrenNodesLevel, from, rowIndex, textWhitespaceWidth, props.InnerColumnGap)
				props.Size.W = math.Max(props.Size.W, rowWidth)
			})
			props.Size.W = math.Max(0, props.Size.W+props.Padding.Left()+props.Padding.Right())
		}

		if props.Size.H == -1 {
			height, _ := nodes.RowsTotalHeight(childrenNodesLevel, from, props.InnerRowGap)
			props.Size.H = math.Max(0, height+props.Padding.Top()+props.Padding.Bottom())
		}

		applyAbsolutePositions(nodes, childrenNodesLevel, from, &props)

		// Finally, apply offsets
		nodes.IterateChildNodes(childrenNodesLevel, from, func(cn *Node) {
			cn.Pos.Left += cn.Props.Offset.Left()
			cn.Pos.Top += cn.Props.Offset.Top()
		})

		imageVal, err := replaceWithValues(pn.Image, currentValue, iteratorValue, currentValueIndex, context.cache)
		if err != nil {
			return err
		}

		ln := Node{
			Id:    pn.Id,
			Size:  props.Size,
			Props: props,
			Image: imageVal,
			Level: nodeLevel,
			// Pos is not set here, because parent is responsible for doing this
		}

		if ln.Image != "" {
			context.externalImage.Prefetch(ln.Image)
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
		// A lone child has no space between anything, and dividing by the
		// gaps it does not have gives an infinity that spreads through every
		// position derived from it.
		if count > 1 {
			gap = (parentSize - totalSize) / float64(count-1)
		}
	case "space-evenly":
		gap = (parentSize - totalSize) / float64(count+1)
		offset = gap
	}
	gap = math.Max(gap, gapProp)
	return offset, gap
}

// whitespaceAfter is the gap that follows a node on its row. A token the next
// one joins directly contributes no whitespace.
func whitespaceAfter(n *Node, textWhitespaceWidth float64) float64 {
	if n.JoinsNextToken {
		return 0
	}
	return textWhitespaceWidth
}
