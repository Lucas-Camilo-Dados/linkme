package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ironicbadger/linkme/internal/config"
	"github.com/ironicbadger/linkme/internal/generator"
)

const (
	defaultConfigPath = "config/config.yml"
	defaultThemesDir  = "themes"
	defaultOutputDir  = "dist"
	defaultPort       = "3000"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "build":
		if err := build(); err != nil {
			log.Fatalf("Build failed: %v", err)
		}
		fmt.Println("Build complete! Output in ./dist")
	case "watch":
		if err := watch(); err != nil {
			log.Fatalf("Watch failed: %v", err)
		}
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: linkme <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  build    Build the static site")
	fmt.Println("  watch    Build and serve with live reload")
}

func build() error {
	cfg, err := config.Load(defaultConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	gen, err := generator.New(cfg, defaultThemesDir, defaultOutputDir)
	if err != nil {
		return fmt.Errorf("failed to initialize generator: %w", err)
	}

	if err := gen.Generate(); err != nil {
		return fmt.Errorf("failed to generate site: %w", err)
	}

	return nil
}

func watch() error {
	// Initial build
	if err := build(); err != nil {
		return err
	}

	// Set up file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	// Watch config directory
	if err := watcher.Add("config"); err != nil {
		log.Printf("Warning: could not watch config directory: %v", err)
	}

	// Watch themes directory recursively
	if err := watchDir(watcher, defaultThemesDir); err != nil {
		log.Printf("Warning: could not watch themes directory: %v", err)
	}

	// Watch assets directory
	if err := watcher.Add("assets"); err != nil {
		log.Printf("Warning: could not watch assets directory: %v", err)
	}

	// Channel to signal rebuilds
	rebuild := make(chan struct{}, 1)

	// Watch for file changes
	go func() {
		var debounce <-chan time.Time
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
					// Debounce rapid changes
					debounce = time.After(100 * time.Millisecond)
				}
			case <-debounce:
				select {
				case rebuild <- struct{}{}:
				default:
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Watcher error: %v", err)
			}
		}
	}()

	// Rebuild on changes
	go func() {
		for range rebuild {
			fmt.Println("\n--- Rebuilding... ---")
			if err := build(); err != nil {
				log.Printf("Build error: %v", err)
			} else {
				fmt.Println("--- Rebuild complete ---")
			}
		}
	}()

	// Serve the output directory
	port := defaultPort
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	fmt.Printf("\nServing at http://localhost:%s\n", port)
	fmt.Println("Watching for changes... (Ctrl+C to stop)")

	// Inject live reload script
	fs := http.FileServer(http.Dir(defaultOutputDir))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Serve index.html for root
		if r.URL.Path == "/" {
			injectLiveReload(w, r, filepath.Join(defaultOutputDir, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})

	// SSE endpoint for live reload
	http.HandleFunc("/__livereload", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE not supported", http.StatusInternalServerError)
			return
		}

		// Send initial connection message
		fmt.Fprintf(w, "data: connected\n\n")
		flusher.Flush()

		// Wait for rebuild signals
		for {
			select {
			case <-rebuild:
				fmt.Fprintf(w, "data: reload\n\n")
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	})

	return http.ListenAndServe(":"+port, nil)
}

func watchDir(watcher *fsnotify.Watcher, dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
}

func injectLiveReload(w http.ResponseWriter, r *http.Request, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Inject live reload script before </body>
	liveReloadScript := `<script>
(function() {
  const es = new EventSource('/__livereload');
  es.onmessage = function(e) {
    if (e.data === 'reload') {
      location.reload();
    }
  };
  es.onerror = function() {
    console.log('Live reload disconnected, retrying...');
    setTimeout(function() { location.reload(); }, 1000);
  };
})();
</script>`

	html := strings.Replace(string(data), "</body>", liveReloadScript+"</body>", 1)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
