package parsing

import (
	"regexp"

	"gopkg.in/yaml.v3"
)

// colorHexScalar matches the plain scalars a sample can hold that the color
// parser would accept: 0x followed by six or eight hex digits. Hex numbers of
// any other length stay integers, so nothing but a color is affected.
var colorHexScalar = regexp.MustCompile(`^0[xX][a-fA-F0-9]{6}([a-fA-F0-9]{2})?$`)

// SampleValue is the arbitrary object a template is previewed with.
//
// It exists to keep hex colors readable. YAML resolves a plain 0xRRGGBB scalar
// to an integer, so `color: 0x4fc3f7` inside a sample reached the renderer as
// "5227511", failed to parse as a color and silently rendered transparent -
// while the identical literal written directly on a node is a string and
// renders fine, because every node property is typed string. Re-tagging those
// scalars makes a color mean the same thing wherever it is written.
type SampleValue struct {
	Value any
}

func (s *SampleValue) UnmarshalYAML(node *yaml.Node) error {
	retagColorScalars(node)
	return node.Decode(&s.Value)
}

// retagColorScalars walks the sample and marks every plain hex-color scalar as
// a string, leaving quoted scalars and every other kind of value alone.
func retagColorScalars(n *yaml.Node) {
	if n.Kind == yaml.ScalarNode {
		if n.Tag == "!!int" && n.Style == 0 && colorHexScalar.MatchString(n.Value) {
			n.Tag = "!!str"
		}
		return
	}
	for _, c := range n.Content {
		retagColorScalars(c)
	}
}
