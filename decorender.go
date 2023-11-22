package decorender

import (
	"errors"
	"fmt"
	"github.com/godknowsiamgood/decorender/fonts"
	"github.com/godknowsiamgood/decorender/layout"
	"github.com/godknowsiamgood/decorender/parsing"
	"github.com/godknowsiamgood/decorender/render"
	"github.com/godknowsiamgood/decorender/utils"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

var NothingToRenderErr = errors.New("nothing to render")

type EncodeFormat uint

const (
	EncodeFormatPNG EncodeFormat = iota
	EncodeFormatJPG
)

type Options struct {
	UseSample bool
	Quality   float64
}

type Renderer struct {
	root        parsing.Node
	renderCache *render.Cache
}

func NewRenderer(yamlFileName string) (*Renderer, error) {
	content, err := os.ReadFile(yamlFileName)
	if err != nil {
		return nil, err
	}

	node := yaml.Node{}
	err = yaml.Unmarshal(content, &node)
	if err != nil {
		return nil, err
	}

	var root parsing.Node

	err = node.Decode(&root)
	if err != nil {
		return nil, err
	}

	root = parsing.KeepDebugNodes(root)

	renderer := &Renderer{
		root:        root,
		renderCache: render.NewCache(),
	}

	if err = fonts.LoadFaces(root.FontFaces); err != nil {
		return nil, err
	}

	return renderer, nil
}

func (r *Renderer) Render(userData any, format EncodeFormat, w io.Writer, opts *Options) error {
	if opts != nil && opts.UseSample {
		userData = r.root.Sample
	}

	nodes, err := layout.Do(r.root, userData)
	if err != nil {
		return err
	}

	root := nodes.GetRootNode()
	if root == nil || root.Size.W < 0.1 || root.Size.H < 0.1 {
		return NothingToRenderErr
	}

	dst, err := render.Do(nodes, r.renderCache)
	if err != nil {
		return err
	}

	defer func() {
		utils.ReleaseImage(dst)
	}()

	layout.Release(nodes)

	if w == nil {
		return nil
	}

	switch format {
	case EncodeFormatPNG:
		return png.Encode(w, dst)
	case EncodeFormatJPG:
		return jpeg.Encode(w, dst, &jpeg.Options{
			Quality: lo.Ternary(opts == nil || opts.Quality < math.SmallestNonzeroFloat64, 95, int(100*opts.Quality)),
		})
	}

	return nil
}

func (r *Renderer) RenderToFile(userData any, fileName string, opts *Options) error {
	var format EncodeFormat
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".jpg", ".jpeg":
		format = EncodeFormatJPG
	case ".png":
		format = EncodeFormatPNG
	default:
		return fmt.Errorf("unsupported file format")
	}

	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	err = r.Render(userData, format, file, opts)
	if err != nil {
		return err
	}

	return nil
}
