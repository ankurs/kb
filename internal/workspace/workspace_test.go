package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ankurs/kb/internal/config"
)

func fixture(t *testing.T) (config.Workspace, string) {
	t.Helper()
	dir := t.TempDir()
	err := filepath.WalkDir(filepath.Join("..", "..", "testdata", "sample"), func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		rel, _ := filepath.Rel(filepath.Join("..", "..", "testdata", "sample"), path)
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
	return c, dir
}
func TestInventoryAndNavigation(t *testing.T) {
	c, _ := fixture(t)
	inv, err := Load(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Keys) != 37 {
		t.Fatalf("pages=%d keys=%v", len(inv.Keys), inv.Keys)
	}
	if issues := inv.Check(); len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	sidebar := Sidebar(inv.Nav())
	if !strings.Contains(sidebar, "- Tool Library Pilot\n  - [Overview](/tool-library/README.md)") {
		t.Fatalf("sidebar:\n%s", sidebar)
	}
	path, frag, err := inv.Resolve("handbook/README.md", "/check-in/README.md#at-the-desk")
	if err != nil || path != "handbook/check-in/README.md" || frag != "at-the-desk" {
		t.Fatalf("resolved %q #%q %v", path, frag, err)
	}
}
func TestCaptureAndNew(t *testing.T) {
	c, dir := fixture(t)
	now := time.Date(2026, 10, 10, 8, 0, 0, 0, time.UTC)
	if err := Capture(c, "tool-library", "Follow up", now); err != nil {
		t.Fatal(err)
	}
	review, err := Review(c)
	if err != nil || !strings.Contains(review, "2026-10-10 [tool-library] Follow up") || !strings.Contains(review, "3 unprocessed item(s)") {
		t.Fatalf("review: %s %v", review, err)
	}
	path, err := New(c, "meeting", "tool-library", "Design Check", now)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, dir) {
		t.Fatal(path)
	}
	if _, err := New(c, "meeting", "tool-library", "Design Check", now); err == nil {
		t.Fatal("overwrote existing page")
	}
	if _, err := New(c, "page", "../outside", "Unsafe", now); err == nil {
		t.Fatal("escaped root")
	}
}
func TestLinkErrorsAndExclusion(t *testing.T) {
	c, dir := fixture(t)
	os.WriteFile(filepath.Join(dir, "tool-library", "kb", "broken.md"), []byte("# Broken\n\n[missing](nothing.md)\n"), 0644)
	inv, err := Load(c)
	if err != nil {
		t.Fatal(err)
	}
	if issues := inv.Check(); len(issues) != 1 || !strings.Contains(issues[0].Message, "missing target") {
		t.Fatalf("issues=%+v", issues)
	}
	if !Match("**/README.md", "handbook/check-in/README.md") || !Match("derived/**", "derived/a.md") || Match("derived/**", "tool-library/derived/a.md") {
		t.Fatal("glob semantics")
	}
}

func TestMountedAssetSource(t *testing.T) {
	c, _ := fixture(t)
	inv, err := Load(c)
	if err != nil {
		t.Fatal(err)
	}
	key := "handbook/check-in/check-in-flow.svg"
	source, err := inv.Source(key)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(filepath.ToSlash(source), "/reference/check-in/check-in-flow.svg") {
		t.Fatalf("source %q", source)
	}
	if info, err := os.Stat(source); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("mounted asset: %v", err)
	}
	if issues := inv.Check(); len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestNavigationUsesDocumentTitlesAndRootReadingOrder(t *testing.T) {
	c, _ := fixture(t)
	c.Navigation.Overrides = nil
	inv, err := Load(c)
	if err != nil {
		t.Fatal(err)
	}
	sidebar := Sidebar(inv.Nav())
	for _, expected := range []string{
		"- [Home](/README.md)\n- [Now](/NOW.md)\n- [Inbox](/inbox.md)",
		"- Tool Library\n  - [Overview](/tool-library/README.md)",
		"  - [Actions](/tool-library/actions.md)",
		"  - [Status](/tool-library/status.md)",
		"  - Knowledge Base\n    - [Overview](/tool-library/kb/README.md)\n    - [Borrowing Policy](/tool-library/kb/borrowing-policy.md)",
		"  - Source Register\n    - [Overview](/tool-library/sources/README.md)",
		"- Volunteer Handbook\n  - [Overview](/handbook/README.md)",
	} {
		if !strings.Contains(sidebar, expected) {
			t.Fatalf("missing %q in sidebar:\n%s", expected, sidebar)
		}
	}
	if strings.Contains(sidebar, "- project\n") || strings.Contains(sidebar, "- reference\n") {
		t.Fatalf("filesystem names leaked into labels:\n%s", sidebar)
	}
}

func TestNavigationOverridesAndSingleChildChain(t *testing.T) {
	c, root := fixture(t)
	dir := filepath.Join(root, "tool-library", "notes", "archive")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Archive Notes\n"), 0644); err != nil {
		t.Fatal(err)
	}
	inv, err := Load(c)
	if err != nil {
		t.Fatal(err)
	}
	sidebar := Sidebar(inv.Nav())
	if !strings.Contains(sidebar, "  - Notes / Archive Notes\n    - [Overview](/tool-library/notes/archive/README.md)") {
		t.Fatalf("chain not flattened:\n%s", sidebar)
	}
	c.Navigation.Overrides["tool-library/notes"] = config.NavOverride{Title: "Working Notes"}
	inv, err = Load(c)
	if err != nil {
		t.Fatal(err)
	}
	sidebar = Sidebar(inv.Nav())
	if strings.Contains(sidebar, "Working Notes / Archive Notes") || !strings.Contains(sidebar, "  - Working Notes\n    - Archive Notes\n") {
		t.Fatalf("override not respected:\n%s", sidebar)
	}
}

func TestNavigationUsesDirectoryForRepoStyleHeading(t *testing.T) {
	c, root := fixture(t)
	if err := os.WriteFile(filepath.Join(root, "reference", "README.md"), []byte("# team-reference\n\nHandbook contents.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c.Mounts = nil
	c.Navigation.Overrides = nil
	inv, err := Load(c)
	if err != nil {
		t.Fatal(err)
	}
	if sidebar := Sidebar(inv.Nav()); !strings.Contains(sidebar, "- Reference\n  - [Overview](/reference/README.md)") {
		t.Fatalf("repository-style title was not shortened:\n%s", sidebar)
	}
}

func TestCaptureKeepsUnprocessedSectionAndMode(t *testing.T) {
	c, dir := fixture(t)
	file := filepath.Join(dir, "inbox.md")
	if err := os.Chmod(file, 0600); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 10, 8, 0, 0, 0, time.UTC)
	if err := Capture(c, "tool-library", "Return desk follow-up", at); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	before, _, found := strings.Cut(string(contents), "## Triaged")
	if !found || !strings.Contains(before, "Return desk follow-up") || strings.Contains(string(contents), "## Triaged\n- 2026-10-10 [tool-library] Return desk follow-up") {
		t.Fatalf("capture not under Unprocessed: %s", contents)
	}
	info, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("inbox permissions changed to %04o", got)
	}
}
