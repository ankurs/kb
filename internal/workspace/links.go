package workspace

import (
	"path/filepath"
	"strings"
)

type Reference struct {
	Target string
	Line   int
	Image  bool
}

// ExcalidrawPreviewKey is the PNG sidecar of an image link to an editable
// Excalidraw scene. Ordinary links to the scene remain links to its source.
func ExcalidrawPreviewKey(ref Reference, key string) string {
	if ref.Image && strings.EqualFold(filepath.Ext(key), ".excalidraw") {
		return key + ".png"
	}
	return ""
}

// References finds inline Markdown links outside fenced code, including
// destinations with balanced parentheses.
func References(md string) []Reference {
	var out []Reference
	fence := false
	for n, line := range strings.Split(md, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") || strings.HasPrefix(trim, "~~~") {
			fence = !fence
			continue
		}
		if fence {
			continue
		}
		for at := 0; at < len(line); {
			i := strings.Index(line[at:], "](")
			if i < 0 {
				break
			}
			start := at + i + 2
			depth := 1
			end := -1
			for j := start; j < len(line); j++ {
				if line[j] == '\\' {
					j++
					continue
				}
				switch line[j] {
				case '(':
					depth++
				case ')':
					depth--
					if depth == 0 {
						end = j
					}
				}
				if end >= 0 {
					break
				}
			}
			if end < 0 {
				at = start
				continue
			}
			raw := strings.TrimSpace(line[start:end])
			target := ""
			if strings.HasPrefix(raw, "<") {
				if i := strings.Index(raw, ">"); i > 0 {
					target = raw[1:i]
				}
			} else {
				target = raw
				for j, c := range raw {
					if c == ' ' || c == '\t' {
						target = raw[:j]
						break
					}
				}
			}
			if target != "" {
				open := strings.LastIndex(line[:start-2], "[")
				image := open > 0 && line[open-1] == '!'
				out = append(out, Reference{Target: target, Line: n + 1, Image: image})
			}
			at = end + 1
		}
	}
	return out
}
