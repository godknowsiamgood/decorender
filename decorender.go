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
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

var NothingToRenderErr = errors.New("nothing to render")

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

func (r *Renderer) Render(userData any) (image.Image, error) {
	nodes := layout.Do(r.root, userData, r.drawer)
	if nodes == nil {
		return nil, NothingToRenderErr
	}

	render.Do(nodes[0], r.drawer)

	return r.drawer.RetrieveImage(), nil
}

func (r *Renderer) RenderToFile(userData any, fileName string) error {
	img, err := r.Render(userData)
	if err != nil {
		return err
	}

	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".jpg", ".jpeg":
		return jpeg.Encode(file, img, &jpeg.Options{Quality: 95})
	case ".png":
		return png.Encode(file, img)
	default:
		return fmt.Errorf("unsupported file format")
	}
}
