package layout

import (
	"github.com/godknowsiamgood/decorender/internal/fonts"
	"github.com/godknowsiamgood/decorender/internal/utils"
	"golang.org/x/text/unicode/norm"
	"strings"
	"unicode"
)

const hyphen = '-'

// Zero-width break characters mark a place a word may be split without
// showing anything at the break. They are dropped from the text and leave a
// joining token behind, the same shape a hyphen leaves minus the hyphen.
const (
	zeroWidthSpace          = '\u200B'
	mongolianVowelSeparator = '\u180E'
)

// textToken is one run of text that cannot be broken further, along with
// whether the token after it follows immediately rather than after a space.
type textToken struct {
	text      string
	joinsNext bool
}

func spitTextToNodes(nodes *Nodes, text string, context layoutPhaseContext) float64 {
	tokens := splitText(text)

	var height float64
	if context.props.LineHeight == -1 {
		height = float64(context.props.FontDescription.Size) * 1.2
	} else {
		height = context.props.LineHeight
	}

	for i := len(tokens) - 1; i >= 0; i-- {
		t := tokens[i]

		node := Node{
			Size: utils.Size{
				W: context.faces.MeasureTextWidth(t.text, context.props.FontDescription),
				H: height,
			},
			Props: CalculatedProperties{
				FontColor:       context.props.FontColor,
				FontDescription: context.props.FontDescription,
				LineHeight:      context.props.LineHeight,
			},
			Text:           t.text,
			JoinsNextToken: t.joinsNext,
			Level:          context.level + 1,
		}

		*nodes = append(*nodes, node)
	}

	return context.faces.MeasureTextWidth(" ", context.props.FontDescription)
}

func splitText(input string) []textToken {
	var result []textToken
	var token strings.Builder

	const nonBreakable = '\u00A0'
	input = strings.ReplaceAll(input, "&nbsp;", string(nonBreakable))

	input = norm.NFC.String(input)

	flush := func(joinsNext bool) {
		if token.Len() == 0 {
			return
		}
		result = append(result, textToken{text: token.String(), joinsNext: joinsNext})
		token.Reset()
	}

	for _, r := range input {
		switch {
		case unicode.IsSpace(r) && r != nonBreakable || r == '\n':
			flush(false)
		case r == zeroWidthSpace || r == mongolianVowelSeparator:
			// A break opportunity that occupies no space and draws nothing,
			// so the rune itself is dropped rather than measured or rendered.
			flush(true)
		case r == hyphen:
			token.WriteRune(r)
			flush(true)
		default:
			token.WriteRune(r)
		}
	}

	flush(false)

	return result
}

// Little tricky method to merge texts nodes in rows into one node per row for optimized rendering
func mergeTextNodes(nodes *Nodes, level int, from int, faces *fonts.FaceSet) {
	var sb strings.Builder

	originalFrom := from
	index := 0
	nodes.IterateRowsReverse(level, from, func(rowIndex int) {
		sb.Reset()
		var last *Node
		nodes.IterateRow(level, from, rowIndex, func(n *Node) {
			if last != nil && !last.JoinsNextToken {
				sb.WriteString(" ")
			}
			sb.WriteString(n.Text)
			last = n
			from++
		})
		if last == nil {
			return
		}
		last.Text = sb.String()
		last.Size.W = faces.MeasureTextWidth(last.Text, last.Props.FontDescription)
		last.InRowIndex = 0

		(*nodes)[originalFrom+index] = *last
		index++
	})

	*nodes = (*nodes)[0 : originalFrom+index]
}
