package layout

func applyAbsolutePositions(nodes *Nodes, childrenNodesLevel int, from int, props *CalculatedProperties) {
	nodes.IterateChildNodes(childrenNodesLevel, from, func(cn *Node) {
		if !cn.IsAbsolutePositioned() {
			return
		}

		var left float64
		var top float64

		topPadding := props.Padding.Top()
		rightPadding := props.Padding.Right()
		bottomPadding := props.Padding.Bottom()
		leftPadding := props.Padding.Left()

		if cn.Props.AbsolutePosition.HasLeft() || cn.Props.AbsolutePosition.HasRight() {
			if !cn.Props.AbsolutePosition.HasTop() && !cn.Props.AbsolutePosition.HasBottom() {
				top = (props.Size.H-topPadding-bottomPadding)/2 - cn.Size.H/2
			}
			if cn.Props.AbsolutePosition.HasRight() {
				left = props.Size.W - leftPadding - rightPadding - cn.Size.W - cn.Props.AbsolutePosition.Right()
			} else {
				left = cn.Props.AbsolutePosition.Left()
			}
		}
		if cn.Props.AbsolutePosition.HasTop() || cn.Props.AbsolutePosition.HasBottom() {
			if !cn.Props.AbsolutePosition.HasLeft() && !cn.Props.AbsolutePosition.HasRight() {
				left = (props.Size.W-leftPadding-rightPadding)/2 - cn.Size.W/2
			}
			if cn.Props.AbsolutePosition.HasBottom() {
				top = props.Size.H - topPadding - bottomPadding - cn.Size.H - cn.Props.AbsolutePosition.Bottom()
			} else {
				top = cn.Props.AbsolutePosition.Top()
			}
		}

		//if cn.Props.AbsolutePosition.HasLeft() || cn.Props.AbsolutePosition.HasRight() {
		//	if !cn.Props.AbsolutePosition.HasTop() && !cn.Props.AbsolutePosition.HasBottom() {
		//		top = props.Size.H/2 - cn.Size.H/2
		//	}
		//	if cn.Props.AbsolutePosition.HasRight() {
		//		left = props.Size.W - cn.Size.W - cn.Props.AbsolutePosition.Right()
		//	} else {
		//		left = cn.Props.AbsolutePosition.Left()
		//	}
		//}
		//if cn.Props.AbsolutePosition.HasTop() || cn.Props.AbsolutePosition.HasBottom() {
		//	if !cn.Props.AbsolutePosition.HasLeft() && !cn.Props.AbsolutePosition.HasRight() {
		//		left = props.Size.W/2 - cn.Size.W/2
		//	}
		//	if cn.Props.AbsolutePosition.HasBottom() {
		//		top = props.Size.H - cn.Size.H - cn.Props.AbsolutePosition.Bottom()
		//	} else {
		//		top = cn.Props.AbsolutePosition.Top()
		//	}
		//}

		cn.Pos.Left = left
		cn.Pos.Top = top
	})
}
