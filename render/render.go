package render

import (
	"bytes"
	"github.com/godknowsiamgood/decorender/draw"
	"github.com/godknowsiamgood/decorender/layout"
	"github.com/godknowsiamgood/decorender/resources"
	"image"
	"math"
)

func Do(n layout.Node, drawer draw.Drawer) {
	drawer.SaveState()

	topPadding := n.Props.Padding[0]
	rightPadding := n.Props.Padding[1]
	bottomPadding := n.Props.Padding[2]
	leftPadding := n.Props.Padding[3]

	if math.Abs(n.Props.Rotation) > 0.001 {
		centerX, centerY := n.Pos.Left+n.Size.W/2, n.Pos.Top+n.Size.H/2
		drawer.SetTranslation(centerX, centerY)
		drawer.SetRotation(n.Props.Rotation)
		drawer.SetTranslation(-centerX, -centerY)
	}

	drawer.SetTranslation(n.Pos.Left, n.Pos.Top)

	if n.Size.W > 0 && n.Size.H > 0 && n.Props.BkgColor.A > 0 {
		drawer.DrawRect(n.Size.W, n.Size.H, n.Props.BkgColor, n.Props.BorderRadius)
	}

	if n.Image != "" {
		imageBytes, err := resources.GetResourceContent(n.Image)
		if err == nil {
			imgReader := bytes.NewReader(imageBytes)
			srcImg, _, err := image.Decode(imgReader)
			srcImg = scaleAndCropImage(srcImg, n.Size.W, n.Size.H)
			if n.Props.BorderRadius.HasValues() {
				srcImg = applyBorderRadius(srcImg, n.Props.BorderRadius)
			}
			if err == nil {
				drawer.DrawImage(srcImg)
			}
		}
	}

	drawer.SetTranslation(leftPadding, topPadding)

	if n.Text != "" {
		heightAndSizeOffset := n.Size.H - (n.Size.H-n.Props.FontDescription.Size)/2
		drawer.SetTranslation(0, heightAndSizeOffset)
		drawer.DrawText(n.Text, n.Props.FontDescription, n.Props.FontColor)
		drawer.SetTranslation(0, -heightAndSizeOffset)
	}

	for _, cn := range n.Children {
		var absTop, absLeft float64

		if cn.HasAnchors() {
			if cn.Props.Anchors.Left() || cn.Props.Anchors.Right() {
				if !cn.Props.Anchors.Top() && !cn.Props.Anchors.Bottom() {
					absTop = (n.Size.H-topPadding-bottomPadding)/2 - cn.Size.H/2
				}
				if cn.Props.Anchors.Right() {
					absLeft = n.Size.W - leftPadding - rightPadding - cn.Size.W
				}
			}
			if cn.Props.Anchors.Top() || cn.Props.Anchors.Bottom() {
				if !cn.Props.Anchors.Left() && !cn.Props.Anchors.Right() {
					absLeft = (n.Size.W-leftPadding-rightPadding)/2 - cn.Size.W/2
				}
				if cn.Props.Anchors.Bottom() {
					absTop = n.Size.H - topPadding - bottomPadding - cn.Size.H
				}
			}
		}

		drawer.SetTranslation(absLeft, absTop)

		Do(cn, drawer)

		drawer.SetTranslation(-absLeft, -absTop)
	}

	drawer.RestoreState()
}
