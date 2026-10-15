package render

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
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

func TestVerifyWholeSite(t *testing.T) {
	c := testWorkspace(t)
	if _, err := Build(c); err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		name, file string
		edit       []byte
	}{
		{"linked CSV", filepath.Join(c.Root, "tool-library/sources/loans-2026-10-10.csv"), []byte("\nnew record\n")},
		{"viewer CSS", filepath.Join(c.Root, ".kb/site/_kb/live.css"), []byte("\n/* modified */\n")},
		{"copied asset", filepath.Join(c.Root, ".kb/site/handbook/check-in/check-in-flow.svg"), []byte("\n<!-- modified -->\n")},
	}
	for _, tc := range checks {
		t.Run(tc.name, func(t *testing.T) {
			f, err := os.OpenFile(tc.file, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				t.Fatal(err)
			}
			_, err = f.Write(tc.edit)
			f.Close()
			if err != nil {
				t.Fatal(err)
			}
			if err := Verify(c); err == nil || !strings.Contains(err.Error(), "stale") {
				t.Fatalf("expected stale output: %v", err)
			}
			if _, err := Build(c); err != nil {
				t.Fatal(err)
			}
			if err := Verify(c); err != nil {
				t.Fatal(err)
			}
		})
	}
	if err := os.WriteFile(filepath.Join(c.Root, ".kb/site/extra"), []byte("unexpected"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Verify(c); err == nil {
		t.Fatal("unexpected site file ignored")
	}
}

func TestLinkedSymlinksAndSnapshotDirectory(t *testing.T) {
	c := testWorkspace(t)
	page := filepath.Join(c.Root, "README.md")
	outside := t.TempDir()
	os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("private"), 0644)
	os.Symlink(outside, filepath.Join(c.Root, "escape"))
	f, _ := os.OpenFile(page, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("\n[secret](escape/secret.txt)\n")
	f.Close()
	inv, err := workspace.Load(c)
	if err != nil {
		t.Fatal(err)
	}
	issues := inv.Check()
	found := false
	for _, issue := range issues {
		if strings.Contains(issue.Message, "escapes content root") && !issue.Warning {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected hard containment error: %+v", issues)
	}
	response := httptest.NewRecorder()
	Handler(c).ServeHTTP(response, httptest.NewRequest("GET", "/escape/secret.txt", nil))
	if response.Code == 200 || bytes.Contains(response.Body.Bytes(), []byte("private")) {
		t.Fatal("served outside file")
	}
	if _, err := Build(c); err == nil {
		t.Fatal("built outside file")
	}
	// A link whose resolved target stays inside the workspace is valid.
	f, _ = os.OpenFile(page, os.O_RDWR, 0644)
	b, _ := os.ReadFile(page)
	f.Truncate(0)
	f.Write(bytes.ReplaceAll(b, []byte("escape/secret.txt"), []byte("inside/sources/desk-rules.txt")))
	f.Close()
	os.Symlink(filepath.Join(c.Root, "tool-library"), filepath.Join(c.Root, "inside"))
	if _, err := Build(c); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	Handler(c).ServeHTTP(response, httptest.NewRequest("GET", "/inside/sources/desk-rules.txt", nil))
	if response.Code != 200 {
		t.Fatalf("inside symlink status %d", response.Code)
	}
	os.RemoveAll(filepath.Join(c.Root, ".kb/snapshots"))
	os.Symlink(outside, filepath.Join(c.Root, ".kb/snapshots"))
	if _, err := Build(c); err == nil {
		t.Fatal("accepted symlinked snapshot directory")
	}
}

func TestUnsafeSVGRejected(t *testing.T) {
	c := testWorkspace(t)
	svg := filepath.Join(c.Root, "reference/check-in/check-in-flow.svg")
	os.WriteFile(svg, []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), 0644)
	if _, err := Build(c); err == nil {
		t.Fatal("embedded active SVG")
	}
	r := httptest.NewRecorder()
	Handler(c).ServeHTTP(r, httptest.NewRequest("GET", "/handbook/check-in/check-in-flow.svg", nil))
	if r.Code != 403 {
		t.Fatalf("unsafe SVG served: %d", r.Code)
	}
}

func TestFallbackSVGValidated(t *testing.T) {
	c := testWorkspace(t)
	if err := os.MkdirAll(filepath.Join(c.Root, ".kb/snapshots"), 0755); err != nil {
		t.Fatal(err)
	}
	payload := BundleData{Pages: map[string]string{"README.md": "# Old handbook"}, Assets: map[string]string{"icon.svg": "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"/>`))}}
	encoded, _ := json.Marshal(payload)
	snapshot := append([]byte(`<script type="application/json" id="page-data">`), encoded...)
	snapshot = append(snapshot, []byte(`</script>`)...)
	if err := os.WriteFile(filepath.Join(c.Root, ".kb/snapshots/handbook.html"), snapshot, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(c.Root, "reference")); err != nil {
		t.Fatal(err)
	}
	if _, err := Publish(c); err == nil {
		t.Fatal("accepted unsafe SVG from fallback snapshot")
	}
}

func TestActiveLinkedHTMLRequiresTrust(t *testing.T) {
	c := testWorkspace(t)
	page := filepath.Join(c.Root, "README.md")
	f, err := os.OpenFile(page, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("\n[Open](payload.html)\n")
	f.Close()
	active := filepath.Join(c.Root, "payload.html")
	if err := os.WriteFile(active, []byte(`<script>document.body.textContent='executed'</script>`), 0644); err != nil {
		t.Fatal(err)
	}
	c.Check.WarnMissing = []string{"README.md"}
	inv, err := workspace.Load(c)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, issue := range inv.Check() {
		if strings.Contains(issue.Message, "active linked file") && !issue.Warning {
			found = true
		}
	}
	if !found {
		t.Fatal("check did not report a hard active-file error")
	}
	if _, err := Build(c); err == nil || !strings.Contains(err.Error(), "active linked file") {
		t.Fatalf("build: %v", err)
	}
	if _, err := Publish(c); err == nil || !strings.Contains(err.Error(), "active linked file") {
		t.Fatalf("publish: %v", err)
	}
	r := httptest.NewRecorder()
	Handler(c).ServeHTTP(r, httptest.NewRequest("GET", "/payload.html", nil))
	if r.Code != 403 || strings.Contains(r.Body.String(), "executed") {
		t.Fatalf("serve: %d %s", r.Code, r.Body.String())
	}
	c.Render.TrustHTML = true
	inv, err = workspace.Load(c)
	if err != nil || len(inv.Check()) != 0 {
		t.Fatalf("trusted check: %v %+v", err, inv.Check())
	}
	result, err := Build(c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(result.Site, "payload.html")); err != nil {
		t.Fatal(err)
	}
	if _, err := Publish(c); err != nil {
		t.Fatal(err)
	}
	r = httptest.NewRecorder()
	Handler(c).ServeHTTP(r, httptest.NewRequest("GET", "/payload.html", nil))
	if r.Code != 200 || r.Header().Get("X-Content-Type-Options") != "nosniff" || !strings.Contains(r.Body.String(), "executed") {
		t.Fatalf("trusted serve: %d %v", r.Code, r.Header())
	}
}

func TestDisguisedHTMLBlocked(t *testing.T) {
	c := testWorkspace(t)
	f, err := os.OpenFile(filepath.Join(c.Root, "README.md"), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("\n[Raw](renamed.txt)\n")
	f.Close()
	os.WriteFile(filepath.Join(c.Root, "renamed.txt"), []byte(" \n<!doctype html><script>alert(1)</script>"), 0644)
	if _, err := Build(c); err == nil {
		t.Fatal("copied disguised HTML")
	}
	r := httptest.NewRecorder()
	Handler(c).ServeHTTP(r, httptest.NewRequest("GET", "/renamed.txt", nil))
	if r.Code != 403 {
		t.Fatalf("served disguised HTML: %d", r.Code)
	}
}
