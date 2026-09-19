package layout

import (
	"github.com/godknowsiamgood/decorender/internal/fonts"
	"github.com/godknowsiamgood/decorender/internal/utils"
	"golang.org/x/text/unicode/norm"
	"strings"
	"unicode"
)

const hyphen = '-'
const hyphenString = string(hyphen)

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
				W: context.faces.MeasureTextWidth(t, context.props.FontDescription),
				H: height,
			},
			Props: CalculatedProperties{
				FontColor:       context.props.FontColor,
				FontDescription: context.props.FontDescription,
				LineHeight:      context.props.LineHeight,
			},
			Text:               t,
			TextHasHyphenAtEnd: strings.HasSuffix(t, hyphenString),
			Level:              context.level + 1,
		}

		*nodes = append(*nodes, node)
	}

	return context.faces.MeasureTextWidth(" ", context.props.FontDescription)
}

func splitText(input string) []string {
	var result []string
	var token strings.Builder

	const nonBreakable = '\u00A0'
	input = strings.ReplaceAll(input, "&nbsp;", string(nonBreakable))

	input = norm.NFC.String(input)

	for _, r := range input {
		if unicode.IsSpace(r) && r != nonBreakable || r == '\n' {
			if token.Len() > 0 {
				result = append(result, token.String())
				token.Reset()
			}
		} else if r == hyphen {
			token.WriteRune(r)
			result = append(result, token.String())
			token.Reset()
		} else {
			token.WriteRune(r)
		}
	}

	if token.Len() > 0 {
		result = append(result, token.String())
	}

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
			if last != nil && !last.TextHasHyphenAtEnd {
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
