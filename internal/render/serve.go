package render

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ankurs/kb/assets"
	"github.com/ankurs/kb/internal/config"
	"github.com/ankurs/kb/internal/workspace"
)

func Handler(c config.Workspace) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			http.Error(w, "method not allowed", 405)
			return
		}
		if r.URL.Path == "/_kb/events" {
			events(w, r, c)
			return
		}
		for _, name := range []string{"live.css", "ui.css", "ui.js", "live.js"} {
			if r.URL.Path == "/_kb/"+name {
				w.Header().Set("X-Content-Type-Options", "nosniff")
				w.Header().Set("Content-Type", mimeType(name))
				io.WriteString(w, webFile(name))
				return
			}
		}
		if strings.HasPrefix(r.URL.Path, "/vendor/") {
			name := strings.TrimPrefix(r.URL.Path, "/")
			if strings.Contains(name, "..") {
				http.NotFound(w, r)
				return
			}
			file, err := fs.ReadFile(assets.FS, name)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", mimeType(name))
			w.Write(file)
			return
		}
		inv, err := workspace.Load(c)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			io.WriteString(w, LiveHTML(c, true))
			return
		}
		if r.URL.Path == "/_sidebar.md" {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			io.WriteString(w, workspace.Sidebar(inv.Nav()))
			return
		}
		if r.URL.Path == "/_kb/viewer-data.json" {
			b, err := viewerData(inv)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Write(b)
			return
		}
		key := strings.TrimPrefix(r.URL.Path, "/")
		if !config.SafeRelative(key) || strings.HasPrefix(key, ".") {
			http.NotFound(w, r)
			return
		}
		if page, ok := inv.Pages[key]; ok {
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			io.WriteString(w, page.Markdown)
			return
		}
		path, err := inv.Source(key)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			http.NotFound(w, r)
			return
		}
		// Other files can only be served when linked from an inventoried Markdown page.
		linked := false
		for _, page := range inv.Pages {
			for _, ref := range workspace.References(page.Markdown) {
				resolved, _, e := inv.Resolve(page.Key, ref.Target)
				if e != nil {
					continue
				}
				if resolved == key {
					linked = true
					break
				}
				if workspace.ExcalidrawPreviewKey(ref, resolved) == key {
					scene, e := inv.Source(resolved)
					if e == nil {
						if info, e := os.Stat(scene); e == nil && info.Mode().IsRegular() {
							if _, e := inv.ExcalidrawPreview(resolved); e == nil {
								linked = true
								break
							}
						}
					}
				}
			}
			if linked {
				break
			}
		}
		if !linked {
			http.NotFound(w, r)
			return
		}
		if err := checkActiveResource(c, path, key); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		if strings.EqualFold(filepath.Ext(path), ".svg") {
			b, err := os.ReadFile(path)
			if err != nil || safeSVG(b) != nil {
				http.Error(w, "unsafe SVG", http.StatusForbidden)
				return
			}
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeFile(w, r, path)
	})
}
func mimeType(name string) string {
	switch filepath.Ext(name) {
	case ".js":
		return "application/javascript"
	case ".css":
		return "text/css"
	case ".json":
		return "application/json"
	}
	return "application/octet-stream"
}
func digest(c config.Workspace) [32]byte {
	inv, err := workspace.Load(c)
	if err != nil {
		return sha256.Sum256([]byte(err.Error()))
	}
	h := sha256.New()
	for _, key := range inv.Keys {
		io.WriteString(h, key)
		io.WriteString(h, inv.Pages[key].Markdown)
	}
	var hash [32]byte
	copy(hash[:], h.Sum(nil))
	return hash
}
func events(w http.ResponseWriter, r *http.Request, c config.Workspace) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unavailable", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	last := digest(c)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			next := digest(c)
			if next != last {
				last = next
				fmt.Fprint(w, "data: changed\n\n")
				flusher.Flush()
			} else {
				fmt.Fprint(w, ": heartbeat\n\n")
				flusher.Flush()
			}
		}
	}
}
func Serve(c config.Workspace) error {
	return ServeWithLog(c, nil)
}

func ServeWithLog(c config.Workspace, logf func(string, ...any)) error {
	addr := fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
	fmt.Printf("kb serving %s at http://%s (Ctrl-C to stop)\n", c.Root, addr)
	handler := Handler(c)
	if logf != nil {
		next := handler
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logf("serve: %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}
	return http.ListenAndServe(addr, handler)
}
