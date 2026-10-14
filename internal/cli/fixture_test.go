package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ankurs/kb/internal/config"
	"github.com/ankurs/kb/internal/workspace"
	"gopkg.in/yaml.v3"
)

func TestThirtyDayFixture(t *testing.T) {
	root := copySample(t)
	c, err := config.Load(root, filepath.Join(root, "absent.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	inv, err := workspace.Load(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Keys) != 37 {
		t.Fatalf("expected 37 authored pages, got %d", len(inv.Keys))
	}
	if issues := inv.Check(); len(issues) > 0 {
		t.Fatalf("broken fixture links: %+v", issues)
	}
	journal, err := filepath.Glob(filepath.Join(root, "tool-library", "journal", "2026-*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(journal) != 5 {
		t.Fatalf("expected five weekly journal files, got %d", len(journal))
	}
	dateHeading := regexp.MustCompile(`(?m)^## (2026-\d\d-\d\d)$`)
	dates := map[string]bool{}
	for _, file := range journal {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range dateHeading.FindAllStringSubmatch(string(body), -1) {
			if dates[match[1]] {
				t.Fatalf("duplicate journal date %s", match[1])
			}
			dates[match[1]] = true
		}
	}
	start := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	for day := 0; day < 30; day++ {
		date := start.AddDate(0, 0, day).Format("2006-01-02")
		if !dates[date] {
			t.Fatalf("missing journal entry %s", date)
		}
	}
	if len(dates) != 30 {
		t.Fatalf("expected 30 dates, got %d", len(dates))
	}
	for _, file := range []string{"tool-library/kb/reminder-workflow.md", "reference/shifts/README.md"} {
		b, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "```mermaid") {
			t.Fatalf("missing Mermaid example in %s", file)
		}
	}
	for file, language := range map[string]string{"tool-library/kb/reminder-workflow.md": "yaml", "tool-library/kb/pilot-metrics.md": "json"} {
		body, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), "```"+language) {
			t.Fatalf("missing %s code block in %s", language, file)
		}
	}
	for _, file := range []string{"reference/check-in/check-in-flow.svg", "tool-library/sources/loans-2026-10-10.csv", "tool-library/sources/desk-rules.txt"} {
		if _, err := os.Stat(filepath.Join(root, file)); err != nil {
			t.Fatal(err)
		}
	}
	var search struct {
		Collections map[string]struct {
			Path   string `yaml:"path"`
			Update string `yaml:"update"`
		} `yaml:"collections"`
	}
	b, err := os.ReadFile(filepath.Join(root, ".qmd", "index.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(b, &search); err != nil {
		t.Fatal(err)
	}
	if len(search.Collections) != 2 || search.Collections["wiki"].Path != "." || search.Collections["handbook"].Path != "reference" {
		t.Fatalf("qmd collections: %+v", search.Collections)
	}
	for _, entry := range search.Collections {
		if entry.Update != "" {
			t.Fatal("fixture must not execute qmd hooks")
		}
	}
	var bench struct {
		Version    int    `json:"version"`
		Collection string `json:"collection"`
		Queries    []struct {
			ID string `json:"id"`
		} `json:"queries"`
	}
	b, err = os.ReadFile(filepath.Join(root, ".qmd", "bench-tool-library.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &bench); err != nil {
		t.Fatal(err)
	}
	if bench.Version != 1 || bench.Collection != "wiki" || len(bench.Queries) != 4 {
		t.Fatalf("benchmark fixture: %+v", bench)
	}
}

func TestSampleConfiguredSync(t *testing.T) {
	root := copySample(t)
	common := []string{"--workspace", root, "--global-config", filepath.Join(root, "absent.yaml")}
	runCLI(t, append(append([]string{}, common...), "publish")...)
	mirror := filepath.Join(root, ".kb", "mirror", "wiki.html")
	if _, err := os.Stat(mirror); !os.IsNotExist(err) {
		t.Fatalf("sync ran during publish: %v", err)
	}
	preview, _ := runCLI(t, append(append([]string{}, common...), "sync", "--show")...)
	if !strings.Contains(preview, ".kb/mirror") || !strings.Contains(preview, "KB_PUBLISH_DIR") {
		t.Fatal(preview)
	}
	if _, err := os.Stat(mirror); !os.IsNotExist(err) {
		t.Fatalf("sync --show executed: %v", err)
	}
	runCLI(t, append(append([]string{}, common...), "sync")...)
	b, err := os.ReadFile(mirror)
	if err != nil {
		t.Fatalf("local copy missing: %v", err)
	}
	if !strings.Contains(string(b), "Tool Library") {
		t.Fatal("copy lacks wiki content")
	}
}
