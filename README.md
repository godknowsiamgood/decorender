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
// and initializes the necessary resources
renderer, err := decorender.NewRenderer("./layout.yaml", &decorender.Options{
	...
})

// Then it can be used multiple times 
// with different data and concurrent-safely.
img, _ := renderer.Render(yourData, &decorender.RenderOptions{})
renderer.RenderAndWrite(yourData, decorender.EncodeFormatPNG, writer, &decorender.RenderOptions{})
renderer.RenderToFile(yourData, "result.jpg", &decorender.RenderOptions{})
```
## Concept
NewRenderer parses a YAML file with layout, validates the data, compiles templates, initializes the necessary data and caches, and returns an object that is then used for rendering with data.

Usually, one YAML layout is enough to modify the layout using templates.

## Format

```yaml
size: 1000 1000       # - Optional size of result image in pixels.
scale: 2              # - Optional multiplier of result image (e.g. 0.5, 1.5, 10).
fontFaces:            # - Font faces that will be used in layout.
  - family: Inter
    style: italic
    weight: 400
    file: ./Inter-italic-400.ttf
sample:                 # - Any arbitrary object to test layout with expr templates.
inner:                  # - Child nodes.
  - size: 100% 100%     # - Size. Use absolute values, or percents.
    bkgColor: salmon    # - Background color. Use predefined colors, or 0xaabbcc, 0xaabbccff.
    color: black        # - Color of text. This property is inherited to all children.
    font: Inter 23 400  # - Current font in format <family> <size> <weight>. Every part is optional,
                        #   except single number will be interpreted as size.
    text: Hello         # - Text that will be wrapped if needed.
    innerDirection: row # - Values row/column instructs how children will be located.
    justify: end        # - Values start/center/end/space-between - how children will be positioned.
    innerGap: 5         # - Minimal gap between children.
    padding: 10 20      # - Padding for children.
    borderRadius: 20    # - Border radii (e.g. 15 66, 10 20 30 40).
    absolute: left      # - Instructs how element should be anchored to parent at desired position
                        #   with respect of parent padding, e.g.
                        #   left - at center left, right bottom - at corner,
                        #   left right - node will be stretched horizontally.
                        #   Also you can specify offset for each direction, e.g. left/-10 top/55.
    bkgImage:           # - Image for element background. Use local or external file starting with https://...
    bkgImageSize: cover # - Values cover/contain
    forEach: Array      # - Name of field in user data. Node will be replicated accordingly.
```
See `test.yaml` and `test.png` for more examples.

### Templates with Expr
In almost any field, you can use an expression instead of a fixed one. `github.com/antonmedv/expr` is used. Just write `~ Field` to access to field. In the context of loops there are variables `value`, `index` and `parent`.

## Performance

Almost everything is written with performance considerations in mind.
 * No rendering libraries are used, everything is drawn with standard libraries. The only exception is github.com/disintegration/imaging for rotations.
 * Work with all heavy objects (internal node tree, buffers for images, rasterizers) is done through sync.Pool.
 * A small LRU cache is used for frequently used images. Also, an LRU cache is used for frequently used masks (which, for example, are used for drawing rounded rectangles).
 * Downloaded external images are stored in the system's tmp directory and are not downloaded again upon reuse.

Take into consideration:
 * If possible use images with exact size as will be appear in layout. Scaling is quite expensive operation.
