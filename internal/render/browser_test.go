package render

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ankurs/kb/internal/config"
)

func headlessBrowser(t *testing.T) string {
	t.Helper()
	if name := os.Getenv("KB_BROWSER"); name != "" {
		return name
	}
	for _, name := range []string{"chromium", "chromium-browser", "google-chrome"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	if runtime.GOOS == "darwin" {
		for _, path := range []string{"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser"} {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}
	t.Skip("no headless Chromium available (set KB_BROWSER to run browser integration test)")
	return ""
}

func dumpDOM(t *testing.T, browser, url string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	args := []string{"--headless", "--disable-gpu", "--no-sandbox", "--disable-background-networking", "--disable-extensions", "--no-first-run", "--no-default-browser-check", "--user-data-dir=" + t.TempDir(), "--host-resolver-rules=MAP * ~NOTFOUND, EXCLUDE 127.0.0.1", "--virtual-time-budget=5000", "--dump-dom", url}
	cmd := exec.CommandContext(ctx, browser, args...)
	b, err := cmd.Output()
	if ctx.Err() != nil {
		t.Fatalf("browser timed out at %s", url)
	}
	if err != nil && !bytes.Contains(b, []byte("<!DOCTYPE html>")) {
		t.Fatalf("browser %s: %v (%d bytes)", url, err, len(b))
	}
	return string(b)
}

func TestBrowserRenderingOffline(t *testing.T) {
	browser := headlessBrowser(t)
	for _, trusted := range []bool{false, true} {
		name := "sanitized"
		if trusted {
			name = "trusted"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			md := "# Home\n\n<br><table><tr><td>benign</td></tr></table>\n\n" +
				`<img src="missing" onerror="document.body.setAttribute('data-executed','1')">` + "\n" +
				`<script id="attack">document.body.setAttribute('data-executed','1')</script>` + "\n\n" +
				"[bad](javascript:alert(1))\n\n![photo](pic.png)\n\n```js\nconst answer = 42;\n```\n\n```mermaid\ngraph LR\n A-->B\n```\n"
			if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(md), 0644); err != nil {
				t.Fatal(err)
			}
			png, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/lF0AAAAASUVORK5CYII=")
			os.WriteFile(filepath.Join(root, "pic.png"), png, 0644)
			c := config.Workspace{Root: root, Name: "Browser Test", Output: config.Output{Site: ".kb/site", Bundles: ".kb/snapshots"}, Render: config.Render{TrustHTML: trusted}}
			result, err := Build(c)
			if err != nil {
				t.Fatal(err)
			}
			bundle := filepath.Join(root, ".kb/snapshots/wiki.html")
			served := Handler(c)
			live := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/" {
					w.Header().Set("Content-Type", "text/html")
					w.Write([]byte(LiveHTML(c, false)))
					return
				}
				served.ServeHTTP(w, r)
			}))
			defer live.Close()
			static := httptest.NewServer(http.FileServer(http.Dir(result.Site)))
			defer static.Close()
			for _, viewer := range []struct{ name, url string }{
				{"bundle", "file://" + bundle}, {"serve", live.URL + "/#/README.md"}, {"site", static.URL + "/#/README.md"},
			} {
				t.Run(viewer.name, func(t *testing.T) {
					dom := dumpDOM(t, browser, viewer.url)
					if !strings.Contains(dom, "benign</td>") || !strings.Contains(dom, "<br>") || !strings.Contains(dom, "answer") || !strings.Contains(dom, "mermaid") || !strings.Contains(dom, "pic.png") && !strings.Contains(dom, "data:image/png") {
						t.Fatalf("missing benign render in %s (DOM %d bytes): %.1000s", viewer.name, len(dom), dom)
					}
					if !strings.Contains(dom, `data-processed="true"`) || !strings.Contains(dom, "<svg") {
						t.Fatal("Mermaid diagram did not render")
					}
					if viewer.name == "bundle" && !strings.Contains(dom, `<img src="data:image/png`) {
						t.Fatal("bundle image did not resolve to embedded data")
					}
					if trusted {
						if !strings.Contains(dom, `onerror="document.body.setAttribute`) {
							t.Fatal("trusted raw HTML removed")
						}
					} else if strings.Contains(dom, `onerror="document.body.setAttribute`) || strings.Contains(dom, `<script id="attack"`) || strings.Contains(dom, `data-executed="1"`) || strings.Contains(dom, `href="javascript:`) {
						t.Fatal("unsanitized executable Markdown in DOM")
					}
				})
			}
		})
	}
}
