package render

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func readBundleData(t *testing.T, path string) BundleData {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	start := []byte(`<script type="application/json" id="page-data">`)
	part := bytes.SplitN(body, start, 2)
	if len(part) != 2 {
		t.Fatal("missing page data")
	}
	payload := bytes.SplitN(part[1], []byte(`</script>`), 2)[0]
	var data BundleData
	if err := json.Unmarshal(payload, &data); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestPublishHTMLOnly(t *testing.T) {
	c := testWorkspace(t)
	result, err := Publish(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Bundles) != 3 {
		t.Fatalf("bundles=%v", result.Bundles)
	}
	for _, name := range []string{"wiki.html", "borrowing-pilot.html", "handbook.html", publishMarker} {
		if _, err := os.Stat(filepath.Join(result.Dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	data := readBundleData(t, filepath.Join(result.Dir, "wiki.html"))
	if len(data.Pages) != 37 || len(data.Files) != 0 || len(data.Assets) == 0 {
		t.Fatalf("pages=%d files=%d assets=%d", len(data.Pages), len(data.Files), len(data.Assets))
	}
	if _, err := os.Stat(filepath.Join(result.Dir, "tool-library", "sources", "desk-rules.txt")); !os.IsNotExist(err) {
		t.Fatalf("unexpected raw source: %v", err)
	}
}

func TestPublishSelectionRenamingAndReplacement(t *testing.T) {
	c := testWorkspace(t)
	c.Publish.Dir = filepath.Join(t.TempDir(), "wiki-distribution")
	c.Publish.Wiki = "index.html"
	c.Publish.Include = []string{"tool-library"}
	c.Publish.Names = map[string]string{"tool-library": "lending.html"}
	result, err := Publish(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Bundles) != 2 {
		t.Fatalf("bundles=%v", result.Bundles)
	}
	data := readBundleData(t, filepath.Join(result.Dir, "index.html"))
	if _, ok := data.Pages["README.md"]; ok {
		t.Fatal("root page included despite explicit selection")
	}
	if _, ok := data.Pages["handbook/README.md"]; ok {
		t.Fatal("excluded folder in wiki")
	}
	if _, ok := data.Pages["tool-library/README.md"]; !ok {
		t.Fatal("selected folder missing")
	}
	if _, err := os.Stat(filepath.Join(result.Dir, "lending.html")); err != nil {
		t.Fatal(err)
	}
	c.Publish.Include = []string{"handbook"}
	if _, err := Publish(c); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(result.Dir, "lending.html")); !os.IsNotExist(err) {
		t.Fatalf("stale bundle retained: %v", err)
	}
	if _, err := os.Stat(filepath.Join(result.Dir, "handbook.html")); err != nil {
		t.Fatal(err)
	}
}

func TestPublishRefusesUnownedDirectory(t *testing.T) {
	c := testWorkspace(t)
	c.Publish.Dir = filepath.Join(t.TempDir(), "existing")
	if err := os.MkdirAll(c.Publish.Dir, 0755); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(c.Publish.Dir, "keep.txt")
	if err := os.WriteFile(keep, []byte("do not delete"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Publish(c); err == nil {
		t.Fatal("replaced directory without publish marker")
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("user file removed: %v", err)
	}
}

func TestPublishRootPagesAndFilenameCollision(t *testing.T) {
	c := testWorkspace(t)
	c.Publish.Include = []string{"tool-library"}
	c.Publish.Root = true
	c.Publish.Names = map[string]string{"tool-library": "wiki.html"}
	if _, err := Publish(c); err == nil {
		t.Fatal("accepted duplicate wiki filename")
	}
	if _, err := os.Stat(filepath.Join(c.Root, c.Publish.Dir)); !os.IsNotExist(err) {
		t.Fatalf("published despite filename collision: %v", err)
	}
	c.Publish.Names = map[string]string{"tool-library": "lending.html"}
	result, err := Publish(c)
	if err != nil {
		t.Fatal(err)
	}
	data := readBundleData(t, filepath.Join(result.Dir, "wiki.html"))
	if _, ok := data.Pages["README.md"]; !ok {
		t.Fatal("root pages omitted when publish.root is enabled")
	}
}
