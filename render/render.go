package render

import (
	"fmt"
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
	"strings"
	"sync"
)

/*
	Ideally, all positioning, including absolute world coordinates, should be handled during the layout phase.
	However, only local coordinates and sizes are calculated at layout stage.
	The computation of the final world position is performed during the rendering phase.
	This is solely due to one factor: if an element has a rotation,
	it needs to be drawn in a separate target, whose positions start from zero.
*/

type drawContext struct {
	cache *Cache
}

// drawState represents the current state in the rendering stack
type drawState struct {
	dst  *image.RGBA
	node *layout.Node
	// pos is a current world position
	pos utils.Pos
}

var stacksPool = sync.Pool{
	New: func() any {
		return make(utils.Stack[drawState], 0, 10)
	},
}

func Do(nodes layout.Nodes, cache *Cache) (*image.RGBA, error) {
	stack := stacksPool.Get().(utils.Stack[drawState])

	defer func() {
		// In case of early return (e.g. error occurred),
		// we have to release images in stack, but keeping top one - it will be released in caller
		var topDst image.Image
		for i := range stack {
			if i == 0 {
				topDst = stack[i].dst
			}
			if topDst != stack[i].dst {
				utils.ReleaseImage(stack[i].dst)
			}
		}

		stack = stack[0:0]
		stacksPool.Put(stack)
	}()

	dc := drawContext{
		cache: cache,
	}

	// Due to the nature of storing nodes in a one-dimensional array (see comments in the layout package),
	// the root node is located at the very end.
	for i := len(nodes) - 1; i >= 0; i-- {
		n := &nodes[i]

		// Ascend the stack if necessary
		popupStack(&stack, n.Level)

		state := stack.Last() // next node, new state

		// Create new destination image in case of root node and nodes that rotating
		if state.dst == nil || math.Abs(n.Props.Rotation) > math.SmallestNonzeroFloat64 {
			// New destination requires resetting world position.
			// Borders should be respected since they can be outside of element.
			borderOffset := n.Props.Border.GetOutsetOffset()
			state.dst = utils.NewRGBAImageFromPool(int(math.Ceil(n.Size.W+borderOffset*2)), int(math.Ceil(n.Size.H+borderOffset*2)))

			if err := drawNode(state.dst, n, borderOffset, borderOffset, dc); err != nil {
				utils.ReleaseImage(state.dst)
				return nil, err
			}

			// Next world position is just current node padding
			state.pos = utils.Pos{
				Left: n.Props.Padding.Left(),
				Top:  n.Props.Padding.Top(),
			}
		} else {
			if err := drawNode(state.dst, n, state.pos.Left+n.Pos.Left, state.pos.Top+n.Pos.Top, dc); err != nil {
				return nil, err
			}

			// Next world position is previous world + current node local position + current node padding
			state.pos = utils.Pos{
				Left: state.pos.Left + n.Pos.Left + n.Props.Padding.Left(),
				Top:  state.pos.Top + n.Pos.Top + n.Props.Padding.Top(),
			}
		}

		state.node = n
		stack.Push(state)
	}

	popupStack(&stack, 1)

	return stack[0].dst, nil
}

// At the next node, which is higher than the previous node in level,
// it is necessary to ascend the stack as many times as needed.
// Along the way, apply final renderings for rotations.
func popupStack(stack *utils.Stack[drawState], level int) {
	if stack.Len() == 0 || level > stack.Last().node.Level {
		return
	}

	for {
		state := stack.Pop()
		upperState := stack.Last()

		if state.dst != upperState.dst && upperState.dst != nil {
			// at this moment only case when destination may differ is rotation
			// so perform rotation of image and then render it on image upper on stack
			rotated := imaging.Rotate(state.dst, state.node.Props.Rotation, color.RGBA{})
			rotatedBounds := rotated.Bounds()

			// rotated image has different sizes, so we have to center it
			dx := rotatedBounds.Dx() - state.dst.Bounds().Dx() + int(state.node.Props.Border.GetOutsetOffset()*2)
			dy := rotatedBounds.Dy() - state.dst.Bounds().Dy() + int(state.node.Props.Border.GetOutsetOffset()*2)

			left, top := state.node.Pos.Left, state.node.Pos.Top
			bounds := image.Rect(
				int(upperState.pos.Left+left)-dx/2, int(upperState.pos.Top+top)-dy/2,
				int(upperState.pos.Left+left)-dx/2+rotatedBounds.Dx(), int(upperState.pos.Top+top)-dy/2+rotatedBounds.Dy())
			draw.Draw(upperState.dst, bounds, rotated, image.Point{}, draw.Over)
			utils.ReleaseImage(state.dst)
		}

		if level == state.node.Level {
			break
		}
	}
}

func drawNode(dst *image.RGBA, n *layout.Node, left float64, top float64, dc drawContext) error {
	if n.Props.BkgColor.A > 0 {
		drawRoundedRect(dc.cache, dst, alphaPremultiply(n.Props.BkgColor), left, top, n.Size.W, n.Size.H, n.Props.BorderRadius)
	}

	if n.Image != "" {
		err := dc.cache.useScaledImage(n.Image, n.Size.W, n.Size.H, n.Props.BkgImageSize, func(scaledAndCroppedImage image.Image) {
			bounds := scaledAndCroppedImage.Bounds()
			if n.Props.BorderRadius.HasValues() {
				utils.UseTempImage(bounds, func(tempImage *image.RGBA) {
					copyImage(tempImage, scaledAndCroppedImage)
					applyBorderRadius(dc.cache, tempImage, n.Props.BorderRadius)
					draw.Draw(dst, image.Rect(int(left), int(top), int(left)+bounds.Dx(), int(top)+bounds.Dy()), tempImage, image.Point{}, draw.Over)
				})
			} else {
				draw.Draw(dst, image.Rect(int(left), int(top), int(left)+bounds.Dx(), int(top)+bounds.Dy()), scaledAndCroppedImage, image.Point{}, draw.Over)
			}
		})
		if err != nil {
			return fmt.Errorf("cant draw node image (id: %v): %w", n.Id, err)
		}
	}

	if n.Text != "" {
		if err := renderText(dst, n, left, top); err != nil {
			return err
		}
	}

	if n.Props.Border.Width > 0 {
		drawRoundedBorder(dc.cache, dst, left, top, n.Size.W, n.Size.H, n.Props.BorderRadius, n.Props.Border)
	}

	return nil
}

func renderText(dst draw.Image, n *layout.Node, left float64, top float64) error {
	face, err := fonts.GetFontFace(n.Props.FontDescription)
	if err != nil {
		return fmt.Errorf("cant draw node text (id: %v): %w", n.Id, err)
	}

	offset := fonts.GetFontFaceBaseLineOffset(face, n.Size.H)
	pt := fixed.P(int(left), int(top+offset))
	ptY := pt.Y

	if strings.Contains(n.Text, ":") || strings.Contains(n.Text, ";") {
		for _, r := range n.Text {
			r = utils.SimplifyRune(r)

			// Since there is no sophisticated font rasterizer as harfbuzz
			// we have some issues with rendering some runes, like colons
			if r == ':' || r == ';' {
				pt.Y = ptY - fixed.I(int(3.0*n.Props.FontDescription.Size/44))
			} else {
				pt.Y = ptY
			}

			dr, mask, maskPoint, advance, ok := face.Glyph(pt, r)
			if !ok {
				continue
			}

			draw.DrawMask(dst, dr.Bounds(), image.NewUniform(n.Props.FontColor), image.Point{}, mask, maskPoint, draw.Over)
			pt.X += advance
		}
	} else {
		fontDrawer := font.Drawer{
			Dst:  dst,
			Face: face,
			Dot:  fixed.P(int(left), int(top+offset)),
			Src:  image.NewUniform(n.Props.FontColor),
		}

		fontDrawer.DrawString(n.Text)
	}

	return nil
}
