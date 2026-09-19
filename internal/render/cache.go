package render

import (
	"bytes"
	"fmt"
	"github.com/godknowsiamgood/decorender/internal/layout"
	resources_internal "github.com/godknowsiamgood/decorender/internal/resources"
	"github.com/godknowsiamgood/decorender/internal/utils"
	"github.com/godknowsiamgood/decorender/resources"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"image"
	"io"
	"io/fs"
	"time"
)

// cacheTTL is how long an entry stays warm after its last use.
const cacheTTL = 15 * time.Minute

// roundedRectMaskCacheSize bounds the rounded-corner mask cache.
const roundedRectMaskCacheSize = 30

// Cache used to keep cached values through all renders
type Cache struct {
	scaledResourceImages *expirable.LRU[uint, image.Image]
	roundedRectMasks     *expirable.LRU[uint, *image.Alpha]

	externalImages resources.ExternalImage
	localImages    fs.FS

	mx utils.ShardedMutex
}

func NewCache(externalImages resources.ExternalImage, localImages fs.FS, imageCacheSize int) *Cache {
	cache := &Cache{
		roundedRectMasks: expirable.NewLRU[uint, *image.Alpha](roundedRectMaskCacheSize, nil, cacheTTL),

		externalImages: externalImages,
		localImages:    localImages,
	}

	if imageCacheSize > 0 {
		cache.scaledResourceImages = expirable.NewLRU[uint, image.Image](imageCacheSize, nil, cacheTTL)
	}

	return cache
}

func (c *Cache) useRoundedMaskImage(w float64, h float64, radii utils.FourValues, onCreate func(mask *image.Alpha), onUse func(mask *image.Alpha)) {
	key := utils.HashDJB2Num(w, h, radii[0], radii[1], radii[2], radii[3])

	c.mx.Lock(key)
	defer c.mx.Unlock(key)

	alphaImg, ok := c.roundedRectMasks.Get(key)
	if !ok {
		alphaImg = image.NewAlpha(image.Rect(0, 0, int(w), int(h)))
		onCreate(alphaImg)
	}

	onUse(alphaImg)

	c.roundedRectMasks.Add(key, alphaImg)
}

func (c *Cache) useScaledImage(fileName string, w, h float64, sizeType layout.BkgImageSizeType, onUse func(img image.Image)) error {
	key := utils.HashDJB2(fileName) + utils.HashDJB2Num(w, h, float64(sizeType))

	img, pooled, err := c.loadScaledImage(key, fileName, w, h, sizeType)
	if err != nil {
		return err
	}

	// Drawing happens outside the shard lock. Cached images are only ever read
	// after they are built, so concurrent renders need not queue behind each
	// other to draw the same image.
	onUse(img)

	if pooled {
		// Image caching is off, so nothing retained this buffer.
		utils.ReleaseImage(img)
	}

	return nil
}

// loadScaledImage returns the scaled image for key, decoding it on first use.
// pooled reports that the image came from the shared buffer pool and that no
// cache retained it, so the caller owns it and must release it after use.
func (c *Cache) loadScaledImage(key uint, fileName string, w, h float64, sizeType layout.BkgImageSizeType) (img image.Image, pooled bool, err error) {
	c.mx.Lock(key)
	defer c.mx.Unlock(key)

	if c.scaledResourceImages != nil {
		if img, ok := c.scaledResourceImages.Get(key); ok && img != nil {
			// The cache keeps this image, so the caller must not release it.
			return img, false, nil
		}
	}

	imageBytes, err := c.getResourceContent(fileName)
	if err != nil {
		return nil, false, fmt.Errorf("cant get image %v: %w", fileName, err)
	}

	srcImg, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return nil, false, fmt.Errorf("cant decode image %v: %w", fileName, err)
	}

	// Only a scaled image owns a pooled buffer; srcImg comes from the decoder
	// and its pixels must never be handed to the buffer pool.
	fromPool := false
	if srcImg.Bounds().Dx() == int(w) && srcImg.Bounds().Dy() == int(h) {
		img = srcImg
	} else {
		img = scaleAndCropImage(srcImg, w, h, sizeType)
		fromPool = true
	}

	if c.scaledResourceImages == nil {
		return img, fromPool, nil
	}

	c.scaledResourceImages.Add(key, img)

	return img, false, nil
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
