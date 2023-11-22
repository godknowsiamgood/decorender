package main

import (
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/godknowsiamgood/decorender"
	"github.com/samber/lo"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run [layout.yaml]")
		os.Exit(1)
	}
	layoutFileName := os.Args[1]

	if _, err := os.Stat(layoutFileName); os.IsNotExist(err) {
		fmt.Println("Layout file does not exist")
	}

	var renderer *decorender.Renderer
	var rendererErr error
	var ver int
	var mx sync.Mutex

	updateRenderer := func() {
		mx.Lock()
		defer mx.Unlock()

		renderer, rendererErr = decorender.NewRenderer(layoutFileName)
		if rendererErr == nil {
			rendererErr = renderer.Render(nil, decorender.EncodeFormatPNG, nil, nil)
			if rendererErr == nil {
				ver += 1
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
		http.ServeFile(w, r, "./cmd/index.html")
	})

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		mx.Lock()
		defer mx.Unlock()

		w.Header().Set("Content-Type", "application/json")
		jsonData, _ := json.Marshal(map[string]string{
			"ver": strconv.Itoa(ver),
			"err": lo.Ternary(rendererErr != nil, fmt.Sprintf("%v", rendererErr), ""),
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
		_ = renderer.Render(nil, decorender.EncodeFormatPNG, w, nil)
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
	defer watcher.Close()

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
