package decorender

import (
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/godknowsiamgood/decorender/internal/fonts"
	"github.com/godknowsiamgood/decorender/internal/layout"
	"github.com/godknowsiamgood/decorender/internal/parsing"
	"github.com/godknowsiamgood/decorender/internal/render"
	resources_internal "github.com/godknowsiamgood/decorender/internal/resources"
	"github.com/godknowsiamgood/decorender/internal/utils"
	"github.com/godknowsiamgood/decorender/resources"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
)

var NothingToRenderErr = errors.New("nothing to render")

type EncodeFormat uint

const (
	EncodeFormatNone EncodeFormat = iota
	EncodeFormatPNG
	EncodeFormatJPG
)

// RenderOptions are options for particular render
type RenderOptions struct {
	// UseSample instructs to use sample object in yaml file.
	// Should be used for debug purposes only or for dev server
	UseSample bool
	// Quality sets quality for encoding formats that supports quality
	Quality float64
}

// Options are options for Decorender instance
type Options struct {
	// ExternalImage used to customize default behavior
	// how external images downloaded
	ExternalImage resources.ExternalImage

	// LocalFiles will be used to take local files
	LocalFiles fs.FS

	// When NoImageCache is true, no decoded and scaled images are kept in memory cache.
	// Default cache is fixed size LRU.
	NoImageCache bool
}

type Decorender struct {
	root          parsing.Node
	layoutCache   *layout.Cache
	renderCache   *render.Cache
	externalImage resources.ExternalImage
	localFiles    fs.FS
}

func NewRenderer(yamlFileName string, opts *Options) (*Decorender, error) {
	content, err := os.ReadFile(yamlFileName)
	if err != nil {
		return nil, err
	}
	return NewRendererWithTemplate(content, opts)
}

func NewRendererWithTemplate(template []byte, opts *Options) (*Decorender, error) {
	node := yaml.Node{}
	err := yaml.Unmarshal(template, &node)
	if err != nil {
		return nil, err
	}

	var root parsing.Node

	err = node.Decode(&root)
	if err != nil {
		return nil, err
	}

	root = parsing.KeepDebugNodes(root)

	dr := &Decorender{
		root: root,
	}

	if opts != nil && opts.ExternalImage != nil {
		dr.externalImage = opts.ExternalImage
	} else {
		dr.externalImage = resources_internal.NewDefaultExternalImage()
	}

	if opts != nil && opts.LocalFiles != nil {
		dr.localFiles = opts.LocalFiles
	} else {
		dr.localFiles = os.DirFS(".")
	}

	imagesCacheSize := lo.Ternary(opts != nil && opts.NoImageCache, 0, 30)

	dr.layoutCache = layout.NewCache()
	dr.renderCache = render.NewCache(dr.externalImage, dr.localFiles, imagesCacheSize)

	if err = fonts.LoadFaces(root.FontFaces, dr.localFiles); err != nil {
		return nil, err
	}

	return dr, nil
}

func (r *Decorender) RenderAndWrite(userData any, format EncodeFormat, w io.Writer, opts *RenderOptions) error {
	dst, release, err := r.Render(userData, opts)
	if err != nil {
		return err
	}
	defer release()

	if w != nil {
		switch format {
		case EncodeFormatPNG:
			return png.Encode(w, dst)
		case EncodeFormatJPG:
			return jpeg.Encode(w, dst, &jpeg.Options{
				Quality: lo.Ternary(opts == nil || opts.Quality < math.SmallestNonzeroFloat64, 95, int(100*opts.Quality)),
			})
		default:
		}
	}

	return nil
}

// Render renders layout to image. Images are pooled resource,
// so make sure to call release function when you are done with image.
func (r *Decorender) Render(userData any, opts *RenderOptions) (dst image.Image, release func(), err error) {
	userData = lo.Ternary(opts != nil && opts.UseSample, r.root.Sample, userData)

	// First phase is layout

	nodes, err := layout.Do(r.root, userData, r.externalImage, r.layoutCache)
	if err != nil {
		return nil, nil, err
	}

	root := nodes.GetRootNode()
	if root == nil || root.Size.W < 0.01 || root.Size.H < 0.01 {
		return nil, nil, NothingToRenderErr
	}

	// Second phase is render

	dst, err = render.Do(nodes, r.renderCache)
	if err != nil {
		return nil, nil, err
	}

	layout.Release(nodes)

	return dst, func() {
		utils.ReleaseImage(dst)
	}, nil
}

func (r *Decorender) RenderToFile(userData any, fileName string, opts *RenderOptions) error {
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

	err = r.RenderAndWrite(userData, format, file, opts)
	if err != nil {
		return err
	}

	return nil
}
