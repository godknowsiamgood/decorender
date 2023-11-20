package layout

import (
	"github.com/godknowsiamgood/decorender/fonts"
	"github.com/godknowsiamgood/decorender/utils"
	"strings"
	"unicode"
)

const hyphen = '-'
const hyphenString = string(hyphen)

func spitTextToNodes(nodes *Nodes, text string, context layoutPhaseContext) float64 {
	tokens := splitText(text)

	var height float64
	if context.props.LineHeight == -1 {
		height = float64(context.props.FontDescription.Size) * 1.3
	} else {
		height = context.props.LineHeight
	}

	for _, t := range tokens {
		node := Node{
			Size: utils.Size{
				W: fonts.MeasureTextWidth(t, context.props.FontDescription),
				H: height,
			},
			Props: CalculatedProperties{
				FontColor:       context.props.FontColor,
				FontDescription: context.props.FontDescription,
				LineHeight:      context.props.LineHeight,
			},
			Text:               t,
			TextHasHyphenAtEnd: strings.HasSuffix(t, hyphenString),
			ParentId:           context.parentId,
		}

		*nodes = append(*nodes, node)
	}

	withSpace := context.drawer.GetTextWidth("a b", context.props.FontDescription)
	withoutSpace := context.drawer.GetTextWidth("ab", context.props.FontDescription)

	return withSpace - withoutSpace
}

func splitText(input string) []string {
	var result []string
	var token strings.Builder

	const nonBreakable = '\u00A0'

	input = strings.ReplaceAll(input, "&nbsp;", string(nonBreakable))

	for _, r := range input {
		if unicode.IsSpace(r) && r != nonBreakable {
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
