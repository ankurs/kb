package workspace

import "strings"

type Reference struct {
	Target string
	Line   int
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
				out = append(out, Reference{target, n + 1})
			}
			at = end + 1
		}
	}
	return out
}
