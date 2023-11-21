package render

import (
	"github.com/bluele/gcache"
	"github.com/godknowsiamgood/decorender/fonts"
	"github.com/godknowsiamgood/decorender/utils"
	"github.com/nasa9084/go-builderpool"
	"golang.org/x/image/font"
	"image"
	"strconv"
	"sync"
	"time"
)

type cache struct {
	prevUsedFaceDescription fonts.FaceDescription
	prevUsedFaceOffset      float64
	prevUsedFaceDrawer      *font.Drawer
	prevUsedFaceMx          sync.Mutex

	scaledImages     gcache.Cache
	roundedRectMasks gcache.Cache

	keysBuildersPool *builderpool.BuilderPool
}

func (c *cache) release() {
	c.scaledImages.Purge()
	c.prevUsedFaceDrawer = nil
	c.roundedRectMasks.Purge()
}

func (c *cache) addRoundedRectMask(w int, h int, radii utils.FourValues, alpha *image.Alpha) {
	key := c.roundedRectMaskKey(w, h, radii)
	_ = c.roundedRectMasks.SetWithExpire(key, alpha, time.Minute*15)
}

func (c *cache) getRoundedRectMask(w int, h int, radii utils.FourValues) *image.Alpha {
	key := c.roundedRectMaskKey(w, h, radii)
	if c.roundedRectMasks.Has(key) {
		v, _ := c.roundedRectMasks.Get(key)
		return v.(*image.Alpha)
	} else {
		return nil
	}
}

func (c *cache) roundedRectMaskKey(w int, h int, radii utils.FourValues) string {
	sb := c.keysBuildersPool.Get()
	defer func() {
		c.keysBuildersPool.Release(sb)
	}()

	sb.Reset()
	sb.WriteString(strconv.Itoa(w))
	sb.WriteString("-")
	sb.WriteString(strconv.Itoa(h))
	sb.WriteString("-")
	sb.WriteString(strconv.Itoa(int(radii[0] * 100)))
	sb.WriteString(strconv.Itoa(int(radii[1] * 100)))
	sb.WriteString(strconv.Itoa(int(radii[2] * 100)))
	sb.WriteString(strconv.Itoa(int(radii[3] * 100)))
	return sb.String()
}

func newCache() cache {
	return cache{
		scaledImages: gcache.New(10).LRU().EvictedFunc(func(_ any, v any) {
			img, _ := v.(*image.RGBA)
			utils.ReleaseImage(img)
		}).Build(),
		roundedRectMasks: gcache.New(10).LRU().EvictedFunc(func(_ any, v any) {
			img, _ := v.(*image.Alpha)
			utils.ReleaseImage(img)
		}).Build(),
		keysBuildersPool: builderpool.New(),
	}
}
