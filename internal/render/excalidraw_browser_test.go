package render

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserExcalidrawPreview(t *testing.T) {
	browser := headlessBrowser(t)
	c := excalidrawFixture(t)
	built, err := Build(c)
	if err != nil {
		t.Fatal(err)
	}
	probe := `<script>(function(){let count=0,timer=setInterval(function(){
  let article=document.querySelector('#article,.markdown-section');
  let image=article?.querySelector('img[alt="Process"]');
  if(!image||!image.complete){if(++count>200){clearInterval(timer);let o=document.createElement('output');o.id='excalidraw-probe';o.textContent='timeout';document.body.append(o)}return}
  clearInterval(timer);let result={url:image.getAttribute('src'),width:image.naturalWidth,
    source:!!Array.from(article.querySelectorAll('a')).find(a=>a.textContent.trim()==='Edit scene')};
  let out=document.createElement('output');out.id='excalidraw-probe';out.textContent=JSON.stringify(result);document.body.append(out)
},20)})();</script>`
	staticFiles := http.FileServer(http.Dir(built.Site))
	static := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(injectBrowserProbe(t, LiveHTML(c, false), probe)))
			return
		}
		staticFiles.ServeHTTP(w, r)
	}))
	defer static.Close()
	served := Handler(c)
	live := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(injectBrowserProbe(t, LiveHTML(c, false), probe)))
			return
		}
		served.ServeHTTP(w, r)
	}))
	defer live.Close()
	bundle, err := os.ReadFile(filepath.Join(c.Root, ".kb", "snapshots", "wiki.html"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "wiki.html")
	if err := os.WriteFile(path, []byte(injectBrowserProbe(t, string(bundle), probe)), 0644); err != nil {
		t.Fatal(err)
	}
	for _, viewer := range []struct{ name, url string }{
		{"bundle", "file://" + path + "#/README.md"},
		{"serve", live.URL + "/#/README.md"},
		{"site", static.URL + "/#/README.md"},
	} {
		t.Run(viewer.name, func(t *testing.T) {
			dom := dumpDOM(t, browser, viewer.url)
			var got struct {
				URL    string
				Width  int
				Source bool
			}
			browserProbeResult(t, dom, "excalidraw-probe", &got)
			if got.Width != 1 || !got.Source {
				t.Fatalf("preview or source link missing: %+v", got)
			}
			if viewer.name == "bundle" {
				if len(got.URL) < len("data:image/png;base64,") || got.URL[:len("data:image/png;base64,")] != "data:image/png;base64," {
					t.Fatalf("bundle preview was not embedded: %q", got.URL)
				}
			} else if got.URL != "diagrams/process.excalidraw.png" && got.URL != "/diagrams/process.excalidraw.png" {
				t.Fatalf("viewer did not use the sidecar preview: %q", got.URL)
			}
		})
	}
}
