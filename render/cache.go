package render

import (
	"bytes"
	"fmt"
	"github.com/bluele/gcache"
	"github.com/godknowsiamgood/decorender/layout"
	"github.com/godknowsiamgood/decorender/resources"
	"github.com/godknowsiamgood/decorender/utils"
	"github.com/nasa9084/go-builderpool"
	"image"
	"time"
)

type Cache struct {
	scaledResourceImages gcache.Cache
	roundedRectMasks     gcache.Cache

	mx utils.ShardedMutex

	keysBuildersPool *builderpool.BuilderPool
}

func (c *Cache) useRoundedMaskImage(w float64, h float64, radii utils.FourValues, onCreate func(mask *image.Alpha), onUse func(mask *image.Alpha)) {
	key := utils.HashDJB2Num(w, h, radii[0], radii[1], radii[2], radii[3])

	c.mx.LockInt(key)
	defer c.mx.UnlockInt(key)

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

	c.mx.LockInt(key)
	defer c.mx.UnlockInt(key)

	v, _ := c.scaledResourceImages.Get(key)
	img, _ := v.(image.Image)

	if img == nil {
		imageBytes, err := resources.GetResourceContent(fileName)
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

	_ = c.scaledResourceImages.SetWithExpire(key, img, time.Minute*15)

	return nil
}

func NewCache() *Cache {
	return &Cache{
		scaledResourceImages: gcache.New(30).LRU().Build(),
		roundedRectMasks:     gcache.New(30).LRU().Build(),
		keysBuildersPool:     builderpool.New(),
	}
}
