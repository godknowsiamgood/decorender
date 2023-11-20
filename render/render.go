package render

import (
	"github.com/godknowsiamgood/decorender/draw"
	"github.com/godknowsiamgood/decorender/layout"
	"github.com/godknowsiamgood/decorender/utils"
	"image"
	"math"
	"sync"
)

type DrawState struct {
	dst *image.RGBA

	left float64
	top  float64
}

var stacksPool = sync.Pool{
	New: func() any {
		return make(utils.Stack[int], 0, 10)
	},
}

func Do(nodes layout.Nodes, drawer draw.Drawer) {
	stack := stacksPool.Get().(utils.Stack[int])

	for i := len(nodes) - 1; i >= 0; i-- {
		n := &nodes[i]

		if i == 0 {
			drawer.InitImage(int(math.Ceil(n.Size.W)), int(math.Ceil(n.Size.H)))
		}

		if n.ParentId <= stack.Last(-1) {
			//drawer.RestoreState()
			for stack.Pop() != n.ParentId {
				//drawer.RestoreState()
			}
		}

		if n.ParentId > stack.Last(-1) {
			stack.Push(n.ParentId)
			//drawer.SaveState()
		}

		if n.Size.W > 0 && n.Size.H > 0 && n.Props.BkgColor.A > 0 {
			drawer.DrawRect(n.Size.W, n.Size.H, n.Props.BkgColor, n.Props.Border, n.Props.BorderRadius)
		}
	}

	stack = stack[0:0]
	stacksPool.Put(stack)
}

//
//func do(nodes layout.Nodes, drawer draw.Drawer) {
//	drawer.SaveState()
//
//	topPadding := n.Props.Padding[0]
//	rightPadding := n.Props.Padding[1]
//	bottomPadding := n.Props.Padding[2]
//	leftPadding := n.Props.Padding[3]
//
//	if math.Abs(n.Props.Rotation) > 0.001 {
//		centerX, centerY := n.Pos.Left+n.Size.W/2, n.Pos.Top+n.Size.H/2
//		drawer.SetTranslation(centerX, centerY)
//		drawer.SetRotation(n.Props.Rotation)
//		drawer.SetTranslation(-centerX, -centerY)
//	}
//
//	drawer.SetTranslation(n.Pos.Left, n.Pos.Top)
//
//	if n.Size.W > 0 && n.Size.H > 0 && n.Props.BkgColor.A > 0 {
//		drawer.DrawRect(n.Size.W, n.Size.H, n.Props.BkgColor, n.Props.Border, n.Props.BorderRadius)
//	}
//
//	if n.Image != "" {
//		imageBytes, err := resources.GetResourceContent(n.Image)
//		if err == nil {
//			imgReader := bytes.NewReader(imageBytes)
//			srcImg, _, err := image.Decode(imgReader)
//
//			scaledAndCroppedImage := scaleAndCropImage(srcImg, n.Size.W, n.Size.H, n.Props.BkgImageSize == "contain")
//			if n.Props.BorderRadius.HasValues() {
//				srcImg = applyBorderRadius(srcImg, n.Props.BorderRadius)
//			}
//			if err == nil {
//				drawer.DrawImage(&scaledAndCroppedImage)
//			}
//			utils.ReleaseImage(scaledAndCroppedImage)
//		}
//	}
//
//	drawer.SetTranslation(leftPadding, topPadding)
//
//	if n.Text != "" {
//		offset := fonts.GetFontFaceBaseLineOffset(fonts.GetFontFace(n.Props.FontDescription), n.Size.H)
//		drawer.SetTranslation(0, offset)
//		drawer.DrawText(n.Text, n.Props.FontDescription, n.Props.FontColor)
//		drawer.SetTranslation(0, -offset)
//	}
//
//	for _, cn := range n.Children {
//		var absTop, absLeft float64
//
//		if cn.HasAnchors() {
//			if cn.Props.Anchors.HasLeft() || cn.Props.Anchors.HasRight() {
//				if !cn.Props.Anchors.HasTop() && !cn.Props.Anchors.HasBottom() {
//					absTop = (n.Size.H-topPadding-bottomPadding)/2 - cn.Size.H/2
//				}
//				if cn.Props.Anchors.HasRight() {
//					absLeft = n.Size.W - leftPadding - rightPadding - cn.Size.W - cn.Props.Anchors.Right()
//				} else {
//					absLeft = cn.Props.Anchors.Left()
//				}
//			}
//			if cn.Props.Anchors.HasTop() || cn.Props.Anchors.HasBottom() {
//				if !cn.Props.Anchors.HasLeft() && !cn.Props.Anchors.HasRight() {
//					absLeft = (n.Size.W-leftPadding-rightPadding)/2 - cn.Size.W/2
//				}
//				if cn.Props.Anchors.HasBottom() {
//					absTop = n.Size.H - topPadding - bottomPadding - cn.Size.H - cn.Props.Anchors.Bottom()
//				} else {
//					absTop = cn.Props.Anchors.Top()
//				}
//			}
//		}
//
//		drawer.SetTranslation(absLeft, absTop)
//
//		do(cn, drawer)
//
//		drawer.SetTranslation(-absLeft, -absTop)
//	}
//
//	drawer.RestoreState()
//}
