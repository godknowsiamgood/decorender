package layout

const (
	SPACE                   rune = 0x0020
	NBSP                    rune = 0x00A0
	NNBSP                   rune = 0x202F
	MMSP                    rune = 0x205F
	MongolianVowelSeparator rune = 0x180E
	EnQuad                  rune = 0x2000
	EmQuad                  rune = 0x2001
	EnSpace                 rune = 0x2002
	EmSpace                 rune = 0x2003
	ThreePerEmSpace         rune = 0x2004
	FourPerEmSpace          rune = 0x2005
	SixPerEmSpace           rune = 0x2006
	FigureSpace             rune = 0x2007
	PunctuationSpace        rune = 0x2008
	ThinSpace               rune = 0x2009
	HairSpace               rune = 0x200A
	ZeroWidthSpace          rune = 0x200B
	IdeographicSpace        rune = 0x3000
)

func simplifyRune(char rune) rune {
	switch char {
	case NNBSP:
		return NBSP
	case MMSP, MongolianVowelSeparator, EnQuad,
		EmQuad, EnSpace, EmSpace, ThreePerEmSpace, FourPerEmSpace,
		SixPerEmSpace, FigureSpace, PunctuationSpace, ThinSpace,
		HairSpace, ZeroWidthSpace, IdeographicSpace:
		return SPACE
	default:
		return char
	}
}
