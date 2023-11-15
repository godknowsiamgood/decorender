package decorender

import (
	"decorender/draw"
	"decorender/fonts"
	"decorender/layout"
	"decorender/parsing"
	"decorender/render"
	"errors"
	"gopkg.in/yaml.v3"
	"image"
	"os"
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
