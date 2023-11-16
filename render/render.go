package render

import (
	"bytes"
	"github.com/godknowsiamgood/decorender/draw"
	"github.com/godknowsiamgood/decorender/fonts"
	"github.com/godknowsiamgood/decorender/layout"
	"github.com/godknowsiamgood/decorender/resources"
	"image"
	"math"
)

func Do(n layout.Node, drawer draw.Drawer) {
	drawer.SaveState()

	topPadding := n.Props.Padding[0]
	leftPadding := n.Props.Padding[3]

	if math.Abs(n.Props.Rotation) > 0.001 {
		centerX, centerY := n.Pos.Left+n.Size.W/2, n.Pos.Top+n.Size.H/2
		drawer.SetTranslation(centerX, centerY)
		drawer.SetRotation(n.Props.Rotation)
		drawer.SetTranslation(-centerX, -centerY)
	}

	drawer.SetTranslation(n.Pos.Left, n.Pos.Top)

	if n.Size.W > 0 && n.Size.H > 0 && n.Props.BkgColor.A > 0 {
		drawer.DrawRect(n.Size.W, n.Size.H, n.Props.BkgColor, n.Props.Border, n.Props.BorderRadius)
	}

	if n.Image != "" {
		imageBytes, err := resources.GetResourceContent(n.Image)
		if err == nil {
			imgReader := bytes.NewReader(imageBytes)
			srcImg, _, err := image.Decode(imgReader)
			srcImg = scaleAndCropImage(srcImg, n.Size.W, n.Size.H, n.Props.BkgImageSize == "contain")
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
		offset := fonts.GetFontFaceBaseLineOffset(fonts.GetFontFace(n.Props.FontDescription), n.Size.H)
		drawer.SetTranslation(0, offset)
		drawer.DrawText(n.Text, n.Props.FontDescription, n.Props.FontColor)
		drawer.SetTranslation(0, -offset)
	}

	for _, cn := range n.Children {
		var absTop, absLeft float64

		if cn.HasAnchors() {
			if cn.Props.Anchors.HasLeft() || cn.Props.Anchors.HasRight() {
				if !cn.Props.Anchors.HasTop() && !cn.Props.Anchors.HasBottom() {
					absTop = n.Size.H/2 - cn.Size.H/2
				}
				if cn.Props.Anchors.HasRight() {
					absLeft = n.Size.W - cn.Size.W - cn.Props.Anchors.Right()
				} else {
					absLeft = cn.Props.Anchors.Left()
				}
			}
			if cn.Props.Anchors.HasTop() || cn.Props.Anchors.HasBottom() {
				if !cn.Props.Anchors.HasLeft() && !cn.Props.Anchors.HasRight() {
					absLeft = n.Size.W/2 - cn.Size.W/2
				}
				if cn.Props.Anchors.HasBottom() {
					absTop = n.Size.H - cn.Size.H - cn.Props.Anchors.Bottom()
				} else {
					absTop = cn.Props.Anchors.Top()
				}
			}

			// absolute positioned should not respect parent padding
			absTop -= topPadding
			absLeft -= leftPadding
		}

		drawer.SetTranslation(absLeft, absTop)

		Do(cn, drawer)

		drawer.SetTranslation(-absLeft, -absTop)
	}

	drawer.RestoreState()
}
