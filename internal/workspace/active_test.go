package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestActiveLinkedFile(t *testing.T) {
	for _, tc := range []struct {
		name, content string
		active        bool
	}{
		{"page.html", "ordinary text", true},
		{"page.HTML", "ordinary text", true},
		{"renamed.txt", " \n<!doctype html><script>alert(1)</script>", true},
		{"script.js", "alert(1)", true},
		{"data.xml", `<?xml version="1.0"?><root/>`, true},
		{"notes.txt", "A plain text notice.\n", false},
		{"ledger.csv", "date,count\n2026-10-10,42\n", false},
		{"data.json", `{"answer":42}`, false},
		{"diagram.svg", `<svg xmlns="http://www.w3.org/2000/svg"/>`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tc.name)
			if err := os.WriteFile(path, []byte(tc.content), 0644); err != nil {
				t.Fatal(err)
			}
			active, err := ActiveLinkedFile(path)
			if err != nil || active != tc.active {
				t.Fatalf("active=%t, err=%v", active, err)
			}
		})
	}
}
