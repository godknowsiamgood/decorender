package render

import (
	"github.com/disintegration/imaging"
	"github.com/godknowsiamgood/decorender/fonts"
	"github.com/godknowsiamgood/decorender/layout"
	"github.com/godknowsiamgood/decorender/utils"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	"math"
	"sync"
)

// drawState is a current state in render stack
// it keeps current level, to track when to pop (actually it is stored in node, but separate field is more convenient)
// it keeps current image destination (when it comes to rotation, all nodes should be rendered separately)
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

func Do(nodes layout.Nodes) *image.RGBA {
	cache := newCache()
	stack := stacksPool.Get().(utils.Stack[drawState])

	defer func() {
		stack = stack[0:0]
		stacksPool.Put(stack)
		cache.release()
	}()

	// last node in slice is a root, since in recursive layout phase it was added in very last step
	for i := len(nodes) - 1; i >= 0; i-- {
		n := &nodes[i]

		popupStack(&stack, n.Level)

		state := stack.Last() // copy of current state

		// Create new destination image in case of root node and nodes that rotating
		if state.dst == nil || math.Abs(n.Props.Rotation) > math.SmallestNonzeroFloat64 {
			state.dst = utils.NewRGBAImageFromPool(int(math.Ceil(n.Size.W)), int(math.Ceil(n.Size.H)))
			drawNode(&cache, state.dst, n, 0, 0) // new destination, starting from origin point
			state.pos = utils.Pos{
				Left: n.Props.Padding.Left(),
				Top:  n.Props.Padding.Top(),
			}
		} else {
			left, top := getLocalLeftTop(state.node, n)
			drawNode(&cache, state.dst, n, state.pos.Left+left, state.pos.Top+top)

			// Next position is world + current + current's padding
			state.pos = utils.Pos{
				Left: state.pos.Left + left + n.Props.Padding.Left(),
				Top:  state.pos.Top + top + n.Props.Padding.Top(),
			}
		}

		state.node = n
		state.level = n.Level
		stack.Push(state) // new state added
	}

	popupStack(&stack, 1)

	return stack[0].dst
}

func popupStack(stack *utils.Stack[drawState], level int) {
	if level <= stack.Last().level {
		for {
			state := stack.Pop()
			upperState := stack.Last()

			if state.dst != upperState.dst && upperState.dst != nil {
				// at this moment only case when destination may differ is rotation
				// so perform rotation of image and then render it on image upper on stack
				rotated := imaging.Rotate(state.dst, state.node.Props.Rotation, color.RGBA{})
				rotatedBounds := rotated.Bounds()

				// rotated image has different sizes, so we have to center it
				dx := rotatedBounds.Dx() - state.dst.Bounds().Dx()
				dy := rotatedBounds.Dy() - state.dst.Bounds().Dy()

				left, top := getLocalLeftTop(upperState.node, state.node)
				bounds := image.Rect(
					int(upperState.pos.Left+left)-dx/2, int(upperState.pos.Top+top)-dy/2,
					int(upperState.pos.Left+left)-dx/2+rotatedBounds.Dx(), int(upperState.pos.Top+top)-dy/2+rotatedBounds.Dy())
				draw.Draw(upperState.dst, bounds, rotated, image.Point{}, draw.Over)
				utils.ReleaseImage(state.dst)
			}

			if level == state.level {
				break
			}
		}
	}
}

func getLocalLeftTop(parentNode *layout.Node, childNode *layout.Node) (float64, float64) {
	var left float64
	var top float64

	if childNode.HasAnchors() {
		topPadding := parentNode.Props.Padding.Top()
		rightPadding := parentNode.Props.Padding.Right()
		bottomPadding := parentNode.Props.Padding.Bottom()
		leftPadding := parentNode.Props.Padding.Left()

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

func drawNode(cache *cache, dst *image.RGBA, n *layout.Node, left float64, top float64) {
	if n.Props.BkgColor.A > 0 {
		drawRoundedRect(cache, dst, alphaPremultiply(n.Props.BkgColor), left, top, n.Size.W, n.Size.H, n.Props.BorderRadius)
	}

	if n.Image != "" {
		scaledAndCroppedImage := getScaledImage(cache, n.Image, n.Size.W, n.Size.H, n.Props.BkgImageSize)
		if scaledAndCroppedImage != nil {
			bounds := scaledAndCroppedImage.Bounds()
			if n.Props.BorderRadius.HasValues() {
				utils.UseTempImage(bounds.Dx(), bounds.Dy(), func(tempImage *image.RGBA) {
					copyImage(tempImage, scaledAndCroppedImage)
					applyBorderRadius(cache, tempImage, n.Props.BorderRadius)
					draw.Draw(dst, image.Rect(int(left), int(top), int(left)+bounds.Dx(), int(top)+bounds.Dy()), tempImage, image.Point{}, draw.Over)
				})
			} else {
				draw.Draw(dst, image.Rect(int(left), int(top), int(left)+bounds.Dx(), int(top)+bounds.Dy()), scaledAndCroppedImage, image.Point{}, draw.Over)
			}
		}
	}

	if n.Text != "" {
		var fontDrawer *font.Drawer
		cache.prevUsedFaceMx.Lock()
		if cache.prevUsedFaceDescription == n.Props.FontDescription {
			fontDrawer = cache.prevUsedFaceDrawer
			fontDrawer.Dot = fixed.P(int(left), int(top+cache.prevUsedFaceOffset))
		} else {
			face := fonts.GetFontFace(n.Props.FontDescription)
			offset := fonts.GetFontFaceBaseLineOffset(face, n.Size.H)
			fontDrawer = &font.Drawer{
				Dst:  dst,
				Src:  image.NewUniform(n.Props.FontColor),
				Face: face,
				Dot:  fixed.P(int(left), int(top+offset)),
			}

			cache.prevUsedFaceDescription = n.Props.FontDescription
			cache.prevUsedFaceDrawer = fontDrawer
			cache.prevUsedFaceOffset = offset
		}
		cache.prevUsedFaceMx.Unlock()
		fontDrawer.DrawString(n.Text)
	}

	if n.Props.Border.Width > 0 {
		drawRoundedBorder(cache, dst, left, top, n.Size.W, n.Size.H, n.Props.BorderRadius, n.Props.Border)
	}
}
