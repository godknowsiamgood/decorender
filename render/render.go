package render

import (
	"github.com/disintegration/imaging"
	"github.com/godknowsiamgood/decorender/fonts"
	"github.com/godknowsiamgood/decorender/layout"
	"github.com/godknowsiamgood/decorender/utils"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
	"image"
	"image/color"
	"math"
	"sync"
)

type drawState struct {
	level int
	dst   *image.RGBA
	pos   utils.Pos
	node  *layout.Node
}

var stacksPool = sync.Pool{
	New: func() any {
		return make(utils.Stack[drawState], 0, 10)
	},
}

var rasterizerPool = sync.Pool{
	New: func() any {
		return &vector.Rasterizer{}
	},
}

func Do(nodes layout.Nodes) *image.RGBA {
	cache := newCache()

	stack := stacksPool.Get().(utils.Stack[drawState])

	for i := len(nodes) - 1; i >= 0; i-- {
		n := &nodes[i]

		popupStack(&stack, n.Level)

		// Prepare state and render
		state := stack.Last()
		if state.dst == nil || math.Abs(n.Props.Rotation) > math.SmallestNonzeroFloat64 {
			state.dst = utils.NewRGBAImageFromPool(int(math.Ceil(n.Size.W)), int(math.Ceil(n.Size.H)))
			left, top := getLeftTopOffsets(state.node, n)
			drawNode(&cache, state.dst, n, left, top)
			state.pos = utils.Pos{
				Left: n.Props.Padding[3],
				Top:  n.Props.Padding[0],
			}
		} else {
			left, top := getLeftTopOffsets(state.node, n)
			drawNode(&cache, state.dst, n, state.pos.Left+left, state.pos.Top+top)
			state.pos = utils.Pos{
				Left: state.pos.Left + n.Pos.Left + n.Props.Padding[3],
				Top:  state.pos.Top + n.Pos.Top + n.Props.Padding[0],
			}
		}
		state.node = n
		state.level = n.Level

		stack.Push(state)
	}

	popupStack(&stack, 1)

	defer func() {
		stack = stack[0:0]
		stacksPool.Put(stack)
		cache.release()
	}()

	return stack[0].dst
}

func popupStack(stack *utils.Stack[drawState], level int) {
	if level <= stack.Last().level {
		for {
			state := stack.Pop()
			upperState := stack.Last()

			if state.dst != upperState.dst && upperState.dst != nil {
				rotated := imaging.Rotate(state.dst, state.node.Props.Rotation, color.RGBA{})
				rBounds := rotated.Bounds()
				currBounds := state.dst.Bounds()
				dx := rBounds.Dx() - currBounds.Dx()
				dy := rBounds.Dy() - currBounds.Dy()
				bounds := image.Rect(
					int(state.node.Pos.Left)-dx/2, int(state.node.Pos.Top)-dy/2,
					int(state.node.Pos.Left)-dx/2+rBounds.Dx(), int(state.node.Pos.Top)-dy/2+rBounds.Dy())
				draw.Draw(upperState.dst, bounds, rotated, image.Point{}, draw.Over)
				utils.ReleaseImage(state.dst)
			}

			if level == state.level {
				break
			}
		}
	}
}

func getLeftTopOffsets(parentNode *layout.Node, childNode *layout.Node) (float64, float64) {
	var left float64
	var top float64

	if childNode.HasAnchors() {
		topPadding := parentNode.Props.Padding[0]
		rightPadding := parentNode.Props.Padding[1]
		bottomPadding := parentNode.Props.Padding[2]
		leftPadding := parentNode.Props.Padding[3]

		if childNode.Props.Anchors.HasLeft() || childNode.Props.Anchors.HasRight() {
			if !childNode.Props.Anchors.HasTop() && !childNode.Props.Anchors.HasBottom() {
				top = (parentNode.Size.H-topPadding-bottomPadding)/2 - childNode.Size.H/2
			}
			if childNode.Props.Anchors.HasRight() {
				left = parentNode.Size.W - leftPadding - rightPadding - childNode.Size.W - childNode.Props.Anchors.Right()
			} else {
				left = childNode.Props.Anchors.Left()
			}
		}
		if childNode.Props.Anchors.HasTop() || childNode.Props.Anchors.HasBottom() {
			if !childNode.Props.Anchors.HasLeft() && !childNode.Props.Anchors.HasRight() {
				left = (parentNode.Size.W-leftPadding-rightPadding)/2 - childNode.Size.W/2
			}
			if childNode.Props.Anchors.HasBottom() {
				top = parentNode.Size.H - topPadding - bottomPadding - childNode.Size.H - childNode.Props.Anchors.Bottom()
			} else {
				top = childNode.Props.Anchors.Top()
			}
		}
	} else {
		left, top = childNode.Pos.Left, childNode.Pos.Top
	}

	return left, top
}

func drawNode(cache *cache, dst *image.RGBA, n *layout.Node, x float64, y float64) {
	if n.Props.BkgColor.A > 0 {
		drawRoundedRect(cache, dst, alphaPremultiply(n.Props.BkgColor), x, y, n.Size.W, n.Size.H, n.Props.BorderRadius)
	}

	if n.Image != "" {
		scaledAndCroppedImage := getScaledImage(cache, n.Image, n.Size.W, n.Size.H, n.Props.BkgImageSize)
		if scaledAndCroppedImage != nil {
			bounds := scaledAndCroppedImage.Bounds()
			if n.Props.BorderRadius.HasValues() {
				utils.UseTempImage(bounds.Dx(), bounds.Dy(), func(tempImage *image.RGBA) {
					copyImage(tempImage, scaledAndCroppedImage)
					applyBorderRadius(cache, tempImage, n.Props.BorderRadius)
					draw.Draw(dst, image.Rect(int(x), int(y), int(x)+bounds.Dx(), int(y)+bounds.Dy()), tempImage, image.Point{}, draw.Over)
				})
			} else {
				draw.Draw(dst, image.Rect(int(x), int(y), int(x)+bounds.Dx(), int(y)+bounds.Dy()), scaledAndCroppedImage, image.Point{}, draw.Over)
			}
		}
	}

	if n.Text != "" {
		var fontDrawer *font.Drawer
		cache.prevUsedFaceMx.Lock()
		if cache.prevUsedFaceDescription == n.Props.FontDescription {
			fontDrawer = cache.prevUsedFaceDrawer
			fontDrawer.Dot = fixed.P(int(x), int(y+cache.prevUsedFaceOffset))
		} else {
			face := fonts.GetFontFace(n.Props.FontDescription)
			offset := fonts.GetFontFaceBaseLineOffset(face, n.Size.H)
			fontDrawer = &font.Drawer{
				Dst:  dst,
				Src:  image.NewUniform(n.Props.FontColor),
				Face: face,
				Dot:  fixed.P(int(x), int(y+offset)),
			}

			cache.prevUsedFaceDescription = n.Props.FontDescription
			cache.prevUsedFaceDrawer = fontDrawer
			cache.prevUsedFaceOffset = offset
		}
		cache.prevUsedFaceMx.Unlock()
		fontDrawer.DrawString(n.Text)
	}

	if n.Props.Border.Width > 0 {
		drawRoundedBorder(cache, dst, x, y, n.Size.W, n.Size.H, n.Props.BorderRadius, n.Props.Border)
	}
}
