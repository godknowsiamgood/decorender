package layout

import (
	"github.com/godknowsiamgood/decorender/draw"
	"github.com/godknowsiamgood/decorender/fonts"
	"github.com/godknowsiamgood/decorender/parsing"
	"github.com/godknowsiamgood/decorender/resources"
	"github.com/godknowsiamgood/decorender/utils"
	"github.com/samber/lo"
	"golang.org/x/image/font"
	"image/color"
	"sync"
)

type layoutPhaseContext struct {
	size     utils.Size
	pos      utils.Pos
	props    CalculatedProperties
	isRoot   bool
	drawer   draw.Drawer
	parentId int
}

var nodesPool = sync.Pool{
	New: func() any {
		return make(Nodes, 0, 10)
	},
}

func Do(pn parsing.Node, userData any, drawer draw.Drawer) Nodes {
	nodes := nodesPool.Get().(Nodes)

	doLayoutNode(pn, &nodes, layoutPhaseContext{
		size: utils.Size{},
		props: CalculatedProperties{
			FontColor:  color.RGBA{A: 0},
			LineHeight: -1,
			FontDescription: fonts.FaceDescription{
				Family: "Roboto",
				Size:   23,
				Weight: 400,
				Style:  font.StyleNormal,
			},
		},
		isRoot: true,
		drawer: drawer,
	}, userData, nil)

	root := nodes.GetRootNode()
	if root == nil {
		return nil
	}

	if pn.GetScale() != 1.0 {
		utils.ScaleAllValues(&nodes[0], pn.GetScale())
	}

	return nodes
}

func Release(nodes Nodes) {
	nodes = nodes[0:0]
	nodesPool.Put(nodes)
}

func doLayoutNode(pn parsing.Node, nodes *Nodes, context layoutPhaseContext, value any, parentValue any) {
	parentId := context.parentId + 1

	utils.RunForEach(value, utils.ReplaceWithValues(pn.ForEach, value, parentValue), func(currentValue any, iteratorValue any) {
		if iteratorValue == nil {
			iteratorValue = parentValue
		}

		props := calculateProperties(pn, context, currentValue, iteratorValue)

		newContext := context
		newContext.props = props
		newContext.parentId = parentId

		// Setup context size

		if props.Size.W != -1 {
			newContext.size.W = props.Size.W
		}
		newContext.size.W -= props.Padding[1] + props.Padding[3]

		if props.Size.H != -1 {
			newContext.size.H = props.Size.H
		}
		newContext.size.H -= props.Padding[0] + props.Padding[2]

		// Special case for root: size should be set explicitly
		if newContext.isRoot {
			if props.Size.W == 0 || props.Size.H == 0 {
				return
			} else {
				context.drawer.InitImage(int(props.Size.W*pn.GetScale()), int(props.Size.H*pn.GetScale()))
			}
			newContext.isRoot = false
		}

		// Retrieve child nodes

		var textWhitespaceWidth float64

		var text string
		if pn.Text != "" {
			text = utils.ReplaceWithValues(pn.Text, currentValue, iteratorValue)
		}

		from := len(*nodes)

		if text != "" {
			textWhitespaceWidth = spitTextToNodes(nodes, text, newContext)
		} else {
			for _, nc := range pn.Inner {
				doLayoutNode(nc, nodes, newContext, currentValue, iteratorValue)
			}
		}

		hasMoreThanOneChild := len(*nodes)-from > 1
		isDirectionRow := props.IsChildrenDirectionRow

		if hasMoreThanOneChild {

			// do child wrapping

			if isDirectionRow {
				var currentRowIndex int
				var currentInRowIndex int
				var currentWidth float64

				var prevInRow *Node
				nodes.IterateChildNodes(parentId, func(cn *Node) {
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
				nodes.IterateChildNodes(parentId, func(cn *Node) {
					cn.RowIndex = i
					i++
				})
			}

			// do justify and vertical position for rows

			if props.IsChildrenDirectionRow {
				var top float64
				nodes.IterateRows(parentId, func(rowIndex int, _ *Node) {
					totalRowSize, countInRow := nodes.RowTotalWidth(parentId, rowIndex, textWhitespaceWidth, props.InnerGap)
					offset, gap := getJustifyOffsetAndGap(props.Justify, props.InnerGap, totalRowSize, newContext.size.W, countInRow)

					var maxHeight float64
					nodes.IterateRow(parentId, rowIndex, func(cn *Node) {
						if cn.HasAnchors() {
							return
						}
						cn.Pos.Left = offset
						cn.Pos.Top = top
						offset += cn.Size.W + lo.Ternary(cn.TextHasHyphenAtEnd, 0, textWhitespaceWidth) + gap
					})

					top += maxHeight + gap
				})
			} else {
				totalHeight, count := nodes.RowsTotalHeight(parentId, props.InnerGap)
				offset, gap := getJustifyOffsetAndGap(props.Justify, props.InnerGap, totalHeight, newContext.size.H, count)
				nodes.IterateRows(parentId, func(_ int, node *Node) {
					if node.HasAnchors() {
						return
					}
					node.Pos.Top = offset
					offset += node.Size.H + gap
				})
			}

			// do horizontal align for column children

			if !isDirectionRow {
				nodes.IterateRow(parentId, 0, func(cn *Node) {
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
			props.Size.W = 0
			nodes.IterateRows(parentId, func(rowIndex int, _ *Node) {
				rowWidth, _ := nodes.RowTotalWidth(parentId, rowIndex, textWhitespaceWidth, props.InnerGap)
				props.Size.W = max(props.Size.W, rowWidth)
			})
			props.Size.W += props.Padding[1] + props.Padding[3]
		}

		if props.Size.H == -1 {
			height, _ := nodes.RowsTotalHeight(parentId, props.InnerGap)
			props.Size.H += height + props.Padding[0] + props.Padding[2]
		}

		ln := Node{
			Id:       pn.Id,
			Size:     props.Size,
			Props:    props,
			Text:     text,
			Image:    utils.ReplaceWithValues(pn.Image, currentValue, iteratorValue),
			ParentId: context.parentId,
		}

		if ln.Image != "" {
			resources.PrefetchResource(ln.Image)
		}

		*nodes = append(*nodes, ln)
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
	gap = max(gap, gapProp)
	return offset, gap
}
