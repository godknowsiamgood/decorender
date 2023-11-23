package resources

import (
	"github.com/godknowsiamgood/decorender/utils"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var downloadsInProgress map[string]chan struct{}
var downloadsInProgressMx sync.Mutex

func init() {
	downloadsInProgress = make(map[string]chan struct{})
}

func tempLocalNameForDownloadableResource(fileName string) string {
	if IsLocalResource(fileName) {
		return ""
	}

	tmpDir := os.TempDir()
	localFileName := filepath.Join(tmpDir, utils.GetSha256(fileName))

	return localFileName
}

func PrefetchResource(fileName string) {
	downloadLocalFileName := tempLocalNameForDownloadableResource(fileName)
	if downloadLocalFileName == "" {
		return
	}

	downloadsInProgressMx.Lock()
	defer downloadsInProgressMx.Unlock()

	if _, has := downloadsInProgress[fileName]; has {
		// downloading in progress
		return
	}

	if _, err := os.Stat(downloadLocalFileName); !os.IsNotExist(err) {
		// file already existed and downloaded
		return
	}

	done := make(chan struct{})
	downloadsInProgress[fileName] = done

	go func() {
		defer close(done)

		defer func() {
			downloadsInProgressMx.Lock()
			delete(downloadsInProgress, fileName)
			downloadsInProgressMx.Unlock()
		}()

		resp, err := http.Get(fileName)
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

func GetResourceContent(fileName string) ([]byte, error) {
	downloadLocalFileName := tempLocalNameForDownloadableResource(fileName)
	if downloadLocalFileName != "" {
		downloadsInProgressMx.Lock()
		done, exists := downloadsInProgress[fileName]
		downloadsInProgressMx.Unlock()

		if exists {
			<-done
		}

		fileName = downloadLocalFileName
	}

	return os.ReadFile(fileName)
}

func IsLocalResource(fileName string) bool {
	return !strings.HasPrefix(fileName, "http://") && !strings.HasPrefix(fileName, "https://")
}
