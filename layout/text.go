package layout

import (
	"strings"
	"unicode"
)

const hyphen = '-'
const hyphenString = string(hyphen)

func spitTextToNodes(text string, context layoutPhaseContext) ([]*Node, Node) {
	tokens := splitText(text)

	var height float64
	if context.props.LineHeight == -1 {
		height = float64(context.props.FontDescription.Size) * 1.3
	} else {
		height = context.props.LineHeight
	}

	var nodes []*Node

	for _, t := range tokens {
		node := nodesPool.Get().(*Node)
		node.Text = t
		node.TextHasHyphenAtEnd = strings.HasSuffix(t, hyphenString)
		node.Size.W = context.drawer.GetTextWidth(t, context.props.FontDescription)
		node.Size.H = height
		node.Props = CalculatedProperties{
			FontColor:       context.props.FontColor,
			FontDescription: context.props.FontDescription,
			LineHeight:      context.props.LineHeight,
		}
		nodes = append(nodes, node)
	}

	var whiteSpaceNode Node
	withSpace := context.drawer.GetTextWidth("a b", context.props.FontDescription)
	withoutSpace := context.drawer.GetTextWidth("ab", context.props.FontDescription)
	whiteSpaceNode.Size.W, whiteSpaceNode.Size.H = withSpace-withoutSpace, height

	return nodes, whiteSpaceNode
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
