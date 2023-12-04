package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/godknowsiamgood/decorender"
	"github.com/samber/lo"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"time"
)

//go:embed index.html
var indexHtml embed.FS

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run [layout.yaml]")
		os.Exit(1)
	}
	layoutFileName := os.Args[1]

	if _, err := os.Stat(layoutFileName); os.IsNotExist(err) {
		fmt.Println("Layout file does not exist")
		return
	}

	var renderer *decorender.Decorender
	var rendererErr error

	var ver = rand.Intn(999999)

	var info string
	var mx sync.Mutex

	renderMultipleTimes := func(encodeTo decorender.EncodeFormat) (time.Duration, int, error) {
		var minTime time.Duration = math.MaxInt64
		var bytesCount int

		options := decorender.RenderOptions{UseSample: true}

		for i := 0; i < 3; i++ {
			start := time.Now()

			var writer CountingWriter

			rendererErr = renderer.RenderAndWrite(nil, encodeTo, &writer, &options)
			dur := time.Now().Sub(start)
			if dur < minTime {
				minTime = dur
			}

			if rendererErr != nil || dur > time.Second*3 {
				return 0, 0, rendererErr
			}

			bytesCount = writer.count
		}

		return minTime, bytesCount, nil
	}

	updateRenderer := func() {
		mx.Lock()
		defer mx.Unlock()

		info = ""
		renderer, rendererErr = decorender.NewRenderer(layoutFileName, nil)
		if rendererErr == nil {
			var timeRender time.Duration
			var timeWithPNGEncode time.Duration
			var timeWithJGPEncode time.Duration

			var pngBytes int
			var jpegBytes int

			timeRender, _, rendererErr = renderMultipleTimes(decorender.EncodeFormatNone)
			if rendererErr == nil {
				timeWithPNGEncode, pngBytes, rendererErr = renderMultipleTimes(decorender.EncodeFormatPNG)
				if rendererErr == nil {
					timeWithJGPEncode, jpegBytes, rendererErr = renderMultipleTimes(decorender.EncodeFormatJPG)
				}
			}

			if rendererErr == nil {
				ver += 1
				info = fmt.Sprintf("render in %.3fs, to png +%.3fs (%s), to jpg +%.3fs (%s)",
					float64(timeRender.Milliseconds())/1000.0,
					float64(timeWithPNGEncode.Milliseconds())/1000.0, bytesToHumanReadable(pngBytes),
					float64(timeWithJGPEncode.Milliseconds())/1000.0, bytesToHumanReadable(jpegBytes))
			}
		}
	}

	go func() {
		watchFile(layoutFileName, func() {
			updateRenderer()
		})
	}()

	updateRenderer()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.FileServer(http.FS(indexHtml)).ServeHTTP(w, r)
	})

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		mx.Lock()
		defer mx.Unlock()

		w.Header().Set("Content-Type", "application/json")
		jsonData, _ := json.Marshal(map[string]string{
			"ver":  strconv.Itoa(ver),
			"info": info,
			"err":  lo.Ternary(rendererErr != nil, fmt.Sprintf("%v", rendererErr), ""),
		})
		_, _ = w.Write(jsonData)
	})

	http.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		mx.Lock()
		defer mx.Unlock()

		if renderer == nil {
			return
		}

		w.Header().Set("Content-Type", "image/png")
		_ = renderer.RenderAndWrite(nil, decorender.EncodeFormatPNG, w, &decorender.RenderOptions{UseSample: true})
	})

	go func() {
		err := openBrowser("http://localhost:8089")
		if err != nil {
			log.Printf("Failed to open browser: %v", err)
		}
	}()

	log.Println("Decorender dev server is running at http://localhost:8089")
	err := http.ListenAndServe(":8089", nil)
	if err != nil {
		log.Fatalf("Failed to start decorender server: %v", err)
	}
}

func openBrowser(url string) error {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	return err
}

func watchFile(filePath string, action func()) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = watcher.Close()
	}()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					action()
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	err = watcher.Add(filePath)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

type CountingWriter struct {
	count int
	mx    sync.Mutex
}

func (cw *CountingWriter) Write(p []byte) (n int, err error) {
	cw.mx.Lock()
	defer cw.mx.Unlock()
	cw.count += len(p)
	return len(p), nil
}

func bytesToHumanReadable(bytes int) string {
	const (
		KB = 1024
		MB = 1024 * KB
	)

	if bytes < MB {
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	}
	return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
}
