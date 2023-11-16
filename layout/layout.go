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
)

type layoutPhaseContext struct {
	size   utils.Size
	pos    utils.Pos
	props  CalculatedProperties
	isRoot bool
	drawer draw.Drawer
}

func Do(n parsing.Node, userData any, drawer draw.Drawer) []Node {
	nodes := doLayoutNode(n, layoutPhaseContext{
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
	}, userData)

	if len(nodes) == 0 {
		return nil
	}

	scale := n.GetScale()
	if scale != 1.0 {
		utils.ScaleAllValues(&nodes[0], scale)
	}

	prefetchResources(&nodes[0])

	return nodes
}

func doLayoutNode(n parsing.Node, context layoutPhaseContext, userData any) []Node {
	var layoutNodes []Node

	utils.RunForEach(userData, n.ForEach, func(value interface{}) {
		newContext := context

		var ln Node

		ln.Props = calculateProperties(n, context, value)
		newContext.props = ln.Props

		needSetNodeSize := false
		if ln.Props.Size.W == -1 {
			needSetNodeSize = true
		} else {
			ln.Size.W = ln.Props.Size.W
			newContext.size.W = ln.Props.Size.W
		}
		if ln.Props.Size.H == -1 {
			needSetNodeSize = true
		} else {
			ln.Size.H = ln.Props.Size.H
			newContext.size.H = ln.Props.Size.H
		}

		if newContext.isRoot {
			if ln.Size.W == 0 || ln.Size.H == 0 {
				return
			} else {
				context.drawer.InitImage(int(ln.Size.W*n.GetScale()), int(ln.Size.H*n.GetScale()))
			}
			newContext.isRoot = false
		}

		// retrieve child nodes

		var childNodes []Node
		var whiteSpaceNode Node

		if n.Image != "" {
			ln.Image = utils.ReplaceWithValues(n.Image, value)
			if needSetNodeSize {
				ln.Size = context.size
			}
		}

		var isText bool
		var text string
		if n.Text != "" {
			text = utils.ReplaceWithValues(n.Text, value)
			isText = text != ""
		}

		newContext.size.W -= ln.Props.Padding[1] + ln.Props.Padding[3]
		newContext.size.H -= ln.Props.Padding[0] + ln.Props.Padding[2]

		if isText {
			childNodes, whiteSpaceNode = spitTextToNodes(text, newContext)
		} else {
			for _, nc := range n.Inner {
				childNodes = append(childNodes, doLayoutNode(nc, newContext, value)...)
			}
		}

		if len(childNodes) > 0 {
			// process wrapping

			rows := make([][]*Node, 1)

			isDirectionRow := ln.Props.IsChildrenDirectionRow
			textWhitespaceWidth := lo.Ternary(isDirectionRow && isText, whiteSpaceNode.Size.W, 0)

			if isDirectionRow {
				var currentRowIndex int
				var currentWidth float64
				for icn, cn := range childNodes {
					if !cn.HasAnchors() {
						// if node Size is not set explicitly, so newContext.Size.W represents max width of node
						if ln.Props.IsWrappingEnabled && currentWidth+cn.Size.W > newContext.size.W && cn.Size.W < newContext.size.W {
							currentWidth = 0
							currentRowIndex += 1
							rows = append(rows, []*Node{})
						}
						currentWidth += cn.Size.W + lo.Ternary(cn.TextHasHyphenAtEnd, 0, textWhitespaceWidth) + ln.Props.InnerGap
					}
					rows[currentRowIndex] = append(rows[currentRowIndex], &childNodes[icn])
				}
			} else {
				for icn := range childNodes {
					rows[0] = append(rows[0], &childNodes[icn])
				}
			}

			// calculate node size

			if needSetNodeSize {
				var s float64
				for _, row := range rows {
					s = max(s, rowTotalSize(row, textWhitespaceWidth, isDirectionRow, ln.Props.InnerGap))

					if isDirectionRow {
						var rowMaxHeight float64
						for _, cn := range row {
							if !cn.HasAnchors() {
								rowMaxHeight = max(rowMaxHeight, cn.Size.H)
							}
						}
						ln.Size.H += rowMaxHeight
					} else if ln.Size.W == 0 {
						var maxWidth float64
						for _, cn := range row {
							if !cn.HasAnchors() {
								maxWidth = max(maxWidth, cn.Size.W)
							}
						}
						ln.Size.W = maxWidth

						// in columns, there are only one row, so add paddings to size right now
						ln.Size.W += ln.Props.Padding[1] + ln.Props.Padding[3]
					}
				}

				if isDirectionRow {
					ln.Size.H += ln.Props.InnerGap * float64(len(rows)-1)
					ln.Size.H += ln.Props.Padding[0] + ln.Props.Padding[2]
					if ln.Size.W == 0 {
						ln.Size.W = s
						ln.Size.W += ln.Props.Padding[1] + ln.Props.Padding[3]
					}
				} else {
					ln.Size.H = s
					ln.Size.H += ln.Props.Padding[0] + ln.Props.Padding[2]
				}
			}

			// process Justify

			var top float64
			for _, row := range rows {
				var offset float64
				var gap float64

				totalRowSize := rowTotalSize(row, textWhitespaceWidth, isDirectionRow, ln.Props.InnerGap)

				parentSize := lo.Ternary(isDirectionRow, ln.Size.W, ln.Size.H)
				if ln.Props.Justify == "center" {
					offset = parentSize/2 - totalRowSize/2
				} else if ln.Props.Justify == "end" {
					offset = parentSize - totalRowSize
				} else if ln.Props.Justify == "space-between" {
					gap = (parentSize - totalRowSize) / float64(len(row)-1)
				} else if ln.Props.Justify == "space-evenly" {
					gap = (parentSize - totalRowSize) / float64(len(row)+1)
					offset = gap
				}
				gap = max(gap, ln.Props.InnerGap)

				var maxHeight float64
				for icn, cn := range row {
					if cn.HasAnchors() {
						continue
					}

					if isDirectionRow {
						row[icn].Pos.Left = offset
						row[icn].Pos.Top = top
						offset += cn.Size.W + lo.Ternary(cn.TextHasHyphenAtEnd, 0, textWhitespaceWidth) + gap
						maxHeight = max(maxHeight, cn.Size.H)
					} else {
						row[icn].Pos.Top = offset
						offset += cn.Size.H + gap
					}
				}
				top += maxHeight + gap
			}

			// process vertical align

			if !isDirectionRow {
				for icn, cn := range rows[0] {
					if cn.HasAnchors() {
						continue
					}
					if ln.Props.ChildrenColumnAlign == "center" {
						rows[0][icn].Pos.Left = ln.Size.W/2 - cn.Size.W/2
					} else if ln.Props.ChildrenColumnAlign == "right" {
						rows[0][icn].Pos.Left = ln.Size.W - cn.Size.W
					}
				}
			}
		}

		ln.Children = append(ln.Children, childNodes...)
		layoutNodes = append(layoutNodes, ln)
	})

	return layoutNodes
}

func prefetchResources(n *Node) {
	if n.Image != "" {
		resources.PrefetchResource(n.Image)
	}
	for icn := range n.Children {
		prefetchResources(&n.Children[icn])
	}
}

func rowTotalSize(row []*Node, textWhitespaceWidth float64, isDirectionRow bool, gap float64) float64 {
	whiteSpaceCount := len(row) - 1
	sz := lo.Reduce(row, func(total float64, n *Node, index int) float64 {
		if n.HasAnchors() {
			return total
		}

		total += lo.Ternary(isDirectionRow, n.Size.W, n.Size.H)
		if n.TextHasHyphenAtEnd {
			whiteSpaceCount -= 1
		}
		return total
	}, 0)

	return sz + textWhitespaceWidth*float64(whiteSpaceCount) + float64(len(row)-1)*gap
}
