package workspace

import (
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ankurs/kb/internal/config"
)

type Page struct {
	Key      string
	File     string
	Markdown string
	Title    string
	Mount    string
}
type Inventory struct {
	Config  config.Workspace
	Pages   map[string]Page
	Keys    []string
	Missing []config.Mount
}

// Match supports path globs and ** across directory separators. Patterns are
// workspace-root relative; a directory pattern ending /** includes the dir.
func Match(pattern, path string) bool {
	pattern = filepath.ToSlash(strings.TrimPrefix(pattern, "./"))
	path = filepath.ToSlash(path)
	if pattern == path {
		return true
	}
	if strings.HasSuffix(pattern, "/**") && path == strings.TrimSuffix(pattern, "/**") {
		return true
	}
	var b strings.Builder
	b.WriteByte('^')
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				if i+2 < len(pattern) && pattern[i+2] == '/' {
					b.WriteString("(?:.*/)?")
					i += 2
				} else {
					b.WriteString(".*")
					i++
				}
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(pattern[i])))
		}
	}
	b.WriteByte('$')
	return regexp.MustCompile(b.String()).MatchString(path)
}
func excluded(c config.Workspace, key string) bool {
	for _, p := range c.Exclude {
		if Match(p, key) {
			return true
		}
	}
	return false
}
func Load(c config.Workspace) (*Inventory, error) {
	inv := &Inventory{Config: c, Pages: map[string]Page{}}
	mounts := map[string]bool{}
	for _, m := range c.Mounts {
		mounts[filepath.Clean(m.Path)] = true
	}
	scan := func(path, prefix, mount string, rootScan bool) error {
		return filepath.WalkDir(path, func(file string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if file == path {
				return nil
			}
			rel, err := filepath.Rel(path, file)
			if err != nil {
				return err
			}
			key := filepath.ToSlash(filepath.Join(prefix, rel))
			if rootScan && mounts[filepath.Clean(rel)] {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasPrefix(d.Name(), ".") {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if excluded(c, key) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 {
				return nil
			} // Never follow a symlink outside the workspace.
			if d.IsDir() || !strings.EqualFold(filepath.Ext(file), ".md") || strings.HasPrefix(d.Name(), "_") || strings.EqualFold(d.Name(), "AGENTS.md") || strings.EqualFold(d.Name(), "CLAUDE.md") {
				return nil
			}
			data, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			inv.Pages[key] = Page{Key: key, File: file, Markdown: string(data), Title: Title(string(data), strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))), Mount: mount}
			return nil
		})
	}
	if err := scan(c.Root, "", "", true); err != nil {
		return nil, err
	}
	for _, m := range c.Mounts {
		dir, err := c.ReadablePath(m.Path)
		if err != nil {
			return nil, err
		}
		if info, e := os.Stat(dir); e != nil || !info.IsDir() {
			inv.Missing = append(inv.Missing, m)
			continue
		}
		if err := scan(dir, m.Name, m.Name, false); err != nil {
			return nil, err
		}
	}
	for k := range inv.Pages {
		inv.Keys = append(inv.Keys, k)
	}
	sort.Strings(inv.Keys)
	return inv, nil
}

var heading = regexp.MustCompile(`(?m)^#{1,6}\s+(.+?)\s*#*\s*$`)

func Title(md, stem string) string {
	if m := heading.FindStringSubmatch(md); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return strings.ReplaceAll(stem, "-", " ")
}
func (i *Inventory) Resolve(from, target string) (string, string, error) {
	path, frag, _ := strings.Cut(target, "#")
	path = strings.SplitN(path, "?", 2)[0]
	if decoded, err := url.PathUnescape(path); err == nil {
		path = decoded
	}
	if path == "" {
		return from, frag, nil
	}
	base := filepath.ToSlash(filepath.Dir(from))
	if strings.HasPrefix(path, "/") {
		base = ""
		if p, ok := i.Pages[from]; ok && p.Mount != "" {
			base = p.Mount
			if strings.HasPrefix(path, "/"+p.Mount+"/") {
				base = ""
			}
		}
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.Join(base, strings.TrimPrefix(path, "/"))))
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", frag, fmt.Errorf("link escapes content root: %s", target)
	}
	return clean, frag, nil
}
