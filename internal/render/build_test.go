package render

import (
	"bytes"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ankurs/kb/internal/config"
	"github.com/ankurs/kb/internal/workspace"
)

func testWorkspace(t *testing.T) config.Workspace {
	t.Helper()
	src := filepath.Join("..", "..", "testdata", "sample")
	dir := t.TempDir()
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		rel, _ := filepath.Rel(src, path)
		if rel == ".kb" || rel == "dist" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if rel == filepath.Join(".qmd", "index.sqlite") || strings.HasPrefix(rel, filepath.Join(".qmd", "index.sqlite")+"-") {
			return nil
		}
		to := filepath.Join(dir, rel)
		if d.IsDir() {
			return os.MkdirAll(to, 0755)
		}
		b, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		return os.WriteFile(to, b, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(dir, filepath.Join(dir, "absent"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func TestBuildVerifyAndStaleness(t *testing.T) {
	c := testWorkspace(t)
	r, err := Build(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Bundles) != 3 {
		t.Fatalf("bundles: %v", r.Bundles)
	}
	if err := Verify(c); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(c.Root, "tool-library", "kb", "borrowing-policy.md")
	f, e := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0644)
	if e != nil {
		t.Fatal(e)
	}
	f.WriteString("\nThe review schedule changed.\n")
	f.Close()
	if err := Verify(c); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("expected stale output: %v", err)
	}
	if _, err := Build(c); err != nil {
		t.Fatal(err)
	}
	if err := Verify(c); err != nil {
		t.Fatal(err)
	}
}
func TestSnapshotFallback(t *testing.T) {
	c := testWorkspace(t)
	if _, err := Build(c); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(c.Root, "reference")); err != nil {
		t.Fatal(err)
	}
	r, err := Build(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Warnings) != 1 {
		t.Fatal(r.Warnings)
	}
	wiki, _ := os.ReadFile(filepath.Join(c.Root, c.Output.Bundles, "wiki.html"))
	if !bytes.Contains(wiki, []byte("handbook/check-in/README.md")) {
		t.Fatal("fallback content omitted")
	}
}
func TestVirtualSidebarAndAssets(t *testing.T) {
	c := testWorkspace(t)
	handler := Handler(c)
	for path, want := range map[string]string{"/_sidebar.md": "Tool Library Pilot", "/README.md": "Neighborhood Tool Library", "/vendor/docsify/docsify-4.13.1.min.js": "docsify", "/handbook/check-in/check-in-flow.svg": "<svg", "/not-linked.pdf": ""} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest("GET", path, nil))
		b := recorder.Body.Bytes()
		if want == "" {
			if recorder.Code != 404 {
				t.Fatalf("%s: %d", path, recorder.Code)
			}
		} else if recorder.Code != 200 || !bytes.Contains(bytes.ToLower(b), bytes.ToLower([]byte(want))) {
			t.Fatalf("%s: %d", path, recorder.Code)
		}
	}
	_ = workspace.Slug("Current understanding")
}
func TestOfflineRuntimeResources(t *testing.T) {
	c := testWorkspace(t)
	result, err := Build(c)
	if err != nil {
		t.Fatal(err)
	}
	html, err := os.ReadFile(filepath.Join(result.Site, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(html, []byte(`src="https://`)) || bytes.Contains(html, []byte(`href="https://`)) {
		t.Fatal("static viewer requires external runtime")
	}
	bundle, err := os.ReadFile(filepath.Join(c.Root, c.Output.Bundles, "wiki.html"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(bundle, []byte(`src="https://`)) || bytes.Contains(bundle, []byte(`href="https://`)) {
		t.Fatal("bundle requires external runtime")
	}
}

func TestMountedAssetInSiteAndBundle(t *testing.T) {
	c := testWorkspace(t)
	result, err := Build(c)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(result.Site, "handbook", "check-in", "check-in-flow.svg")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("site asset: %v", err)
	}
	wiki, err := os.ReadFile(filepath.Join(c.Root, c.Output.Bundles, "wiki.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(wiki, []byte(`"mounts":["handbook"]`)) || !bytes.Contains(wiki, []byte(`handbook/check-in/check-in-flow.svg`)) {
		t.Fatal("mount or embedded asset missing from bundle")
	}
}

func TestLiveSidebarConfiguration(t *testing.T) {
	c := testWorkspace(t)
	handler := Handler(c)
	html := httptest.NewRecorder()
	handler.ServeHTTP(html, httptest.NewRequest("GET", "/", nil))
	for _, part := range []string{"loadSidebar:true", "sidebarDisplayLevel:1", "'/.*/_sidebar.md':'/_sidebar.md'", "plugins:[function(hook,vm){hook.beforeEach(function(md){"} {
		if !bytes.Contains(html.Body.Bytes(), []byte(part)) {
			t.Fatalf("missing %q in served viewer", part)
		}
	}
	if bytes.Contains(html.Body.Bytes(), []byte("hook.beforeEach(function(md,vm)")) {
		t.Fatal("Docsify treats two-argument beforeEach hooks as async; rendering would stall")
	}
	sidebar := httptest.NewRecorder()
	handler.ServeHTTP(sidebar, httptest.NewRequest("GET", "/_sidebar.md", nil))
	if sidebar.Code != 200 || sidebar.Header().Get("Cache-Control") != "no-store" || !bytes.Contains(sidebar.Body.Bytes(), []byte("/tool-library/README.md")) {
		t.Fatalf("sidebar status=%d bytes=%d", sidebar.Code, sidebar.Body.Len())
	}
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest("GET", "/tool-library/kb/borrowing-policy.md", nil))
	if page.Code != 200 {
		t.Fatalf("nested page status=%d", page.Code)
	}
}

func TestLiveThemeParity(t *testing.T) {
	c := testWorkspace(t)
	h := Handler(c)
	response := httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest("GET", "/_kb/live.css", nil))
	if response.Code != 200 || !strings.Contains(response.Header().Get("Content-Type"), "text/css") {
		t.Fatalf("theme status=%d content-type=%q", response.Code, response.Header().Get("Content-Type"))
	}
	css := response.Body.String()
	for _, rule := range []string{"--gh-canvas: #0d1117", "--gh-code-bg: #161b22", "--gh-text: #e6edf3", "--gh-heading: #f0f6fc", ".sidebar-nav > ul > li.folder", ".sidebar-nav ul ul > li.folder", ".markdown-section .token.keyword", ".markdown-section .token.string", ".markdown-section .mermaid"} {
		if !strings.Contains(css, rule) {
			t.Fatalf("missing theme rule %q", rule)
		}
	}
	result, err := Build(c)
	if err != nil {
		t.Fatal(err)
	}
	built, err := os.ReadFile(filepath.Join(result.Site, "_kb", "live.css"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(built, response.Body.Bytes()) {
		t.Fatal("built and served themes differ")
	}
}
