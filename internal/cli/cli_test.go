package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCLI(t *testing.T, args ...string) (string, string) {
	t.Helper()
	root := New()
	var out, diagnostics bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&diagnostics)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	return out.String(), diagnostics.String()
}

func sample(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "testdata", "sample"))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestGlobalVerboseFlag(t *testing.T) {
	path := sample(t)
	config := filepath.Join(t.TempDir(), "absent.yaml")
	common := []string{"--workspace", path, "--global-config", config}
	plain, stderr := runCLI(t, append(append([]string{}, common...), "check")...)
	if stderr != "" {
		t.Fatalf("unexpected diagnostics: %q", stderr)
	}
	for _, args := range [][]string{
		append(append([]string{}, common...), "-v", "check"),
		append(append([]string{}, common...), "check", "--verbose"),
	} {
		out, logs := runCLI(t, args...)
		if out != plain {
			t.Fatalf("verbose changed stdout: %q != %q", out, plain)
		}
		if !strings.Contains(logs, "kb: loading workspace") || !strings.Contains(logs, "kb: check: validating 37 pages") {
			t.Fatalf("missing verbose diagnostics: %q", logs)
		}
	}
}

func copySample(t *testing.T) string {
	t.Helper()
	source := sample(t)
	target := t.TempDir()
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if (relative == ".kb" || relative == "dist") && entry.IsDir() {
			return filepath.SkipDir
		}
		if relative == filepath.Join(".qmd", "index.sqlite") || strings.HasPrefix(relative, filepath.Join(".qmd", "index.sqlite")+"-") {
			return nil
		}
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0755)
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, contents, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return target
}

func TestBuildVerboseLogging(t *testing.T) {
	target := copySample(t)
	_, logs := runCLI(t, "--workspace", target, "--global-config", filepath.Join(target, "absent.yaml"), "build", "-v")
	for _, expected := range []string{"build: compiling offline bundles", "build: writing bundle wiki.html", "build: writing static site"} {
		if !strings.Contains(logs, expected) {
			t.Fatalf("missing %q in %q", expected, logs)
		}
	}
}

func TestToolLibraryWorkflow(t *testing.T) {
	target := copySample(t)
	common := []string{"--workspace", target, "--global-config", filepath.Join(target, "absent.yaml")}
	run := func(args ...string) string {
		t.Helper()
		out, _ := runCLI(t, append(append([]string{}, common...), args...)...)
		return out
	}
	if got := run("review"); !strings.Contains(got, "2 unprocessed item(s)") || !strings.Contains(got, "another branch") {
		t.Fatal(got)
	}
	if got := run("find", "--scope", "tool-library", "seven calendar days"); !strings.Contains(got, "tool-library/kb/borrowing-policy.md") {
		t.Fatal(got)
	}
	if got := run("check"); !strings.Contains(got, "checked 37 pages; 0 error(s)") {
		t.Fatal(got)
	}
	run("capture", "--at", "tool-library", "Confirm where returned tools wait for inspection")
	if got := run("review"); !strings.Contains(got, "3 unprocessed item(s)") || !strings.Contains(got, "returned tools wait") {
		t.Fatal(got)
	}
	run("new", "meeting", "--at", "tool-library", "--title", "Friday Desk Review", "--date", "2026-10-10")
	run("new", "page", "--at", "tool-library", "--title", "Return Locations")
	if got := run("check"); !strings.Contains(got, "checked 39 pages; 0 error(s)") {
		t.Fatal(got)
	}
	run("build")
	if got := run("verify"); !strings.Contains(got, "content and offline outputs current") {
		t.Fatal(got)
	}
	if _, err := os.Stat(filepath.Join(target, ".kb", "site", "tool-library", "sources", "desk-rules.txt")); err != nil {
		t.Fatalf("linked source absent from static site: %v", err)
	}
	bundle, err := os.ReadFile(filepath.Join(target, ".kb", "snapshots", "wiki.html"))
	if err != nil || (!bytes.Contains(bundle, []byte("tool-library/kb/return-locations.md")) || !bytes.Contains(bundle, []byte("desk-rules.txt"))) {
		t.Fatalf("new page absent from offline bundle: %v", err)
	}
}

func TestDoctorWithoutProjectQMDConfig(t *testing.T) {
	root := copySample(t)
	if err := os.RemoveAll(filepath.Join(root, ".qmd")); err != nil {
		t.Fatal(err)
	}
	output, diagnostics := runCLI(t, "--workspace", root, "--global-config", filepath.Join(t.TempDir(), "absent.yaml"), "doctor")
	if diagnostics != "" {
		t.Fatalf("stderr: %q", diagnostics)
	}
	if strings.Count(output, "optional:") != 1 || !strings.Contains(output, "qmd init") || strings.Contains(output, "not trusted") {
		t.Fatalf("doctor output: %q", output)
	}
}

func TestDoctorWithoutQMDCollections(t *testing.T) {
	root := copySample(t)
	dir := filepath.Join(root, ".qmd")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.yml"), []byte("collections: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	output, diagnostics := runCLI(t, "--workspace", root, "--global-config", filepath.Join(root, "absent.yaml"), "doctor")
	if diagnostics != "" || strings.Count(output, "optional:") != 1 || !strings.Contains(output, "no qmd collections") || strings.Contains(output, "not trusted") {
		t.Fatalf("doctor: %q, stderr: %q", output, diagnostics)
	}
}

func TestHybridSearchCommand(t *testing.T) {
	root := copySample(t)
	dir := filepath.Join(root, ".qmd")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.yml"), []byte("collections:\n  wiki:\n    path: .\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "qmd")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nif [ \"$1\" = query ]; then echo 'ranked borrowing policy result'; fi\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	common := []string{"--workspace", root, "--global-config", filepath.Join(root, "absent.yaml")}
	output, diagnostics := runCLI(t, append(append([]string{}, common...), "search", "--scope", "wiki", "borrowing policy")...)
	if diagnostics != "" || !strings.Contains(output, "ranked borrowing policy result") {
		t.Fatalf("search output: %q; diagnostics: %q", output, diagnostics)
	}
	command := New()
	command.SetArgs(append(append([]string{}, common...), "ask", "borrowing policy"))
	if err := command.Execute(); err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("old command accepted: %v", err)
	}
}

func TestPublishAndSyncCLI(t *testing.T) {
	root := copySample(t)
	destination := filepath.Join(t.TempDir(), "distribution")
	common := []string{"--workspace", root, "--global-config", filepath.Join(root, "absent.yaml")}
	file := filepath.Join(root, "kb.yaml")
	contents, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	old := `    cp -R "$KB_PUBLISH_DIR/." "$KB_WORKSPACE_ROOT/.kb/mirror/"`
	replacement := `    touch "$KB_WORKSPACE_ROOT/sync-ran"`
	updated := strings.Replace(string(contents), old, replacement, 1)
	if updated == string(contents) {
		t.Fatal("sample sync command changed")
	}
	if err := os.WriteFile(file, []byte(updated), 0644); err != nil {
		t.Fatal(err)
	}

	marker := filepath.Join(root, "sync-ran")
	args := append(append([]string{}, common...), "publish", "--output", destination, "--wiki", "index.html", "--include", "tool-library")
	output, _ := runCLI(t, args...)
	if !strings.Contains(output, "published: "+destination) {
		t.Fatal(output)
	}
	for _, name := range []string{"index.html", "borrowing-pilot.html"} {
		if _, err := os.Stat(filepath.Join(destination, name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(destination, "handbook.html")); !os.IsNotExist(err) {
		t.Fatalf("unselected bundle present: %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("publish executed sync: %v", err)
	}
	shown, _ := runCLI(t, append(append([]string{}, common...), "sync", "--show")...)
	if !strings.Contains(shown, "command:") || !strings.Contains(shown, "publish:") {
		t.Fatal(shown)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("--show executed sync: %v", err)
	}
	runCLI(t, append(append([]string{}, common...), "sync")...)
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("sync did not run: %v", err)
	}
}

func TestUsageErrors(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want []string
		not  []string
	}{
		{[]string{"new", "hello"}, []string{`unknown page kind "hello"`, "Usage:", "kb new page --at", "Run 'kb new --help'"}, nil},
		{[]string{"new", "page"}, []string{"missing required flags --at and --title", "project directory"}, nil},
		{[]string{"new", "page", "--at", "x"}, []string{"missing required flag --title"}, []string{"--at       project"}},
		{[]string{"new", "page", "--at", "x", "--title", "y", "--date", "bad"}, []string{`invalid --date "bad"`, "YYYY-MM-DD"}, nil},
		{[]string{"review", "extra"}, []string{"takes no arguments", "kb review"}, nil},
		{[]string{"find"}, []string{"missing QUERY argument", "kb find QUERY"}, nil},
		{[]string{"find", "a", "b"}, []string{"quote multi-word text"}, nil},
		{[]string{"chek"}, []string{`unknown command "chek"`, "did you mean check?", "Run 'kb --help' for all commands"}, nil},
		{[]string{"skill", "bogus"}, []string{`unknown command "bogus" for "kb skill"`}, nil},
		{[]string{"new", "--bogus", "page"}, []string{"unknown flag: --bogus", "Usage:"}, nil},
	} {
		root := New()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs(append([]string{"--workspace", sample(t)}, tc.args...))
		err := root.Execute()
		if err == nil {
			t.Errorf("%v: expected error", tc.args)
			continue
		}
		for _, want := range tc.want {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%v: error %q lacks %q", tc.args, err, want)
			}
		}
		for _, not := range tc.not {
			if strings.Contains(err.Error(), not) {
				t.Errorf("%v: error %q unexpectedly contains %q", tc.args, err, not)
			}
		}
	}
}

func TestRootWithoutArgsPrintsHelp(t *testing.T) {
	out, _ := runCLI(t)
	if !strings.Contains(out, "Available Commands:") {
		t.Fatalf("expected help, got %q", out)
	}
}
