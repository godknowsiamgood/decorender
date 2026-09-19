package decorender

import (
	"os"
	"testing"
)

// The README documents the layout format by example. These are those examples,
// rendered, so the documentation cannot quietly drift away from what the parser
// and the renderer actually accept.

func TestReadmeExprExample(t *testing.T) {
	// The snippet under "Templates with Expr", given a size so it renders.
	const layout = `size: 300 200
bkgColor: white
sample:
  title: Report
  rows:
    - label: first
      color: 0x4fc3f7
inner:
  - text: ~ title
  - forEach: rows
    bkgColor: ~ value.color
    inner:
      - text: '~ string(index) + ". " + value.label'
`

	r, err := NewRendererWithTemplate([]byte(layout), nil)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := r.RenderAndWrite(nil, EncodeFormatNone, nil, &RenderOptions{UseSample: true}); err != nil {
		t.Fatalf("render: %v", err)
	}
}

func TestReadmeFormatKeys(t *testing.T) {
	// Every key the "Format" section lists, with the values it shows.
	const layout = `size: 400 300
scale: 1
fontFaces:
  - family: Doc
    style: italic
    weight: 400
    file: internal/fonts/default.ttf
sample:
  n: 2
inner:
  - id: header
    size: 100% 50%
    width: 50%
    height: 30
    padding: 10 20
    offset: left/5 top/-2
    rotate: 45
    innerDirection: row
    justify: space-evenly
    innerColumnAlign: right
    innerRowAlign: center
    innerWrap: none
    innerGap: 5
    bkgColor: rgba(239, 83, 80, 0.55)
    border: 2 salmon inset dashed/10/4/2/4 top right
    borderRadius: 20
    bkgImage: test_img.jpeg
    bkgImageSize: cover
    inner:
      - text: Hello
        color: black
        font: Doc 23 400 italic
        lineHeight: 30
      - absolute: right bottom
        size: 10 10
        bkgColor: 0xaabbccff
      - forEach: 2
        size: 0.1w 0.1h
        bkgColor: 0xaabbcc
        fontFamily: Doc
        fontSize: 23
        fontWeight: 400
        fontStyle: italic
        fontColor: salmon
        text: x
`

	r, err := NewRendererWithTemplate([]byte(layout), &Options{LocalFiles: os.DirFS(".")})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := r.RenderAndWrite(nil, EncodeFormatNone, nil, &RenderOptions{UseSample: true}); err != nil {
		t.Fatalf("render: %v", err)
	}
}
