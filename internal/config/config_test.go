package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrecedenceAndDiscovery(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "child"), 0755)
	os.WriteFile(filepath.Join(dir, "kb.yaml"), []byte("version: 1\nid: demo\nname: Repository\nexclude: [private/**]\nserver: {port: 3001}\n"), 0644)
	global := filepath.Join(t.TempDir(), "global.yaml")
	os.WriteFile(global, []byte("defaults:\n  server: {port: 3002}\nworkspaces:\n  demo:\n    name: Personal\n    exclude: [other/**]\n"), 0644)
	c, err := Load(filepath.Join(dir, "child"), global)
	if err != nil {
		t.Fatal(err)
	}
	if c.Server.Port != 3002 || c.Name != "Personal" || len(c.Exclude) != 1 || c.Exclude[0] != "other/**" {
		t.Fatalf("unexpected merge: %+v", c)
	}
	if c.Root != dir {
		t.Fatalf("root %q", c.Root)
	}
}
func TestRejectUnknownAndUnsafe(t *testing.T) {
	dir := t.TempDir()
	for _, body := range []string{"version: 1\nmystery: true\n", "version: 1\noutput: {site: ../outside}\n", "version: 2\n"} {
		os.WriteFile(filepath.Join(dir, "kb.yaml"), []byte(body), 0644)
		_, err := Load(dir, filepath.Join(dir, "absent"))
		if err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	if !strings.Contains(filepath.Base(dir), "Test") {
		t.Log(dir)
	}
}
func TestInitNoOverwrite(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatal(err)
	}
	if err := Init(dir); err == nil {
		t.Fatal("overwrote config")
	}
}
func TestOutputSeparationAndSymlinkGuard(t *testing.T) {
	root := t.TempDir()
	c := Workspace{Root: root, Output: Output{Site: ".kb/site", Bundles: ".kb/site/snapshots"}, Capture: Capture{Inbox: "inbox.md", Templates: ".templates"}, Search: Search{QMDConfig: ".qmd"}, Server: Server{Port: 3000}}
	if err := c.Validate(); err == nil {
		t.Fatal("accepted overlapping output")
	}
	c.Output.Bundles = ".kb/snapshots"
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.WritablePath("linked/out.html"); err == nil {
		t.Fatal("write escaped via symlink")
	}
}

func TestPublishConfigurationGuards(t *testing.T) {
	root := t.TempDir()
	c := Workspace{Root: root, Output: Output{Site: ".kb/site", Bundles: ".kb/snapshots"}, Capture: Capture{Inbox: "inbox.md", Templates: ".templates"}, Search: Search{QMDConfig: ".qmd"}, Server: Server{Port: 3000}}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{".", "..", root, filepath.Dir(root), ".kb/site/child"} {
		c.Publish.Dir = dir
		if err := c.Validate(); err == nil {
			t.Fatalf("accepted unsafe publish.dir %q", dir)
		}
	}
	c.Publish.Dir = "dist"
	for _, name := range []string{"../wiki.html", "wiki.md", "a/b.html"} {
		c.Publish.Wiki = name
		if err := c.Validate(); err == nil {
			t.Fatalf("accepted publish.wiki %q", name)
		}
	}
	c.Publish.Wiki = "wiki.html"
	c.Publish.Names = map[string]string{"project": "other.html"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestPublishAndSyncGlobalOverrides(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "kb.yaml"), []byte("version: 1\nid: demo\npublish: {dir: dist, wiki: wiki.html}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	global := filepath.Join(t.TempDir(), "global.yaml")
	data := "workspaces:\n  demo:\n    publish: {dir: release, wiki: notes.html}\n    sync: {command: 'echo sync'}\n"
	if err := os.WriteFile(global, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(root, global)
	if err != nil {
		t.Fatal(err)
	}
	if c.Publish.Dir != "release" || c.Publish.Wiki != "notes.html" || c.Sync.Command != "echo sync" {
		t.Fatalf("config: %+v", c)
	}
}

func TestRenderTrustPrecedenceAndReadableSymlinks(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "kb.yaml"), []byte("version: 1\nid: demo\nrender: {trust_html: false}\n"), 0644)
	global := filepath.Join(t.TempDir(), "global.yaml")
	os.WriteFile(global, []byte("defaults:\n  render: {trust_html: true}\nworkspaces:\n  demo:\n    render: {trust_html: false}\n"), 0644)
	c, err := Load(root, global)
	if err != nil || c.Render.TrustHTML {
		t.Fatalf("workspace override: %+v, %v", c.Render, err)
	}
	os.WriteFile(global, []byte("defaults:\n  render: {trust_html: true}\n"), 0644)
	c, err = Load(root, global)
	if err != nil || !c.Render.TrustHTML {
		t.Fatalf("global override: %+v, %v", c.Render, err)
	}
	os.Symlink(root, filepath.Join(root, "inside"))
	if _, err := c.ReadablePath("inside/kb.yaml"); err != nil {
		t.Fatal(err)
	}
	os.Symlink(t.TempDir(), filepath.Join(root, "outside"))
	if _, err := c.ReadablePath("outside/secret.txt"); err == nil {
		t.Fatal("read escaped workspace")
	}
	os.MkdirAll(filepath.Join(root, ".kb"), 0755)
	os.Symlink(t.TempDir(), filepath.Join(root, ".kb", "snapshots"))
	if _, err := c.WritablePath(".kb/snapshots"); err == nil {
		t.Fatal("wrote through snapshot symlink")
	}
}

func TestExternalLinksNewTabPrecedence(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "kb.yaml")
	global := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("version: 1\nid: demo\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(root, global)
	if err != nil || !c.Render.OpenExternalLinksNewTab() {
		t.Fatalf("default should open new tab: %+v, %v", c.Render, err)
	}
	if !((Render{}).OpenExternalLinksNewTab()) {
		t.Fatal("programmatic workspace should use the same default")
	}
	if err := os.WriteFile(configPath, []byte("version: 1\nid: demo\nrender: {external_links_new_tab: false}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c, err = Load(root, global)
	if err != nil || c.Render.OpenExternalLinksNewTab() {
		t.Fatalf("repository override should use current tab: %+v, %v", c.Render, err)
	}
	if err := os.WriteFile(global, []byte("defaults:\n  render: {external_links_new_tab: true}\nworkspaces:\n  demo:\n    render: {external_links_new_tab: false}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c, err = Load(root, global)
	if err != nil || c.Render.OpenExternalLinksNewTab() {
		t.Fatalf("workspace-specific setting should win: %+v, %v", c.Render, err)
	}
	if err := os.WriteFile(global, []byte("defaults:\n  render: {external_links_new_tab: true}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c, err = Load(root, global)
	if err != nil || !c.Render.OpenExternalLinksNewTab() {
		t.Fatalf("user defaults should override repository: %+v, %v", c.Render, err)
	}
}

func TestServeQMDPrecedence(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "kb.yaml")
	global := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("version: 1\nid: demo\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(root, global)
	if err != nil || !c.Search.UseQMDInServe() {
		t.Fatalf("default: %+v %v", c.Search, err)
	}
	if err := os.WriteFile(global, []byte("defaults:\n  search: {serve_qmd: false}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c, err = Load(root, global)
	if err != nil || c.Search.UseQMDInServe() {
		t.Fatalf("user opt-out: %+v %v", c.Search, err)
	}
	if err := os.WriteFile(path, []byte("version: 1\nid: demo\nsearch: {serve_qmd: true}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c, err = Load(root, global)
	if err != nil || c.Search.UseQMDInServe() {
		t.Fatalf("user default precedence: %+v %v", c.Search, err)
	}
	if err := os.WriteFile(global, []byte("workspaces:\n  demo:\n    search: {serve_qmd: true}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c, err = Load(root, global)
	if err != nil || !c.Search.UseQMDInServe() {
		t.Fatalf("workspace override: %+v %v", c.Search, err)
	}
	if err := os.WriteFile(path, []byte("version: 1\nid: demo\nsearch: {serve_qmd: false}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c, err = Load(root, filepath.Join(root, "absent"))
	if err != nil || c.Search.UseQMDInServe() {
		t.Fatalf("workspace opt-out: %+v %v", c.Search, err)
	}
}
