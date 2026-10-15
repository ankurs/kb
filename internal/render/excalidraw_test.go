package render

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ankurs/kb/internal/config"
	"github.com/ankurs/kb/internal/workspace"
)

const exampleScene = `{"type":"excalidraw","version":2,"elements":[],"appState":{},"files":{}}`

var smallPNG = func() string {
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(b.Bytes())
}()

func excalidrawFixture(t *testing.T) config.Workspace {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "diagrams"), 0755); err != nil {
		t.Fatal(err)
	}
	md := "# Home\n\n![Process](diagrams/process.excalidraw)\n\n[Edit scene](diagrams/process.excalidraw)\n"
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(md), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "diagrams", "process.excalidraw"), []byte(exampleScene), 0644); err != nil {
		t.Fatal(err)
	}
	png, err := base64.StdEncoding.DecodeString(smallPNG)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "diagrams", "process.excalidraw.png"), png, 0644); err != nil {
		t.Fatal(err)
	}
	return config.Workspace{Root: root, Name: "Drawing", Output: config.Output{Site: ".kb/site", Bundles: ".kb/snapshots"}, Publish: config.Publish{Dir: "dist", Wiki: "wiki.html"}, Capture: config.Capture{Inbox: "inbox.md", Templates: ".templates"}, Search: config.Search{QMDConfig: ".qmd"}, Server: config.Server{Port: 3000}}
}

func TestExcalidrawSidecarInViewers(t *testing.T) {
	c := excalidrawFixture(t)
	inv, err := workspace.Load(c)
	if err != nil {
		t.Fatal(err)
	}
	if issues := inv.Check(); len(issues) != 0 {
		t.Fatalf("valid scene and preview: %+v", issues)
	}
	build, err := Build(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify(c); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"process.excalidraw", "process.excalidraw.png"} {
		if _, err := os.Stat(filepath.Join(build.Site, "diagrams", name)); err != nil {
			t.Fatalf("static site missing %s: %v", name, err)
		}
	}
	bundle := readBundleData(t, filepath.Join(c.Root, ".kb", "snapshots", "wiki.html"))
	if bundle.Assets["diagrams/process.excalidraw"] != "data:image/png;base64,"+smallPNG {
		t.Fatal("snapshot did not embed the scene's PNG preview")
	}
	if !strings.HasSuffix(bundle.Files["diagrams/process.excalidraw"], "diagrams/process.excalidraw") {
		t.Fatal("offline snapshot did not preserve the editable scene link")
	}
	published, err := Publish(c)
	if err != nil {
		t.Fatal(err)
	}
	public := readBundleData(t, filepath.Join(published.Dir, "wiki.html"))
	if public.Assets["diagrams/process.excalidraw"] != bundle.Assets["diagrams/process.excalidraw"] || len(public.Files) != 0 {
		t.Fatal("published HTML lost preview or included raw sources")
	}
	handler := Handler(c)
	for _, route := range []string{"/diagrams/process.excalidraw", "/diagrams/process.excalidraw.png"} {
		r := httptest.NewRecorder()
		handler.ServeHTTP(r, httptest.NewRequest(http.MethodGet, route, nil))
		if r.Code != http.StatusOK {
			t.Fatalf("%s status=%d", route, r.Code)
		}
	}
	source, err := os.ReadFile(filepath.Join(c.Root, ".kb", "snapshots", "wiki.html"))
	if err != nil || !bytes.Contains(source, []byte("diagrams/process.excalidraw")) {
		t.Fatalf("source link disappeared from bundle: %v", err)
	}
}

func TestNestedExcalidrawImageLabelBuild(t *testing.T) {
	c := excalidrawFixture(t)
	md := "# Home\n\n![Process [draft]](diagrams/process.excalidraw)\n"
	if err := os.WriteFile(filepath.Join(c.Root, "README.md"), []byte(md), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(c); err != nil {
		t.Fatal(err)
	}
	data := readBundleData(t, filepath.Join(c.Root, ".kb", "snapshots", "wiki.html"))
	if data.Assets["diagrams/process.excalidraw"] != "data:image/png;base64,"+smallPNG {
		t.Fatal("nested image label did not embed the preview")
	}
	if err := os.Remove(filepath.Join(c.Root, "diagrams", "process.excalidraw.png")); err != nil {
		t.Fatal(err)
	}
	inv, err := workspace.Load(c)
	if err != nil {
		t.Fatal(err)
	}
	if issues := inv.Check(); len(issues) == 0 {
		t.Fatal("check accepted a nested-label image without its preview")
	}
	if _, err := Build(c); err == nil {
		t.Fatal("build accepted a nested-label image without its preview")
	}
}

func TestExcalidrawMissingInvalidAndEscapingPreview(t *testing.T) {
	for _, condition := range []string{"missing", "invalid", "truncated", "corrupt", "oversized", "too-many-pixels", "escaping"} {
		t.Run(condition, func(t *testing.T) {
			c := excalidrawFixture(t)
			preview := filepath.Join(c.Root, "diagrams", "process.excalidraw.png")
			if err := os.Remove(preview); err != nil {
				t.Fatal(err)
			}
			switch condition {
			case "invalid":
				if err := os.WriteFile(preview, []byte("not a PNG"), 0644); err != nil {
					t.Fatal(err)
				}
			case "truncated", "corrupt", "too-many-pixels":
				b, _ := base64.StdEncoding.DecodeString(smallPNG)
				if condition == "truncated" {
					b = b[:33] // Complete IHDR, no pixel data. DecodeConfig accepts it.
				} else if condition == "corrupt" {
					b[45] ^= 0xff // Corrupt IDAT, keeping the valid IHDR intact.
				} else {
					binary.BigEndian.PutUint32(b[16:20], 8192)
					binary.BigEndian.PutUint32(b[20:24], 8192)
					binary.BigEndian.PutUint32(b[29:33], crc32.ChecksumIEEE(b[12:29]))
				}
				if condition == "truncated" || condition == "corrupt" || condition == "too-many-pixels" {
					if _, err := png.DecodeConfig(bytes.NewReader(b)); err != nil {
						t.Fatalf("test preview must have valid IHDR: %v", err)
					}
				}
				if err := os.WriteFile(preview, b, 0644); err != nil {
					t.Fatal(err)
				}
			case "oversized":
				if err := os.WriteFile(preview, nil, 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.Truncate(preview, 20<<20+1); err != nil {
					t.Fatal(err)
				}
			case "escaping":
				outside := filepath.Join(t.TempDir(), "secret.png")
				png, _ := base64.StdEncoding.DecodeString(smallPNG)
				if err := os.WriteFile(outside, png, 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, preview); err != nil {
					t.Fatal(err)
				}
			}
			c.Check.WarnMissing = []string{"README.md"}
			inv, err := workspace.Load(c)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, issue := range inv.Check() {
				if strings.Contains(issue.Message, "process.excalidraw") && !issue.Warning {
					found = true
				}
			}
			if !found {
				t.Fatal("check did not reject a missing or unsafe preview")
			}
			if _, err := Build(c); err == nil {
				t.Fatal("build accepted an unusable Excalidraw preview")
			}
			if _, err := Publish(c); err == nil {
				t.Fatal("publish accepted an unusable Excalidraw preview")
			}
			r := httptest.NewRecorder()
			Handler(c).ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/diagrams/process.excalidraw.png", nil))
			if r.Code == http.StatusOK {
				t.Fatal("served an unusable Excalidraw preview")
			}
		})
	}
}

func TestExcalidrawSourceLinkNeedsNoPreview(t *testing.T) {
	c := excalidrawFixture(t)
	if err := os.Remove(filepath.Join(c.Root, "diagrams", "process.excalidraw.png")); err != nil {
		t.Fatal(err)
	}
	md := "# Home\n\n[Edit scene](diagrams/process.excalidraw)\n"
	if err := os.WriteFile(filepath.Join(c.Root, "README.md"), []byte(md), 0644); err != nil {
		t.Fatal(err)
	}
	inv, err := workspace.Load(c)
	if err != nil {
		t.Fatal(err)
	}
	if issues := inv.Check(); len(issues) != 0 {
		t.Fatalf("ordinary scene link should work without a preview: %+v", issues)
	}
	if _, err := Build(c); err != nil {
		t.Fatal(err)
	}
}

func TestExcalidrawPreviewStaleness(t *testing.T) {
	c := excalidrawFixture(t)
	if _, err := Build(c); err != nil {
		t.Fatal(err)
	}
	preview := filepath.Join(c.Root, "diagrams", "process.excalidraw.png")
	// A different valid PNG must invalidate both the snapshot and copied asset.
	f, err := os.Create(preview)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := Verify(c); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("changed preview did not stale built output: %v", err)
	}
	if _, err := Build(c); err != nil {
		t.Fatal(err)
	}
	if err := Verify(c); err != nil {
		t.Fatal(err)
	}
}

func TestMountedExcalidrawPreviewFallback(t *testing.T) {
	c := testWorkspace(t)
	dir := filepath.Join(c.Root, "reference")
	if err := os.WriteFile(filepath.Join(dir, "scene.excalidraw"), []byte(exampleScene), 0644); err != nil {
		t.Fatal(err)
	}
	pngData, _ := base64.StdEncoding.DecodeString(smallPNG)
	if err := os.WriteFile(filepath.Join(dir, "scene.excalidraw.png"), pngData, 0644); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(filepath.Join(dir, "README.md"), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("\n![Scene](scene.excalidraw)\n"); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(c); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	c.Publish.Include = []string{"handbook"}
	result, err := Publish(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"wiki.html", "handbook.html"} {
		data := readBundleData(t, filepath.Join(result.Dir, name))
		if data.Assets["handbook/scene.excalidraw"] != "data:image/png;base64,"+smallPNG {
			t.Fatalf("%s lost mounted scene's fallback image", name)
		}
	}
}
