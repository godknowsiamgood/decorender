package render

import (
	"bytes"
	"fmt"
	"github.com/bluele/gcache"
	"github.com/godknowsiamgood/decorender/internal/layout"
	resources_internal "github.com/godknowsiamgood/decorender/internal/resources"
	"github.com/godknowsiamgood/decorender/internal/utils"
	"github.com/godknowsiamgood/decorender/resources"
	"github.com/nasa9084/go-builderpool"
	"image"
	"io"
	"io/fs"
	"time"
)

// Cache used to keep cached values through all renders
type Cache struct {
	scaledResourceImages gcache.Cache
	roundedRectMasks     gcache.Cache

	externalImages resources.ExternalImage
	localImages    fs.FS

	keysBuildersPool *builderpool.BuilderPool

	mx utils.ShardedMutex
}

func NewCache(externalImages resources.ExternalImage, localImages fs.FS, imageCacheSize int) *Cache {
	cache := &Cache{
		roundedRectMasks: gcache.New(30).LRU().Build(),
		keysBuildersPool: builderpool.New(),

		externalImages: externalImages,
		localImages:    localImages,
	}

	if imageCacheSize > 0 {
		cache.scaledResourceImages = gcache.New(imageCacheSize).LRU().Build()
	}

	return cache
}

func (c *Cache) useRoundedMaskImage(w float64, h float64, radii utils.FourValues, onCreate func(mask *image.Alpha), onUse func(mask *image.Alpha)) {
	key := utils.HashDJB2Num(w, h, radii[0], radii[1], radii[2], radii[3])

	c.mx.Lock(key)
	defer c.mx.Unlock(key)

	img, _ := c.roundedRectMasks.Get(key)
	var alphaImg *image.Alpha
	if img == nil {
		alphaImg = image.NewAlpha(image.Rect(0, 0, int(w), int(h)))
		onCreate(alphaImg)
	} else {
		alphaImg, _ = img.(*image.Alpha)
	}

	onUse(alphaImg)

	_ = c.roundedRectMasks.SetWithExpire(key, alphaImg, time.Minute*15)
}

func (c *Cache) useScaledImage(fileName string, w, h float64, sizeType layout.BkgImageSizeType, onUse func(img image.Image)) error {
	key := utils.HashDJB2(fileName) + utils.HashDJB2Num(w, h, float64(sizeType))

	c.mx.Lock(key)
	defer c.mx.Unlock(key)

	var img image.Image

	if c.scaledResourceImages != nil {
		v, _ := c.scaledResourceImages.Get(key)
		img, _ = v.(image.Image)
	}

	if img == nil {
		imageBytes, err := c.getResourceContent(fileName)
		if err != nil {
			return fmt.Errorf("cant get image %v: %w", fileName, err)
		}

		imgReader := bytes.NewReader(imageBytes)
		srcImg, _, err := image.Decode(imgReader)
		if err != nil {
			return fmt.Errorf("cant decode image %v: %w", fileName, err)
		}

		if srcImg.Bounds().Dx() == int(w) && srcImg.Bounds().Dy() == int(h) {
			img = srcImg
		} else {
			img = scaleAndCropImage(srcImg, w, h, sizeType)
		}
	}

	onUse(img)

	if c.scaledResourceImages != nil {
		_ = c.scaledResourceImages.SetWithExpire(key, img, time.Minute*15)
	}

	return nil
}

func (c *Cache) getResourceContent(fileName string) ([]byte, error) {
	if resources_internal.IsLocalResource(fileName) {
		f, err := c.localImages.Open(fileName)
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = f.Close()
		}()

		return io.ReadAll(f)
	} else {
		return c.externalImages.Get(fileName)
	}
}
