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

type Cache struct {
	prevUsedFaceDescription fonts.FaceDescription
	prevUsedFace            font.Face
	prevUsedFaceMx          sync.Mutex

	scaledResourceImages gcache.Cache
	roundedRectMasks     gcache.Cache

	keysBuildersPool *builderpool.BuilderPool
}

func (c *Cache) addRoundedRectMask(w int, h int, radii utils.FourValues, alpha *image.Alpha) {
	key := c.roundedRectMaskKey(w, h, radii)
	_ = c.roundedRectMasks.SetWithExpire(key, alpha, time.Minute*15)
}

func (c *Cache) getRoundedRectMask(w int, h int, radii utils.FourValues) *image.Alpha {
	key := c.roundedRectMaskKey(w, h, radii)
	if c.roundedRectMasks.Has(key) {
		v, _ := c.roundedRectMasks.Get(key)
		return v.(*image.Alpha)
	} else {
		return nil
	}
}

func (c *Cache) roundedRectMaskKey(w int, h int, radii utils.FourValues) string {
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

func NewCache() *Cache {
	return &Cache{
		scaledResourceImages: gcache.New(10).LRU().Build(),
		roundedRectMasks:     gcache.New(10).LRU().Build(),
		keysBuildersPool:     builderpool.New(),
	}
}
