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
	Title     string            `json:"title"`
	Pages     map[string]string `json:"pages"`
	Assets    map[string]string `json:"assets"`
	Files     map[string]string `json:"files"`
	Nav       *workspace.Node   `json:"nav"`
	Mounts    []string          `json:"mounts,omitempty"`
	TrustHTML bool              `json:"trustHTML"`
}
type Result struct {
	Site     string
	Bundles  []string
	Warnings []string
}

func writeChanged(path string, data []byte) error {
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing snapshot symlink %s", path)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
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
	out := BundleData{Title: inv.Config.Name, Pages: map[string]string{}, Assets: map[string]string{}, Files: map[string]string{}, Nav: inv.Nav(), TrustHTML: inv.Config.Render.TrustHTML}
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
			if err != nil {
				return out, err
			}
			if target == "" {
				continue
			}
			if _, ok := inv.Pages[target]; ok {
				continue
			}
			source, err := inv.Source(target)
			if err != nil {
				return out, err
			}
			info, err := os.Stat(source)
			if err != nil || !info.Mode().IsRegular() {
				continue
			}
			if err := checkActiveResource(inv.Config, source, target); err != nil {
				return out, err
			}
			ext := strings.ToLower(filepath.Ext(source))
			switch ext {
			case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp":
				b, err := os.ReadFile(source)
				if err != nil {
					return out, err
				}
				if ext == ".svg" {
					if err := safeSVG(b); err != nil {
						return out, fmt.Errorf("%s: %w", target, err)
					}
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
	for _, name := range []string{"bundle/marked-4.3.0.min.js", "bundle/purify-3.2.6.min.js", "mermaid/mermaid-11.17.2.min.js", "bundle/highlight-11.9.0.min.js"} {
		src, err := asset(name)
		if err != nil {
			return nil, err
		}
		b.WriteString("<script>" + script(src) + "</script>")
	}
	b.WriteString("<script>mermaid.initialize({startOnLoad:false,theme:'dark',securityLevel:'strict'});</script><script>" + webFile("bundle.js") + "</script></body></html>")
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
	file, err := c.ReadablePath(m.Fallback)
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
	for key, value := range data.Assets {
		media, encoded, ok := strings.Cut(value, ",")
		if !ok || !strings.HasSuffix(media, ";base64") {
			return data, fmt.Errorf("invalid fallback asset %s", key)
		}
		kind := strings.TrimSuffix(media, ";base64")
		switch kind {
		case "data:image/png", "data:image/jpeg", "data:image/gif", "data:image/webp":
		case "data:image/svg+xml":
			b, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				return data, fmt.Errorf("invalid fallback SVG %s: %w", key, err)
			}
			if err := safeSVG(b); err != nil {
				return data, fmt.Errorf("unsafe fallback SVG %s: %w", key, err)
			}
		default:
			return data, fmt.Errorf("unsupported fallback asset %s", key)
		}
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
	data, warnings, err := effectiveData(inv, bundleDir, filter, includeFiles)
	if err != nil {
		return nil, nil, err
	}
	result, err := compileData(c, data, filter)
	return result, warnings, err
}

// effectiveData reads only selected pages and mounts. Build passes nil to
// include the complete inventory; publish selects folders before calling it.
func effectiveData(inv *workspace.Inventory, bundleDir string, filter func(string) bool, includeFiles bool) (BundleData, []string, error) {
	c := inv.Config
	data, err := pageData(inv, filter, bundleDir, includeFiles)
	if err != nil {
		return BundleData{}, nil, err
	}
	var warnings []string
	for _, m := range inv.Missing {
		if filter != nil && !filter(m.Name+"/README.md") {
			continue
		}
		if m.Fallback == "" {
			return BundleData{}, nil, fmt.Errorf("mount %s missing and no fallback configured", m.Name)
		}
		snapshot, err := fallback(c, m)
		if err != nil {
			return BundleData{}, nil, err
		}
		// Current focused snapshots use workspace-relative keys for both pages
		// and assets. Legacy mount-relative snapshots need both normalized.
		canonical, relative := false, false
		for k := range snapshot.Pages {
			if strings.HasPrefix(k, m.Name+"/") {
				canonical = true
			} else {
				relative = true
			}
		}
		if canonical && relative {
			return BundleData{}, nil, fmt.Errorf("mount %s fallback mixes workspace- and mount-relative pages", m.Name)
		}
		for k, v := range snapshot.Pages {
			key := k
			if relative {
				key = m.Name + "/" + key
			}
			data.Pages[key] = v
		}
		for k, v := range snapshot.Assets {
			key := k
			if relative && !strings.HasPrefix(key, m.Name+"/") {
				key = m.Name + "/" + key
			}
			// A live page may reference the same workspace-level asset. Keep
			// the current file rather than replacing it with an older snapshot.
			if _, live := data.Assets[key]; !live {
				data.Assets[key] = v
			}
		}
		warnings = append(warnings, "using fallback snapshot for mount "+m.Name)
	}
	if len(inv.Missing) > 0 || filter != nil {
		data.Nav = navFromPages(data.Pages, c)
	}
	data.TrustHTML = c.Render.TrustHTML
	return data, warnings, nil
}

func compileData(c config.Workspace, data BundleData, filter func(string) bool) (map[string][]byte, error) {
	var err error
	if filter != nil {
		for key := range data.Pages {
			if !filter(key) {
				delete(data.Pages, key)
			}
		}
		data.Assets, data.Files = resourcesForPages(data, data.Pages)
		data.Nav = navFromPages(data.Pages, c)
	}
	result := map[string][]byte{}
	result["wiki.html"], err = Bundle(data)
	if err != nil {
		return nil, err
	}
	groups := map[string]bool{}
	for key := range data.Pages {
		if parts := strings.SplitN(key, "/", 2); len(parts) == 2 {
			groups[parts[0]] = true
		}
	}
	for name := range groups {
		focus := BundleData{Title: c.Name + " — " + name, Pages: map[string]string{}, Assets: map[string]string{}, Files: map[string]string{}, Nav: focusedNav(data.Nav, name), Mounts: data.Mounts, TrustHTML: c.Render.TrustHTML}
		for k, v := range data.Pages {
			if strings.HasPrefix(k, name+"/") {
				focus.Pages[k] = v
			}
		}
		focus.Assets, focus.Files = resourcesForPages(data, focus.Pages)
		result[name+".html"], err = Bundle(focus)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// A filtered page may reference an image outside its own top-level folder.
// Select resources by actual page references, not by the resource's folder.
func resourcesForPages(data BundleData, pages map[string]string) (map[string]string, map[string]string) {
	assets, files := map[string]string{}, map[string]string{}
	inv := &workspace.Inventory{Pages: map[string]workspace.Page{}}
	for key := range pages {
		for _, mount := range data.Mounts {
			if strings.HasPrefix(key, mount+"/") {
				inv.Pages[key] = workspace.Page{Mount: mount}
				break
			}
		}
	}
	for key, md := range pages {
		for _, ref := range workspace.References(md) {
			target, _, err := inv.Resolve(key, ref.Target)
			if err != nil {
				continue
			}
			if value, ok := data.Assets[target]; ok {
				assets[target] = value
			}
			if value, ok := data.Files[target]; ok {
				files[target] = value
			}
		}
	}
	return assets, files
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
		path := filepath.Join(dir, name)
		if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
			return fmt.Errorf("stale snapshot %s; run `kb build`", name)
		}
		actual, err := os.ReadFile(path)
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
	files, err := siteFiles(inv)
	if err != nil {
		return err
	}
	for name, want := range files {
		path := filepath.Join(site, filepath.FromSlash(name))
		info, e := os.Lstat(path)
		if e != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("stale site file %s; run `kb build`", name)
		}
		got, e := os.ReadFile(path)
		if e != nil || !bytes.Equal(got, want) {
			return fmt.Errorf("stale site file %s; run `kb build`", name)
		}
	}
	if err := filepath.WalkDir(site, func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		rel, e := filepath.Rel(site, path)
		if e != nil {
			return e
		}
		if _, ok := files[filepath.ToSlash(rel)]; !ok {
			return fmt.Errorf("stale site file %s; run `kb build`", rel)
		}
		return nil
	}); err != nil {
		return err
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

// siteFiles is the single inventory for build and read-only verification.
func siteFiles(inv *workspace.Inventory) (map[string][]byte, error) {
	c := inv.Config
	files := map[string][]byte{
		"index.html":    []byte(LiveHTML(c, false)),
		"_sidebar.md":   []byte(workspace.Sidebar(inv.Nav())),
		"_kb/live.css":  []byte(webFile("live.css")),
		".kb-generated": []byte("kb generated site\n"),
	}
	err := fs.WalkDir(assets.FS, "vendor", func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		b, e := fs.ReadFile(assets.FS, path)
		if e == nil {
			files[path] = b
		}
		return e
	})
	if err != nil {
		return nil, err
	}
	protected := make(map[string]bool, len(files))
	for name := range files {
		protected[name] = true
	}
	for _, key := range inv.Keys {
		p := inv.Pages[key]
		files[key] = []byte(p.Markdown)
		for _, ref := range workspace.References(p.Markdown) {
			target, _, e := inv.Resolve(key, ref.Target)
			if e != nil {
				return nil, e
			}
			if target == "" {
				continue
			}
			if _, ok := inv.Pages[target]; ok {
				continue
			}
			src, e := inv.Source(target)
			if e != nil {
				return nil, e
			}
			info, e := os.Stat(src)
			if e != nil || !info.Mode().IsRegular() {
				continue
			}
			if e := checkActiveResource(c, src, target); e != nil {
				return nil, e
			}
			b, e := os.ReadFile(src)
			if e != nil {
				return nil, e
			}
			if strings.EqualFold(filepath.Ext(src), ".svg") {
				if e := safeSVG(b); e != nil {
					return nil, fmt.Errorf("%s: %w", target, e)
				}
			}
			if protected[target] {
				return nil, fmt.Errorf("linked file %s collides with generated site file", target)
			}
			files[target] = b
		}
	}
	return files, nil
}
func BuildSite(inv *workspace.Inventory) (string, error) {
	c := inv.Config
	files, err := siteFiles(inv)
	if err != nil {
		return "", err
	}
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
	for name, data := range files {
		if err := write(name, data); err != nil {
			return "", err
		}
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
