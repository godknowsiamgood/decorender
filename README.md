# Decorender

A library for declarative rendering on the backend. Considering that there is no goal to replicate browser rendering, a simple positioning model has been implemented.

![image](https://github.com/godknowsiamgood/decorender/assets/5710885/6b6536a6-b208-4abb-ade1-70b613075695)

## Usage
The layout is described using a yaml file, in which templates can be used to customize any field.

`decorender.NewRenderer` parses this yaml file with your markup, validates the data, compiles the templates, initializes the necessary data, caches, and returns an object that is then used for frequent rendering with data.

To make layout process easier and more convenient, decorender has a dev server with auto-reloading and error display. An example of its use is shown below.

### Usage with dev server
Create file `layout.yaml` with minimal content:
```yaml
text: Hello, world!
```
Make sure you have Go installed
```
brew install go
export PATH=$PATH:$HOME/go/bin
```

Install dev server for easy visualising your layouts
```
go install github.com/godknowsiamgood/decorender/cmd/decorender_server@latest
```

Start dev server. It will open page with autoreload and some useful information. If your template has templates, you can mock them with `sample` field.
```
decorender_server layout.yaml
```

### Then on your backend

```
go get -u github.com/godknowsiamgood/decorender
```

```go
// Create a renderer object that reads the yaml file
// and initializes the necessary resources.
renderer, err := decorender.NewRenderer("./layout.yaml", &decorender.Options{
	// Needed whenever the layout refers to a local image or font file.
	// Paths in the layout are resolved against this.
	LocalFiles: os.DirFS("./assets"),

	// Optional. Customizes how external images are fetched. The default
	// downloads each one once and keeps it in the system temp directory.
	ExternalImage: nil,

	// Optional. Turns off the in-memory cache of decoded and scaled images.
	NoImageCache: false,
})

// decorender.NewRendererWithTemplate(yamlBytes, opts) does the same from a
// template already in memory.

// A renderer can then be used many times, with different data, and from
// several goroutines at once.

// Render hands back a pooled image. Call release when you are done with it,
// otherwise the buffer is never reused.
img, release, err := renderer.Render(yourData, nil)
defer release()

// The writing variants encode and release for you.
err = renderer.RenderAndWrite(yourData, decorender.EncodeFormatPNG, writer, nil)

// RenderToFile picks the format from the extension: .png, .jpg or .jpeg.
err = renderer.RenderToFile(yourData, "result.jpg", &decorender.RenderOptions{
	// JPEG quality as a fraction of 1. Defaults to 0.95.
	Quality: 0.8,

	// Renders the layout's `sample` instead of yourData. For previewing
	// only - this is what the dev server uses.
	UseSample: false,
})
```

## Format

Almost every value below can be an expression instead of a literal - see
*Templates with Expr*. The exception is `fontFaces`, which is read once when the
renderer is built, before there is any data to evaluate against.

### Root

`size`, `scale`, `fontFaces` and `sample` are read from the root node only. Every
other key works on any node, including the root.

```yaml
size: 1000 1000       # - Size of the result image. Leave it out, or give only one
                      #   axis with width/height, to size the image around its content.
scale: 2              # - Multiplier applied to the whole layout (e.g. 0.5, 1.5, 10).
fontFaces:            # - Font faces used in the layout. Roboto 400 is built in and
  - family: Inter     #   is the default family; declaring it replaces the built-in one.
    style: italic     # - normal (default) or italic.
    weight: 400
    file: ./Inter-italic-400.ttf
                      # - One file is one face. A variable font renders as its default
                      #   instance and nothing else, so declaring a weight that only
                      #   its axes could produce is refused rather than drawn wrong.
sample:               # - Arbitrary object used to preview the layout. Rendered only
                      #   when RenderOptions.UseSample is set, which is what the dev
                      #   server does.
inner:                # - Child nodes.
```

### Any node

```yaml
id: header            # - Optional name. Appears in error messages.
only: "1"             # - Debug aid: renders this node alone and discards the rest of
                      #   the layout. Any non-empty value turns it on.

# Size and position

size: 100% 100%       # - Width and height. A single value applies to both.
width: 50%            # - Either axis on its own. Overrides the matching part of size,
height: 30            #   so the other axis can size itself around the content.
padding: 10 20        # - Padding for children. 1, 2, 3 or 4 values, in the CSS order:
                      #   all / vertical horizontal / top horizontal bottom / t r b l.
absolute: left        # - Anchors the node to its parent at the given position, inside
                      #   the parent's padding, e.g. left - at center left,
                      #   right bottom - at that corner, left right - stretched
                      #   horizontally. Each direction takes an optional offset,
                      #   e.g. left/-10 top/55.
offset: left/5 top/-2 # - Shifts the node after it has been positioned. Only left and
                      #   top are honoured; right and bottom are ignored.
rotate: 45            # - Rotation in degrees. The node is drawn to its own image and
                      #   rotated, so this is the most expensive property here.

# Children

innerDirection: row   # - row, or column (default) - how children are laid out.
justify: end          # - start (default), center, end, space-between or space-evenly
                      #   - how children are distributed along innerDirection.
innerColumnAlign: right # - left (default), center or right - horizontal alignment of
                      #   children of a column.
innerRowAlign: center # - top (default), center or bottom - vertical alignment of
                      #   children of a row, within the height of that row. A row is
                      #   as tall as its tallest child.
innerWrap: none       # - wrap (default) or none - whether a row wraps when it runs out
                      #   of width.
innerGap: 5           # - Minimal gap between children.

# Paint

bkgColor: salmon      # - Background color. A predefined name, 0xaabbcc, 0xaabbccff,
                      #   rgb(129, 199, 132) or rgba(239, 83, 80, 0.55). A color that
                      #   does not parse is an error, not a transparent element.
border: 2 salmon inset # - Width, color, placement, sides and dash pattern, in any
                      #   order and all optional. outset (default) grows the node by
                      #   the border width, inset draws inside it, center straddles
                      #   the edge.
                      # - Naming top, right, bottom or left draws those sides only,
                      #   so `border: 1 gray top` is a rule along the top edge and
                      #   a node one pixel high is a horizontal line.
                      # - dashed breaks the line into dashes and gaps of four times
                      #   the border width. dashed/10/4/2/4 sets the lengths itself,
                      #   alternating drawn and skipped like the array given to
                      #   canvas setLineDash: at most 8 of them, and an odd number
                      #   repeats with drawn and skipped swapped, so dashed/5/10/5
                      #   runs 5 on, 10 off, 5 on, 5 off, 10 on, 5 off.
borderRadius: 20      # - Border radii. 1 to 4 values, like padding (e.g. 15 66).
bkgImage: pic.jpeg    # - Image for the node background. A local file resolved through
                      #   Options.LocalFiles, or an external one starting with https://
bkgImageSize: cover   # - cover (default) or contain.

# Text

text: Hello           # - Text, wrapped if needed. Hyphens are break opportunities, and
                      #   &nbsp; holds a pair of words together.
color: black          # - Color of text. Inherited by all children.
fontColor: black      # - Alias of color. When both are given, color wins.
font: Inter 23 400    # - Family, size, weight and italic/normal, in any order and all
                      #   optional. The first number is the size, the second the weight.
fontFamily: Inter     # - The same four, each on its own, for when an expression should
fontSize: 23          #   drive only one of them. These override font.
fontWeight: 400
fontStyle: italic
lineHeight: 30        # - Height of one line of text. Inherited. Defaults to a value
                      #   derived from the font size.

# Repetition

forEach: rows         # - Name of a field in the current data holding a list: the node
                      #   is repeated once per element. A number repeats it that many
                      #   times instead.
                      # - An expression is repeated over whatever it evaluates to: a
                      #   list, a count, or a flag. `forEach: ~ value.goals` draws the
                      #   node only in the panels that have one, and `~ filter(rows,
                      #   .visible)` only over the rows worth drawing.
```

### Units

Wherever a number is expected:

| form | meaning |
| --- | --- |
| `40` | pixels |
| `50%` | percentage of the parent's corresponding axis |
| `0.5w` | multiple of the parent's width |
| `0.5h` | multiple of the parent's height |

See `test.yaml` and `test.png` for more examples.

`testdata/golden.yaml` exercises every key above except `scale` and `only`, which
cannot share an image with the rest and are asserted in unit tests instead. It is
rendered to `testdata/golden.png`, which the test suite compares against. After an
intentional rendering change, regenerate it and inspect the result before
committing:

```
go test -run TestGoldenRender -update-golden
```

### Templates with Expr

In almost any field, you can use an expression instead of a fixed value.
`github.com/expr-lang/expr` is used. Write `~ Field` to reach a field of the data
the layout is rendered with.

Inside a `forEach`, three more names are available, on the repeated node and on all
of its descendants: `value` is the current element, `index` its position, and
`parent` the collection it came from.

Fields of the data are looked up the way expr looks them up, in expressions and in
`forEach` alike: an `expr:"rows"` tag renames a field, and `expr:"-"` hides it. So a
layout can stay lowercase over Go structs.

```yaml
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
```

Quoting is worth care: it is the whole value that is quoted, `'~ "top/" + string(y)'`
and not `~ '"top/" + string(y)'`. The second form hands expr a string literal, which
evaluates to the text of the expression itself.

An expression that does not compile, or that fails against the data, fails the render
and names the property it came from. So does a property that cannot mean anything,
such as a direction `absolute` does not know. Both used to fall back to a default
silently, which left a node unanchored, unsized or uncolored with nothing to say why.
Numbers stay lenient: a value that does not parse still falls back, since `10px` or an
empty string are a matter of degree rather than a mistake.

## Performance

Almost everything is written with performance considerations in mind.
 * No rendering libraries are used, everything is drawn with the standard library and golang.org/x/image.
 * Work with all heavy objects (internal node tree, buffers for images, rasterizers) is done through sync.Pool.
 * A small LRU cache is used for frequently used images. Also, an LRU cache is used for frequently used masks (which, for example, are used for drawing rounded rectangles).
 * Downloaded external images are stored in the system's tmp directory and are not downloaded again upon reuse.

Take into consideration:
 * If possible use images with exact size as will be appear in layout. Scaling is quite expensive operation.
