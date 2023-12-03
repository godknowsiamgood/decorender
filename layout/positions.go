package layout

import "github.com/godknowsiamgood/decorender/utils"

func applyAbsolutePositions(nodes *Nodes, childrenNodesLevel int, from int, parentSize utils.Size) {
	nodes.IterateChildNodes(childrenNodesLevel, from, func(cn *Node) {
		if !cn.IsAbsolutePositioned() {
			return
		}

		var left float64
		var top float64

		if cn.Props.AbsolutePosition.HasLeft() || cn.Props.AbsolutePosition.HasRight() {
			if !cn.Props.AbsolutePosition.HasTop() && !cn.Props.AbsolutePosition.HasBottom() {
				top = parentSize.H/2 - cn.Size.H/2
			}
			if cn.Props.AbsolutePosition.HasRight() {
				left = parentSize.W - cn.Size.W - cn.Props.AbsolutePosition.Right()
			} else {
				left = cn.Props.AbsolutePosition.Left()
			}
		}
		if cn.Props.AbsolutePosition.HasTop() || cn.Props.AbsolutePosition.HasBottom() {
			if !cn.Props.AbsolutePosition.HasLeft() && !cn.Props.AbsolutePosition.HasRight() {
				left = parentSize.W/2 - cn.Size.W/2
			}
			if cn.Props.AbsolutePosition.HasBottom() {
				top = parentSize.H - cn.Size.H - cn.Props.AbsolutePosition.Bottom()
			} else {
				top = cn.Props.AbsolutePosition.Top()
			}
		}

		cn.Pos.Left = left
		cn.Pos.Top = top
	})
}
