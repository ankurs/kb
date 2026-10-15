package workspace

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Issue struct {
	Page    string
	Line    int
	Message string
	Warning bool
}

var headingLine = regexp.MustCompile(`^#{1,6}\s+(.+?)\s*#*\s*$`)
var htmlTag = regexp.MustCompile(`<[^>]+>`)
var slugChars = regexp.MustCompile(`[^A-Za-z0-9_\s-]`)
var whitespace = regexp.MustCompile(`\s+`)

func Slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(htmlTag.ReplaceAllString(s, "")))
	s = slugChars.ReplaceAllString(s, "")
	return whitespace.ReplaceAllString(s, "-")
}
func matching(patterns []string, key string) bool {
	for _, p := range patterns {
		if Match(p, key) {
			return true
		}
	}
	return false
}
func (i *Inventory) Check() []Issue {
	var issues []Issue
	headings := map[string]map[string]bool{}
	for _, key := range i.Keys {
		if matching(i.Config.Check.Exclude, key) {
			continue
		}
		page := i.Pages[key]
		found := map[string]bool{}
		n := 0
		fence := false
		for _, line := range strings.Split(page.Markdown, "\n") {
			trim := strings.TrimSpace(line)
			if strings.HasPrefix(trim, "```") || strings.HasPrefix(trim, "~~~") {
				fence = !fence
				continue
			}
			if fence {
				continue
			}
			if m := headingLine.FindStringSubmatch(line); len(m) > 1 {
				if strings.HasPrefix(line, "# ") {
					n++
				}
				slug := Slug(m[1])
				if found[slug] {
					issues = append(issues, Issue{key, 0, "duplicate heading #" + slug, false})
				}
				found[slug] = true
			}
		}
		if n != 1 {
			issues = append(issues, Issue{key, 0, fmt.Sprintf("expected one H1; found %d", n), false})
		}
		headings[key] = found
	}
	for _, key := range i.Keys {
		if matching(i.Config.Check.Exclude, key) {
			continue
		}
		for _, ref := range References(i.Pages[key].Markdown) {
			target := ref.Target
			if u, err := url.Parse(target); err == nil && u.Scheme != "" {
				continue
			}
			if strings.HasPrefix(target, "//") {
				continue
			}
			resolved, frag, err := i.Resolve(key, target)
			if err != nil {
				issues = append(issues, Issue{key, ref.Line, err.Error(), false})
				continue
			}
			if _, ok := i.Pages[resolved]; ok {
				if frag != "" && headings[resolved] != nil && !headings[resolved][strings.ToLower(frag)] {
					issues = append(issues, Issue{key, ref.Line, "missing heading #" + frag + " in " + resolved, false})
				}
				continue
			}
			file, err := i.Source(resolved)
			if err != nil {
				issues = append(issues, Issue{key, ref.Line, err.Error(), false})
				continue
			}
			if info, e := os.Stat(file); e == nil && info.Mode().IsRegular() {
				if !i.Config.Render.TrustHTML {
					active, e := ActiveLinkedFile(file)
					if e != nil {
						issues = append(issues, Issue{key, ref.Line, e.Error(), false})
					} else if active {
						issues = append(issues, Issue{key, ref.Line, "active linked file requires render.trust_html: true: " + target, false})
					}
				}
				continue
			}
			warning := matching(i.Config.Check.WarnMissing, key)
			for _, mount := range i.Missing {
				if strings.HasPrefix(resolved, mount.Name+"/") && mount.Fallback != "" {
					warning = true
				}
			}
			issues = append(issues, Issue{key, ref.Line, "missing target " + target, warning})
		}
	}
	return issues
}
func (i *Inventory) Source(key string) (string, error) {
	if page, ok := i.Pages[key]; ok {
		return page.File, nil
	}
	for _, mount := range i.Config.Mounts {
		if key == mount.Name {
			return i.Config.ReadablePath(mount.Path)
		}
		if strings.HasPrefix(key, mount.Name+"/") {
			return i.Config.ReadablePath(filepath.Join(mount.Path, filepath.FromSlash(strings.TrimPrefix(key, mount.Name+"/"))))
		}
	}
	return i.Config.ReadablePath(filepath.FromSlash(key))
}
