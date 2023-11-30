package resources

import (
	"github.com/godknowsiamgood/decorender/utils"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type ExternalImage interface {
	Prefetch(path string)
	Get(path string) ([]byte, error)
}

type DefaultExternalImage struct {
	downloadsInProgress   map[string]chan struct{}
	downloadsInProgressMx sync.Mutex
}

func NewDefaultExternalImage() ExternalImage {
	return &DefaultExternalImage{
		downloadsInProgress: make(map[string]chan struct{}),
	}
}

func (d *DefaultExternalImage) Prefetch(path string) {
	downloadLocalFileName := tempLocalNameForDownloadableResource(path)
	if downloadLocalFileName == "" {
		return
	}

	d.downloadsInProgressMx.Lock()
	defer d.downloadsInProgressMx.Unlock()

	if _, has := d.downloadsInProgress[path]; has {
		// downloading in progress
		return
	}

	if _, err := os.Stat(downloadLocalFileName); !os.IsNotExist(err) {
		// file already existed and downloaded
		return
	}

	done := make(chan struct{})
	d.downloadsInProgress[path] = done

	go func() {
		defer close(done)

		defer func() {
			d.downloadsInProgressMx.Lock()
			delete(d.downloadsInProgress, path)
			d.downloadsInProgressMx.Unlock()
		}()

		resp, err := http.Get(path)
		if err != nil {
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return
		}
		err = os.WriteFile(downloadLocalFileName, bytes, 0644)
		if err != nil {
			return
		}
	}()
}

func (d *DefaultExternalImage) Get(path string) ([]byte, error) {
	downloadLocalFileName := tempLocalNameForDownloadableResource(path)
	if downloadLocalFileName != "" {
		d.downloadsInProgressMx.Lock()
		done, exists := d.downloadsInProgress[path]
		d.downloadsInProgressMx.Unlock()

		if exists {
			<-done
		}

		path = downloadLocalFileName
	}

	return os.ReadFile(path)
}

func tempLocalNameForDownloadableResource(fileName string) string {
	if IsLocalResource(fileName) {
		return ""
	}

	tmpDir := os.TempDir()
	localFileName := filepath.Join(tmpDir, utils.GetSha256(fileName))

	return localFileName
}
