package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Workspace configures a Markdown tree and its outputs. All paths except Root
// are relative to the content root.
type Workspace struct {
	Version    int        `yaml:"version"`
	ID         string     `yaml:"id"`
	Name       string     `yaml:"name"`
	Root       string     `yaml:"root"`
	Exclude    []string   `yaml:"exclude"`
	Mounts     []Mount    `yaml:"mounts"`
	Navigation Navigation `yaml:"navigation"`
	Check      Check      `yaml:"check"`
	Render     Render     `yaml:"render"`
	Output     Output     `yaml:"output"`
	Capture    Capture    `yaml:"capture"`
	Search     Search     `yaml:"search"`
	Publish    Publish    `yaml:"publish"`
	Sync       Sync       `yaml:"sync"`
	Server     Server     `yaml:"server"`
	ConfigPath string     `yaml:"-"`
}
type Mount struct {
	Name     string `yaml:"name"`
	Path     string `yaml:"path"`
	Fallback string `yaml:"fallback"`
}
type Navigation struct {
	Overrides map[string]NavOverride `yaml:"overrides"`
}
type NavOverride struct {
	Title string `yaml:"title"`
	Order int    `yaml:"order"`
	Hide  bool   `yaml:"hide"`
}
type Check struct {
	Exclude     []string `yaml:"exclude"`
	WarnMissing []string `yaml:"warn_missing"`
}
type Render struct {
	TrustHTML bool `yaml:"trust_html"`
}
type Output struct {
	Site    string `yaml:"site"`
	Bundles string `yaml:"bundles"`
}
type Capture struct {
	Inbox     string `yaml:"inbox"`
	Templates string `yaml:"templates"`
}
type Search struct {
	Provider  string `yaml:"provider"`
	QMDConfig string `yaml:"qmd_config"`
}
type Publish struct {
	Dir     string            `yaml:"dir"`
	Wiki    string            `yaml:"wiki"`
	Include []string          `yaml:"include"`
	Root    bool              `yaml:"root"`
	Exclude []string          `yaml:"exclude"`
	Names   map[string]string `yaml:"names"`
}
type Sync struct {
	Command string `yaml:"command"`
}
type Server struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type global struct {
	Defaults   map[string]any            `yaml:"defaults"`
	Workspaces map[string]map[string]any `yaml:"workspaces"`
}

func defaults() map[string]any {
	return map[string]any{"version": 1, "root": ".", "exclude": []any{".git/**", ".kb/**", ".qmd/**", ".agents/**", ".claude/**", ".obsidian/**", ".plans/**", "**/.plans/**", "scripts/**", "vendor/**", "dist/**", "derived/**", "**/AGENTS.md", "**/CLAUDE.md", "**/_sidebar.md", "**/_coverpage.md"}, "output": map[string]any{"site": ".kb/site", "bundles": ".kb/snapshots"}, "capture": map[string]any{"inbox": "inbox.md", "templates": ".templates"}, "search": map[string]any{"provider": "qmd", "qmd_config": ".qmd"}, "publish": map[string]any{"dir": "dist", "wiki": "wiki.html"}, "server": map[string]any{"host": "127.0.0.1", "port": 3000}}
}

func merge(dst, src map[string]any) {
	for k, v := range src {
		if sm, ok := v.(map[string]any); ok {
			if dm, ok := dst[k].(map[string]any); ok {
				merge(dm, sm)
				continue
			}
		}
		dst[k] = v
	}
}
func readMap(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err = yaml.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if m == nil {
		return nil, fmt.Errorf("%s: empty configuration", path)
	}
	return m, nil
}
func strict(m map[string]any, out any) error {
	b, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	d := yaml.NewDecoder(bytes.NewReader(b))
	d.KnownFields(true)
	return d.Decode(out)
}
func Find(start string) (string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(abs); err == nil && !info.IsDir() {
		abs = filepath.Dir(abs)
	}
	for {
		candidate := filepath.Join(abs, "kb.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		next := filepath.Dir(abs)
		if next == abs {
			break
		}
		abs = next
	}
	return "", errors.New("kb.yaml not found; run `kb init` in a content repository or pass --workspace")
}
func Load(start, globalPath string) (Workspace, error) {
	path, err := Find(start)
	if err != nil {
		return Workspace{}, err
	}
	repo, err := readMap(path)
	if err != nil {
		return Workspace{}, err
	}
	id, _ := repo["id"].(string)
	merged := defaults()
	merge(merged, repo)
	if globalPath == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			globalPath = filepath.Join(home, ".config", "kb", "config.yaml")
		}
	}
	if globalPath != "" {
		var g global
		if b, err := os.ReadFile(globalPath); err == nil {
			d := yaml.NewDecoder(bytes.NewReader(b))
			d.KnownFields(true)
			if err := d.Decode(&g); err != nil {
				return Workspace{}, fmt.Errorf("%s: %w", globalPath, err)
			}
			merge(merged, g.Defaults)
			merge(merged, g.Workspaces[id])
		} else if !errors.Is(err, os.ErrNotExist) {
			return Workspace{}, err
		}
	}
	var c Workspace
	if err := strict(merged, &c); err != nil {
		return c, fmt.Errorf("invalid kb configuration: %w", err)
	}
	if c.Version != 1 {
		return c, fmt.Errorf("unsupported kb.yaml version %d", c.Version)
	}
	if c.ID == "" {
		c.ID = strings.TrimSuffix(filepath.Base(filepath.Dir(path)), filepath.Ext(path))
	}
	if c.Name == "" {
		c.Name = c.ID
	}
	c.ConfigPath = path
	base := filepath.Dir(path)
	if filepath.IsAbs(c.Root) {
		return c, errors.New("root must be relative to kb.yaml")
	}
	c.Root = filepath.Clean(filepath.Join(base, c.Root))
	if err := c.Validate(); err != nil {
		return c, err
	}
	return c, nil
}
func (c Workspace) Validate() error {
	if info, err := os.Stat(c.Root); err != nil || !info.IsDir() {
		return fmt.Errorf("content root is not a directory: %s", c.Root)
	}
	for _, p := range []string{c.Output.Site, c.Output.Bundles, c.Capture.Inbox, c.Capture.Templates, c.Search.QMDConfig} {
		if !SafeRelative(p) {
			return fmt.Errorf("unsafe relative path: %q", p)
		}
	}
	for _, m := range c.Mounts {
		if m.Name == "" || strings.ContainsAny(m.Name, "/\\") || !SafeRelative(m.Path) || m.Path == "." || (m.Fallback != "" && !SafeRelative(m.Fallback)) {
			return fmt.Errorf("invalid mount: %+v", m)
		}
	}
	site, bundles := filepath.Clean(c.Output.Site), filepath.Clean(c.Output.Bundles)
	if site == "." || bundles == "." || site == bundles || strings.HasPrefix(site, bundles+string(filepath.Separator)) || strings.HasPrefix(bundles, site+string(filepath.Separator)) {
		return errors.New("output.site and output.bundles must be distinct, non-overlapping subdirectories")
	}
	for _, m := range c.Mounts {
		for _, out := range []string{site, bundles} {
			mount := filepath.Clean(m.Path)
			if out == mount || strings.HasPrefix(out, mount+string(filepath.Separator)) {
				return fmt.Errorf("output %s overlaps mount %s", out, m.Name)
			}
		}
	}
	if !ValidHTMLName(c.Publish.WikiName()) {
		return fmt.Errorf("invalid publish.wiki filename %q", c.Publish.Wiki)
	}
	for _, group := range append(append([]string{}, c.Publish.Include...), c.Publish.Exclude...) {
		if group == "" || strings.ContainsAny(group, "/\\") || group == "." || group == ".." {
			return fmt.Errorf("invalid publish folder %q", group)
		}
	}
	for group, name := range c.Publish.Names {
		if group == "" || strings.ContainsAny(group, "/\\") || !ValidHTMLName(name) {
			return fmt.Errorf("invalid publish.names entry %q: %q", group, name)
		}
	}
	publishDir, err := c.PublishDirectory()
	if err != nil {
		return err
	}
	root := filepath.Clean(c.Root)
	if publishDir == root || publishDir == string(filepath.Separator) || strings.HasPrefix(root, publishDir+string(filepath.Separator)) {
		return errors.New("publish.dir cannot be the workspace root or its parent")
	}
	for _, other := range []string{site, bundles} {
		otherDir := filepath.Join(root, other)
		if publishDir == otherDir || strings.HasPrefix(publishDir, otherDir+string(filepath.Separator)) || strings.HasPrefix(otherDir, publishDir+string(filepath.Separator)) {
			return fmt.Errorf("publish.dir overlaps %s", other)
		}
	}
	for _, m := range c.Mounts {
		mountDir := filepath.Join(root, m.Path)
		if publishDir == mountDir || strings.HasPrefix(publishDir, mountDir+string(filepath.Separator)) || strings.HasPrefix(mountDir, publishDir+string(filepath.Separator)) {
			return fmt.Errorf("publish.dir overlaps mount %s", m.Name)
		}
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return errors.New("server.port must be between 1 and 65535")
	}
	return nil
}
func SafeRelative(path string) bool {
	if path == "" || filepath.IsAbs(path) {
		return false
	}
	p := filepath.Clean(path)
	return p != ".." && !strings.HasPrefix(p, ".."+string(filepath.Separator))
}
func (c Workspace) Path(rel string) (string, error) {
	if !SafeRelative(rel) {
		return "", fmt.Errorf("unsafe path: %q", rel)
	}
	return filepath.Join(c.Root, rel), nil
}

func inside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// ReadablePath resolves symlinks in the source and its parents before a read.
// A missing leaf is allowed for ordinary missing-link diagnostics.
func (c Workspace) ReadablePath(rel string) (string, error) {
	target, err := c.Path(rel)
	if err != nil {
		return "", err
	}
	root, err := filepath.EvalSymlinks(c.Root)
	if err != nil {
		return "", err
	}
	ancestor := target
	for {
		if _, err := os.Lstat(ancestor); err == nil {
			break
		} else if !os.IsNotExist(err) {
			return "", err
		}
		next := filepath.Dir(ancestor)
		if next == ancestor {
			return "", errors.New("cannot resolve source")
		}
		ancestor = next
	}
	resolved, err := filepath.EvalSymlinks(ancestor)
	if err != nil {
		return "", fmt.Errorf("invalid source symlink %s: %w", rel, err)
	}
	if !inside(root, resolved) {
		return "", fmt.Errorf("source escapes content root: %s", rel)
	}
	return target, nil
}

// WritablePath rejects destinations with parent symlinks outside the content root.
func (c Workspace) WritablePath(rel string) (string, error) {
	target, err := c.Path(rel)
	if err != nil {
		return "", err
	}
	if info, err := os.Lstat(target); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("destination is a symlink: %s", rel)
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	root, err := filepath.EvalSymlinks(c.Root)
	if err != nil {
		return "", err
	}
	ancestor := filepath.Dir(target)
	for {
		if _, err := os.Lstat(ancestor); err == nil {
			break
		} else if !os.IsNotExist(err) {
			return "", err
		}
		next := filepath.Dir(ancestor)
		if next == ancestor {
			return "", errors.New("cannot resolve destination parent")
		}
		ancestor = next
	}
	resolved, err := filepath.EvalSymlinks(ancestor)
	if err != nil {
		return "", err
	}
	if !inside(root, resolved) {
		return "", fmt.Errorf("destination escapes content root: %s", rel)
	}
	return target, nil
}
func (p Publish) WikiName() string {
	if p.Wiki == "" {
		return "wiki.html"
	}
	return p.Wiki
}
func ValidHTMLName(name string) bool {
	return name != "" && name != "." && name != ".." && filepath.Base(name) == name && !strings.ContainsAny(name, "/\\") && strings.HasSuffix(strings.ToLower(name), ".html")
}
func (c Workspace) PublishDirectory() (string, error) {
	dir := c.Publish.Dir
	if dir == "" {
		dir = "dist"
	}
	if filepath.IsAbs(dir) {
		return filepath.Clean(dir), nil
	}
	if !SafeRelative(dir) || filepath.Clean(dir) == "." {
		return "", fmt.Errorf("unsafe publish.dir %q", dir)
	}
	return c.WritablePath(dir)
}
func Init(dir string) error {
	path := filepath.Join(dir, "kb.yaml")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString("version: 1\nid: " + filepath.Base(dir) + "\n" + `# root: .
# mounts: [{name: handbook, path: reference, fallback: .kb/snapshots/handbook.html}]
# navigation:
#   overrides: {handbook: {title: Handbook, order: 10}}
# output: {site: .kb/site, bundles: .kb/snapshots}
# publish: {dir: dist, wiki: wiki.html}
# sync: {command: 'rsync -av -- "$KB_PUBLISH_DIR/" "$HOME/wiki-backup/"'}
`)
	return err
}
