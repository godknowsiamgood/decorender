package decorender

import (
	"errors"
	"fmt"
	"github.com/godknowsiamgood/decorender/draw"
	"github.com/godknowsiamgood/decorender/fonts"
	"github.com/godknowsiamgood/decorender/layout"
	"github.com/godknowsiamgood/decorender/parsing"
	"github.com/godknowsiamgood/decorender/render"
	"gopkg.in/yaml.v3"
	"image/jpeg"
	"image/png"
	"io"
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

type Renderer struct {
	root   parsing.Node
	drawer draw.Drawer
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
		root:   root,
		drawer: &draw.DefaultDrawer{},
	}

	if err = fonts.LoadFaces(root.FontFaces); err != nil {
		return nil, err
	}

	return renderer, nil
}

func (r *Renderer) Render(userData any, format EncodeFormat, w io.Writer) error {
	nodes := layout.Do(r.root, userData, r.drawer)
	if nodes == nil {
		return NothingToRenderErr
	}

	render.Do(nodes[0], r.drawer)

	layout.Release(nodes)

	img := r.drawer.RetrieveImage()
	defer r.drawer.ReleaseImage()

	if w == nil {
		return nil
	}

	switch format {
	case EncodeFormatPNG:
		return png.Encode(w, img)
	case EncodeFormatJPG:
		return jpeg.Encode(w, img, &jpeg.Options{Quality: 95})
	}

	return nil
}

func (r *Renderer) RenderToFile(userData any, fileName string) error {
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

	err = r.Render(userData, format, file)
	if err != nil {
		return err
	}

	return nil
}
