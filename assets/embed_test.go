package assets

import (
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"testing"
)

func TestEmbeddedBrowserAssets(t *testing.T) {
	// Go module archives omit nested files under directories named vendor.
	// A source checkout can otherwise build a working binary while go install
	// of the tagged module silently embeds only the top-level manifest.
	if _, err := os.Stat("vendor"); !os.IsNotExist(err) {
		t.Fatalf("assets/vendor must not be used for embedded assets: %v", err)
	}
	manifest, err := fs.ReadFile(FS, "browser/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var packages map[string]struct {
		Files []string `json:"files"`
	}
	if err := json.Unmarshal(manifest, &packages); err != nil {
		t.Fatal(err)
	}
	if len(packages) == 0 {
		t.Fatal("empty browser asset manifest")
	}
	for name, pkg := range packages {
		if len(pkg.Files) == 0 {
			t.Errorf("%s has no assets", name)
		}
		for _, file := range pkg.Files {
			b, err := fs.ReadFile(FS, path.Join("browser", file))
			if err != nil || len(b) == 0 {
				t.Errorf("%s: missing embedded asset: %v", file, err)
			}
		}
	}
	licenses, err := fs.ReadDir(FS, "browser/licenses")
	if err != nil || len(licenses) == 0 {
		t.Fatalf("missing browser asset licenses: %v", err)
	}
}
