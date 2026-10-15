package render

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestPublishMissingMountFallbackSelection(t *testing.T) {
	c := testWorkspace(t)
	if _, err := Build(c); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(c.Root, "reference")); err != nil {
		t.Fatal(err)
	}
	for _, include := range [][]string{nil, {"handbook"}} {
		c.Publish.Include = include
		result, err := Publish(c)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Warnings) != 1 {
			t.Fatalf("fallback warning: %v", result.Warnings)
		}
		for _, name := range []string{"wiki.html", "handbook.html"} {
			data := readBundleData(t, filepath.Join(result.Dir, name))
			if _, ok := data.Pages["handbook/check-in/README.md"]; !ok {
				t.Fatalf("%s omitted fallback", name)
			}
		}
	}
}

func TestFilteredPublishKeepsCrossFolderImages(t *testing.T) {
	c := testWorkspace(t)
	image := filepath.Join(c.Root, "shared.png")
	if err := os.WriteFile(image, []byte("image bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(c.Root, "tool-library", "README.md")
	f, err := os.OpenFile(page, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("\n![Shared](../shared.png)\n")
	f.Close()
	c.Publish.Include = []string{"tool-library"}
	r, err := Publish(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"wiki.html", "borrowing-pilot.html"} {
		data := readBundleData(t, filepath.Join(r.Dir, name))
		if data.Assets["shared.png"] == "" {
			t.Fatalf("%s lost cross-folder image", name)
		}
	}
}

func TestPublishValidatesSelectedResourcesOnly(t *testing.T) {
	c := testWorkspace(t)
	page := filepath.Join(c.Root, "reference", "README.md")
	f, err := os.OpenFile(page, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("\n![Unsafe](bad.svg)\n[Active](bad.html)\n")
	f.Close()
	os.WriteFile(filepath.Join(c.Root, "reference", "bad.svg"), []byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"/>`), 0644)
	os.WriteFile(filepath.Join(c.Root, "reference", "bad.html"), []byte(`<script>alert(1)</script>`), 0644)
	c.Publish.Include = []string{"tool-library"}
	if _, err := Publish(c); err != nil {
		t.Fatalf("unselected resources blocked publish: %v", err)
	}
	c.Publish.Include = nil
	c.Publish.Exclude = []string{"handbook"}
	if _, err := Publish(c); err != nil {
		t.Fatalf("excluded resources blocked publish: %v", err)
	}
	c.Publish.Exclude = nil
	if _, err := Publish(c); err == nil {
		t.Fatal("default publish ignored unsafe selected resources")
	}
	c.Publish.Include = []string{"handbook"}
	if _, err := Publish(c); err == nil {
		t.Fatal("selected mount ignored unsafe resources")
	}
}

func TestPublishSkipsUnavailableExcludedMount(t *testing.T) {
	for _, fallback := range []string{"missing", "invalid"} {
		t.Run(fallback, func(t *testing.T) {
			c := testWorkspace(t)
			if fallback == "invalid" {
				path := filepath.Join(c.Root, ".kb", "snapshots", "handbook.html")
				os.MkdirAll(filepath.Dir(path), 0755)
				os.WriteFile(path, []byte("not a snapshot"), 0644)
			}
			if err := os.RemoveAll(filepath.Join(c.Root, "reference")); err != nil {
				t.Fatal(err)
			}
			c.Publish.Include = []string{"tool-library"}
			if _, err := Publish(c); err != nil {
				t.Fatalf("include selected folder: %v", err)
			}
			c.Publish.Include = nil
			c.Publish.Exclude = []string{"handbook"}
			if _, err := Publish(c); err != nil {
				t.Fatalf("excluded unavailable mount: %v", err)
			}
			c.Publish.Exclude = nil
			if _, err := Publish(c); err == nil {
				t.Fatal("default publish omitted unavailable mount")
			}
			c.Publish.Include = []string{"handbook"}
			if _, err := Publish(c); err == nil {
				t.Fatal("selected unavailable mount did not error")
			}
		})
	}
}

func TestFallbackKeepsWorkspaceRelativeAssets(t *testing.T) {
	c := testWorkspace(t)
	image := filepath.Join(c.Root, "shared.png")
	os.WriteFile(image, []byte("shared image"), 0644)
	rootPage := filepath.Join(c.Root, "README.md")
	root, err := os.OpenFile(rootPage, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	root.WriteString("\n![Current shared image](shared.png)\n")
	root.Close()
	page := filepath.Join(c.Root, "reference", "README.md")
	f, err := os.OpenFile(page, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("\n![Shared](../shared.png)\n")
	f.Close()
	if _, err := Build(c); err != nil {
		t.Fatal(err)
	}
	focused := readBundleData(t, filepath.Join(c.Root, ".kb", "snapshots", "handbook.html"))
	if focused.Assets["shared.png"] == "" {
		t.Fatal("focused snapshot lost shared image before fallback")
	}
	os.WriteFile(image, []byte("new shared image"), 0644)
	if err := os.RemoveAll(filepath.Join(c.Root, "reference")); err != nil {
		t.Fatal(err)
	}
	for _, include := range [][]string{nil, {"handbook"}} {
		c.Publish.Include = include
		r, err := Publish(c)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"wiki.html", "handbook.html"} {
			data := readBundleData(t, filepath.Join(r.Dir, name))
			if data.Assets["shared.png"] == "" || data.Assets["handbook/shared.png"] != "" {
				t.Fatalf("%s remapped shared image: %+v", name, data.Assets)
			}
			if include == nil && !strings.Contains(data.Assets["shared.png"], base64.StdEncoding.EncodeToString([]byte("new shared image"))) {
				t.Fatalf("%s replaced the live shared image with the fallback copy", name)
			}
		}
	}
}

func TestLegacyMountRelativeFallback(t *testing.T) {
	c := testWorkspace(t)
	path := filepath.Join(c.Root, ".kb", "snapshots", "handbook.html")
	os.MkdirAll(filepath.Dir(path), 0755)
	data := BundleData{Pages: map[string]string{"README.md": "# Handbook\n\n![Icon](icon.png)\n"}, Assets: map[string]string{"icon.png": "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("image"))}}
	payload, _ := json.Marshal(data)
	contents := append([]byte(`<script type="application/json" id="page-data">`), payload...)
	contents = append(contents, []byte(`</script>`)...)
	os.WriteFile(path, contents, 0644)
	os.RemoveAll(filepath.Join(c.Root, "reference"))
	c.Publish.Include = []string{"handbook"}
	r, err := Publish(c)
	if err != nil {
		t.Fatal(err)
	}
	wiki := readBundleData(t, filepath.Join(r.Dir, "wiki.html"))
	if wiki.Assets["handbook/icon.png"] == "" || !strings.Contains(wiki.Pages["handbook/README.md"], "Icon") {
		t.Fatalf("legacy fallback not normalized: %+v", wiki.Assets)
	}
}
