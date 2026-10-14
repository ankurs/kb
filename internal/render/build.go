package render

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ankurs/kb/assets"
	"github.com/ankurs/kb/internal/config"
	"github.com/ankurs/kb/internal/workspace"
)

type BundleData struct {
	Title  string            `json:"title"`
	Pages  map[string]string `json:"pages"`
	Assets map[string]string `json:"assets"`
	Files  map[string]string `json:"files"`
	Nav    *workspace.Node   `json:"nav"`
	Mounts []string          `json:"mounts,omitempty"`
}
type Result struct {
	Site     string
	Bundles  []string
	Warnings []string
}

func writeChanged(path string, data []byte) error {
	old, err := os.ReadFile(path)
	if err == nil && bytes.Equal(old, data) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".kb-write-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
func pageData(inv *workspace.Inventory, filter func(string) bool, bundleDir string, includeFiles bool) (BundleData, error) {
	out := BundleData{Title: inv.Config.Name, Pages: map[string]string{}, Assets: map[string]string{}, Files: map[string]string{}, Nav: inv.Nav()}
	for _, mount := range inv.Config.Mounts {
		out.Mounts = append(out.Mounts, mount.Name)
	}
	for _, key := range inv.Keys {
		if filter != nil && !filter(key) {
			continue
		}
		p := inv.Pages[key]
		out.Pages[key] = p.Markdown
		for _, ref := range workspace.References(p.Markdown) {
			target, _, err := inv.Resolve(key, ref.Target)
			if err != nil || target == "" {
				continue
			}
			if _, ok := inv.Pages[target]; ok {
				continue
			}
			source, err := inv.Source(target)
			if err != nil {
				continue
			}
			info, err := os.Lstat(source)
			if err != nil || !info.Mode().IsRegular() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(source))
			switch ext {
			case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp":
				b, err := os.ReadFile(source)
				if err != nil {
					return out, err
				}
				mt := mime.TypeByExtension(ext)
				if ext == ".svg" {
					mt = "image/svg+xml"
				}
				out.Assets[target] = "data:" + mt + ";base64," + base64.StdEncoding.EncodeToString(b)
			default:
				if includeFiles {
					rel, err := filepath.Rel(bundleDir, source)
					if err == nil {
						out.Files[target] = filepath.ToSlash(rel)
					}
				}
			}
		}
	}
	if filter != nil {
		out.Nav = navFromPages(out.Pages, inv.Config)
	}
	return out, nil
}
func Bundle(data BundleData) ([]byte, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString("<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>" + html.EscapeString(data.Title) + "</title><style>" + webFile("bundle.css") + "</style>")
	css, err := asset("bundle/highlight-github-dark-11.9.0.min.css")
	if err != nil {
		return nil, err
	}
	b.WriteString("<style>" + string(css) + "</style></head><body><aside><h2>" + html.EscapeString(data.Title) + "</h2><input id=\"search\" type=\"search\" placeholder=\"Search pages and content\" aria-label=\"Search\"><div id=\"results\"></div><nav id=\"nav\"></nav></aside><main><article id=\"article\"></article></main><script type=\"application/json\" id=\"page-data\">")
	b.Write(jsonData)
	b.WriteString("</script>")
	for _, name := range []string{"bundle/marked-4.3.0.min.js", "mermaid/mermaid-11.17.2.min.js", "bundle/highlight-11.9.0.min.js"} {
		src, err := asset(name)
		if err != nil {
			return nil, err
		}
		b.WriteString("<script>" + script(src) + "</script>")
	}
	b.WriteString("<script>mermaid.initialize({startOnLoad:false,theme:'dark',securityLevel:'loose'});</script><script>" + webFile("bundle.js") + "</script></body></html>")
	return []byte(b.String()), nil
}
func focusedNav(root *workspace.Node, prefix string) *workspace.Node {
	var filter func(*workspace.Node) *workspace.Node
	filter = func(n *workspace.Node) *workspace.Node {
		if n.Path != "" {
			if strings.HasPrefix(n.Path, prefix+"/") {
				v := *n
				return &v
			}
			return nil
		}
		copy := &workspace.Node{Name: n.Name}
		for _, child := range n.Children {
			if x := filter(child); x != nil {
				copy.Children = append(copy.Children, x)
			}
		}
		if len(copy.Children) == 0 {
			return nil
		}
		return copy
	}
	if n := filter(root); n != nil {
		return n
	}
	return &workspace.Node{Name: root.Name}
}
func fallback(c config.Workspace, m config.Mount) (BundleData, error) {
	var data BundleData
	file, err := c.Path(m.Fallback)
	if err != nil {
		return data, err
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return data, fmt.Errorf("mount %s missing and fallback %s unavailable: %w", m.Name, m.Fallback, err)
	}
	start := []byte(`<script type="application/json" id="page-data">`)
	end := []byte(`</script>`)
	i := bytes.Index(b, start)
	if i < 0 {
		return data, errors.New("fallback has no page-data")
	}
	b = b[i+len(start):]
	j := bytes.Index(b, end)
	if j < 0 {
		return data, errors.New("fallback page-data malformed")
	}
	if err := json.Unmarshal(b[:j], &data); err != nil {
		return data, err
	}
	if len(data.Pages) == 0 {
		return data, errors.New("fallback contains no pages")
	}
	return data, nil
}

// ExpectedBundles compiles snapshots in memory; it never changes the workspace.
func ExpectedBundles(c config.Workspace) (map[string][]byte, []string, error) {
	bundleDir, err := c.WritablePath(c.Output.Bundles)
	if err != nil {
		return nil, nil, err
	}
	return compileBundles(c, bundleDir, nil, true)
}

func compileBundles(c config.Workspace, bundleDir string, filter func(string) bool, includeFiles bool) (map[string][]byte, []string, error) {
	inv, err := workspace.Load(c)
	if err != nil {
		return nil, nil, err
	}
	data, err := pageData(inv, filter, bundleDir, includeFiles)
	if err != nil {
		return nil, nil, err
	}
	var warnings []string
	for _, m := range inv.Missing {
		if m.Fallback == "" {
			warnings = append(warnings, "optional mount "+m.Name+" absent; omitted")
			continue
		}
		snapshot, err := fallback(c, m)
		if err != nil {
			return nil, nil, err
		}
		for k, v := range snapshot.Pages {
			key := k
			if !strings.HasPrefix(key, m.Name+"/") {
				key = m.Name + "/" + key
			}
			if filter == nil || filter(key) {
				data.Pages[key] = v
			}
		}
		for k, v := range snapshot.Assets {
			key := k
			if !strings.HasPrefix(key, m.Name+"/") {
				key = m.Name + "/" + key
			}
			if filter == nil || filter(key) {
				data.Assets[key] = v
			}
		}
		warnings = append(warnings, "using fallback snapshot for mount "+m.Name)
	}
	if len(inv.Missing) > 0 || filter != nil {
		data.Nav = navFromPages(data.Pages, c)
	}
	result := map[string][]byte{}
	result["wiki.html"], err = Bundle(data)
	if err != nil {
		return nil, nil, err
	}
	groups := map[string]bool{}
	for key := range data.Pages {
		if parts := strings.SplitN(key, "/", 2); len(parts) == 2 {
			groups[parts[0]] = true
		}
	}
	for name := range groups {
		focus := BundleData{Title: c.Name + " — " + name, Pages: map[string]string{}, Assets: map[string]string{}, Files: map[string]string{}, Nav: focusedNav(data.Nav, name), Mounts: data.Mounts}
		for k, v := range data.Pages {
			if strings.HasPrefix(k, name+"/") {
				focus.Pages[k] = v
			}
		}
		for k, v := range data.Assets {
			if strings.HasPrefix(k, name+"/") {
				focus.Assets[k] = v
			}
		}
		for k, v := range data.Files {
			if strings.HasPrefix(k, name+"/") {
				focus.Files[k] = v
			}
		}
		result[name+".html"], err = Bundle(focus)
		if err != nil {
			return nil, nil, err
		}
	}
	return result, warnings, nil
}
func Verify(c config.Workspace) error {
	expected, _, err := ExpectedBundles(c)
	if err != nil {
		return err
	}
	dir, err := c.WritablePath(c.Output.Bundles)
	if err != nil {
		return err
	}
	for name, want := range expected {
		actual, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if !bytes.Equal(actual, want) {
			return fmt.Errorf("stale snapshot %s; run `kb build`", name)
		}
	}
	inv, err := workspace.Load(c)
	if err != nil {
		return err
	}
	site, err := c.WritablePath(c.Output.Site)
	if err != nil {
		return err
	}
	sidebar, err := os.ReadFile(filepath.Join(site, "_sidebar.md"))
	if err != nil {
		return fmt.Errorf("site not built: %w", err)
	}
	if !bytes.Equal(sidebar, []byte(workspace.Sidebar(inv.Nav()))) {
		return errors.New("stale site navigation; run `kb build`")
	}
	for _, key := range inv.Keys {
		b, err := os.ReadFile(filepath.Join(site, filepath.FromSlash(key)))
		if err != nil || !bytes.Equal(b, []byte(inv.Pages[key].Markdown)) {
			return fmt.Errorf("stale site page %s; run `kb build`", key)
		}
	}
	return nil
}
func Build(c config.Workspace) (Result, error) {
	return BuildWithLog(c, nil)
}

func BuildWithLog(c config.Workspace, logf func(string, ...any)) (Result, error) {
	log := func(format string, args ...any) {
		if logf != nil {
			logf(format, args...)
		}
	}
	var r Result
	log("build: compiling offline bundles")
	expected, warnings, err := ExpectedBundles(c)
	if err != nil {
		return r, err
	}
	r.Warnings = warnings
	bundleDir, err := c.WritablePath(c.Output.Bundles)
	if err != nil {
		return r, err
	}
	names := make([]string, 0, len(expected))
	for name := range expected {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		log("build: writing bundle %s (%d bytes)", name, len(expected[name]))
		path := filepath.Join(bundleDir, name)
		if err := writeChanged(path, expected[name]); err != nil {
			return r, err
		}
		r.Bundles = append(r.Bundles, path)
	}
	inv, err := workspace.Load(c)
	if err != nil {
		return r, err
	}
	log("build: writing static site")
	r.Site, err = BuildSite(inv)
	if err == nil {
		log("build: wrote %d bundles and site %s", len(r.Bundles), r.Site)
	}
	return r, err
}
func navFromPages(pages map[string]string, c config.Workspace) *workspace.Node {
	inv := &workspace.Inventory{Config: c, Pages: map[string]workspace.Page{}}
	for k, v := range pages {
		inv.Keys = append(inv.Keys, k)
		inv.Pages[k] = workspace.Page{Key: k, Markdown: v, Title: workspace.Title(v, strings.TrimSuffix(filepath.Base(k), ".md"))}
	}
	sort.Strings(inv.Keys)
	return inv.Nav()
}
func BuildSite(inv *workspace.Inventory) (string, error) {
	c := inv.Config
	site, err := c.WritablePath(c.Output.Site)
	if err != nil {
		return "", err
	}
	if filepath.Clean(site) == filepath.Clean(c.Root) {
		return "", errors.New("site output cannot be the content root")
	}
	parent := filepath.Dir(site)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return "", err
	}
	temp, err := os.MkdirTemp(parent, ".kb-site-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(temp)
	write := func(rel string, data []byte) error {
		target := filepath.Join(temp, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	}
	if err := write("index.html", []byte(LiveHTML(c, false))); err != nil {
		return "", err
	}
	if err := write("_sidebar.md", []byte(workspace.Sidebar(inv.Nav()))); err != nil {
		return "", err
	}
	if err := write("_kb/live.css", []byte(webFile("live.css"))); err != nil {
		return "", err
	}
	err = fs.WalkDir(assets.FS, "vendor", func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		b, e := fs.ReadFile(assets.FS, path)
		if e != nil {
			return e
		}
		return write(path, b)
	})
	if err != nil {
		return "", err
	}
	for _, key := range inv.Keys {
		p := inv.Pages[key]
		if err := write(key, []byte(p.Markdown)); err != nil {
			return "", err
		}
		for _, ref := range workspace.References(p.Markdown) {
			target, _, e := inv.Resolve(key, ref.Target)
			if e != nil || target == "" {
				continue
			}
			if _, ok := inv.Pages[target]; ok {
				continue
			}
			src, e := inv.Source(target)
			if e != nil {
				continue
			}
			info, e := os.Lstat(src)
			if e != nil || !info.Mode().IsRegular() {
				continue
			}
			b, e := os.ReadFile(src)
			if e != nil {
				return "", e
			}
			if e := write(target, b); e != nil {
				return "", e
			}
		}
	}
	if err := write(".kb-generated", []byte("kb generated site\n")); err != nil {
		return "", err
	}
	if info, e := os.Lstat(site); e == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("refusing to replace non-directory %s", site)
		}
		if _, e := os.Stat(filepath.Join(site, ".kb-generated")); e != nil {
			return "", fmt.Errorf("refusing to replace non-kb directory %s", site)
		}
		if e := os.RemoveAll(site); e != nil {
			return "", e
		}
	} else if !os.IsNotExist(e) {
		return "", e
	}
	if err := os.Rename(temp, site); err != nil {
		return "", err
	}
	return site, nil
}
