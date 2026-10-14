package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestQMDProviderTrustAndArguments(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(t.TempDir(), "qmd")
	configDir := filepath.Join(dir, "custom-qmd")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "index.yaml")
	if err := os.WriteFile(configPath, []byte("collections:\n  wiki:\n    path: .\nmodels:\n  embed: hf:example/custom/model.gguf\n"), 0644); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\ncase \"$1\" in\n trust) echo \"" + configPath + " (trusted now)\";;\n query) printf 'config=%s args=%s\\n' \"$QMD_CONFIG_DIR\" \"$*\";;\n update|embed) echo \"$1\";;\nesac\n"
	if err := os.WriteFile(bin, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	var logs []string
	q := QMD{Root: dir, ConfigDir: "custom-qmd", Logf: func(format string, args ...any) { logs = append(logs, fmt.Sprintf(format, args...)) }}
	out, err := q.Search("test query", "wiki")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "query -c wiki test query") || !strings.Contains(out, filepath.Join(dir, "custom-qmd")) {
		t.Fatal(out)
	}
	if err := q.Index(true); err != nil {
		t.Fatal(err)
	}
	diagnostics := strings.Join(logs, "\n")
	for _, step := range []string{"checking configuration trust", "qmd: running query", "qmd: running update", "qmd: running embed"} {
		if !strings.Contains(diagnostics, step) {
			t.Fatalf("missing %q: %s", step, diagnostics)
		}
	}
	if strings.Contains(diagnostics, "test query") {
		t.Fatalf("query leaked into verbose diagnostics: %s", diagnostics)
	}
}

func TestDoctorWithoutProjectQMDConfig(t *testing.T) {
	root := t.TempDir()
	issues := Doctor(root, ".qmd")
	if len(issues) != 1 || !strings.Contains(issues[0], "qmd init") || strings.Contains(issues[0], "trust") {
		t.Fatalf("issues=%q", issues)
	}
	q := QMD{Root: root, ConfigDir: ".qmd"}
	if _, err := q.Search("question", ""); err == nil || !strings.Contains(err.Error(), "qmd init") {
		t.Fatalf("search error: %v", err)
	}
	if err := q.Index(false); err == nil || !strings.Contains(err.Error(), "qmd init") {
		t.Fatalf("index error: %v", err)
	}
}
func TestDefaultProjectConfigNeedsNoTrust(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".qmd")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "collections:\n  wiki:\n    path: .\nmodels:\n  embed: hf:ggml-org/embeddinggemma-300M-GGUF/embeddinggemma-300M-Q8_0.gguf\n  generate: hf:tobil/qmd-query-expansion-1.7B-gguf/qmd-query-expansion-1.7B-q4_k_m.gguf\n  rerank: hf:ggml-org/Qwen3-Reranker-0.6B-Q8_0-GGUF/qwen3-reranker-0.6b-q8_0.gguf\n"
	if err := os.WriteFile(filepath.Join(dir, "index.yml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "qmd")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nif [ \"$1\" = trust ]; then exit 42; fi\necho result\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	if issues := Doctor(root, ".qmd"); len(issues) != 0 {
		t.Fatalf("issues=%q", issues)
	}
	if out, err := (QMD{Root: root, ConfigDir: ".qmd"}).Search("question", ""); err != nil || !strings.Contains(out, "result") {
		t.Fatalf("search: %q %v", out, err)
	}
}

func TestQMDTrustGateSettings(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".qmd")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "index.yml")
	q := QMD{Root: root, ConfigDir: ".qmd"}
	cases := []struct {
		name  string
		yaml  string
		gated bool
	}{
		{"local collection", "collections:\n  notes:\n    path: .\n", false},
		{"external collection", "collections:\n  notes:\n    path: ../outside\n", true},
		{"update hook", "collections:\n  notes:\n    path: .\n    update: echo refresh\n", true},
		{"custom model", "models:\n  embed: hf:example/custom/model.gguf\n", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte(tc.yaml), 0644); err != nil {
				t.Fatal(err)
			}
			got, err := q.needsTrust(path)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.gated {
				t.Fatalf("needsTrust=%t, want %t", got, tc.gated)
			}
		})
	}
}

func TestQMDWithoutCollections(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".qmd")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.yml"), []byte("collections: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	q := QMD{Root: root, ConfigDir: ".qmd"}
	issues := Doctor(root, ".qmd")
	if len(issues) != 1 || !strings.Contains(issues[0], "no qmd collections") || !strings.Contains(issues[0], "qmd collection add") || strings.Contains(issues[0], "trust") {
		t.Fatalf("issues=%q", issues)
	}
	if _, err := q.Search("question", ""); err == nil || !strings.Contains(err.Error(), "no qmd collections") {
		t.Fatalf("search error: %v", err)
	}
	if err := q.Index(false); err == nil || !strings.Contains(err.Error(), "no qmd collections") {
		t.Fatalf("index error: %v", err)
	}
}
func TestQMDUnknownScopeAndEmptyResult(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".qmd")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.yml"), []byte("collections:\n  notes:\n    path: .\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "qmd")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	q := QMD{Root: root, ConfigDir: ".qmd"}
	if _, err := q.Search("question", "missing"); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("scope error: %v", err)
	}
	if _, err := q.Search("question", "notes"); err == nil || !strings.Contains(err.Error(), "no results") {
		t.Fatalf("empty output error: %v", err)
	}
}
