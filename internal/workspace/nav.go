package workspace

import (
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

type Node struct {
	Name     string  `json:"name"`
	Path     string  `json:"path,omitempty"`
	Children []*Node `json:"children,omitempty"`
	Order    int     `json:"-"`
	dir      string
	raw      string
	explicit bool
}

var anchorSuffix = regexp.MustCompile(`\s*\{#[^}]*\}\s*$`)
var titleMarkup = strings.NewReplacer("*", "", "_", "", "`", "")
var qualifier = regexp.MustCompile(`\s+(?:[—–·|]|-{1,2}|/|×|:)\s+`)
var labelWords = regexp.MustCompile(`[\pL\pN]+`)

func cleanTitle(title string) string {
	return strings.TrimSpace(titleMarkup.Replace(anchorSuffix.ReplaceAllString(title, "")))
}

func prettyDirectory(name string) string {
	if name == "kb" {
		return "Knowledge Base"
	}
	name = strings.Trim(name, "_")
	name = strings.ReplaceAll(strings.ReplaceAll(name, "-", " "), "_", " ")
	words := strings.Fields(name)
	for idx, word := range words {
		runes := []rune(strings.ToLower(word))
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
		}
		words[idx] = string(runes)
	}
	return strings.Join(words, " ")
}

func wordSet(title string) map[string]bool {
	words := map[string]bool{}
	for _, word := range labelWords.FindAllString(strings.ToLower(title), -1) {
		words[word] = true
	}
	return words
}

func addWords(context map[string]bool, title string) map[string]bool {
	next := make(map[string]bool, len(context)+4)
	for word := range context {
		next[word] = true
	}
	for word := range wordSet(title) {
		next[word] = true
	}
	return next
}

func contained(words, context map[string]bool) bool {
	if len(words) == 0 {
		return false
	}
	for word := range words {
		if !context[word] {
			return false
		}
	}
	return true
}

func shorten(label string, context map[string]bool) string {
	parts := qualifier.Split(label, -1)
	separators := qualifier.FindAllString(label, -1)
	for len(parts) > 1 && contained(wordSet(parts[0]), context) {
		parts = parts[1:]
		separators = separators[1:]
	}
	for idx, part := range parts {
		tokens := strings.Fields(part)
		for len(tokens) > 1 && isAcronym(tokens[0]) && contained(wordSet(tokens[0]), context) {
			tokens = tokens[1:]
		}
		parts[idx] = strings.Join(tokens, " ")
	}
	label = parts[0]
	for idx, sep := range separators {
		label += sep + parts[idx+1]
	}
	return label
}

func isAcronym(token string) bool {
	found := false
	for _, r := range token {
		if unicode.IsLetter(r) {
			if !unicode.IsUpper(r) {
				return false
			}
			found = true
		} else if !unicode.IsDigit(r) {
			return false
		}
	}
	return found
}

func topLabel(title, directory string) string {
	// A repo-style heading such as "team-reference" is less useful as a
	// section label than its directory name, "Reference".
	if !strings.ContainsAny(title, " \t") && strings.HasSuffix(strings.ToLower(title), "-"+strings.ToLower(directory)) {
		return prettyDirectory(directory)
	}
	if loc := qualifier.FindStringIndex(title); loc != nil {
		return strings.TrimSpace(title[:loc[0]])
	}
	return title
}

func (i *Inventory) Nav() *Node {
	root := &Node{Name: i.Config.Name}
	dirs := map[string]*Node{"": root}
	for _, key := range i.Keys {
		parts := strings.Split(key, "/")
		parent := root
		directory := ""
		for _, part := range parts[:len(parts)-1] {
			directory = path.Join(directory, part)
			node, ok := dirs[directory]
			if !ok {
				node = &Node{dir: directory}
				if override, ok := i.Config.Navigation.Overrides[directory]; ok {
					node.Order = override.Order
					node.explicit = override.Title != "" || override.Order != 0
				}
				dirs[directory] = node
				if !i.Config.Navigation.Overrides[directory].Hide {
					parent.Children = append(parent.Children, node)
				}
			}
			parent = node
		}
		page := i.Pages[key]
		node := &Node{Name: cleanTitle(page.Title), Path: key, raw: cleanTitle(page.Title)}
		if strings.EqualFold(parts[len(parts)-1], "README.md") {
			node.Name = "Overview"
			if directory == "" {
				node.Name = "Home"
			}
		}
		if override, ok := i.Config.Navigation.Overrides[key]; ok {
			if override.Hide {
				continue
			}
			node.Order = override.Order
			if override.Title != "" {
				node.Name = override.Title
				node.explicit = true
			}
		}
		parent.Children = append(parent.Children, node)
	}
	rootReferences := map[string]int{}
	if readme, ok := i.Pages["README.md"]; ok {
		for idx, ref := range References(readme.Markdown) {
			if key, _, err := i.Resolve("README.md", ref.Target); err == nil {
				if _, exists := rootReferences[key]; !exists {
					rootReferences[key] = idx + 1
				}
			}
		}
	}
	var decorate func(*Node, map[string]bool)
	decorate = func(n *Node, context map[string]bool) {
		if n != root && n.Path == "" {
			base := path.Base(n.dir)
			raw := prettyDirectory(base)
			if readme, ok := i.Pages[path.Join(n.dir, "README.md")]; ok {
				raw = cleanTitle(readme.Title)
			}
			n.raw = raw
			title := raw
			if path.Dir(n.dir) == "." {
				title = topLabel(raw, base)
			} else {
				title = shorten(raw, context)
			}
			if override := i.Config.Navigation.Overrides[n.dir]; override.Title != "" {
				title = override.Title
			}
			n.Name = title
			context = addWords(context, raw)
		}
		for _, child := range n.Children {
			if child.Path == "" {
				decorate(child, context)
			} else if child.Path != "README.md" && !strings.HasSuffix(child.Path, "/README.md") && !child.explicit {
				child.Name = shorten(child.raw, context)
			}
		}
		sort.SliceStable(n.Children, func(a, b int) bool {
			x, y := n.Children[a], n.Children[b]
			if x.Order != y.Order {
				return x.Order < y.Order
			}
			if x.Path == "README.md" || strings.HasSuffix(x.Path, "/README.md") {
				return true
			}
			if y.Path == "README.md" || strings.HasSuffix(y.Path, "/README.md") {
				return false
			}
			if n == root && (x.Path != "") != (y.Path != "") {
				return x.Path != ""
			}
			if n == root && x.Path != "" && y.Path != "" {
				rx, okx := rootReferences[x.Path]
				ry, oky := rootReferences[y.Path]
				if okx != oky {
					return okx
				}
				if okx && rx != ry {
					return rx < ry
				}
			}
			if (x.Path != "") != (y.Path != "") {
				return x.Path != ""
			}
			left, right := x.Path, y.Path
			if left == "" {
				left = x.dir
			}
			if right == "" {
				right = y.dir
			}
			return strings.ToLower(left) < strings.ToLower(right)
		})
		for _, child := range n.Children {
			if child.Path != "" || child.explicit {
				continue
			}
			for len(child.Children) == 1 && child.Children[0].Path == "" && !child.Children[0].explicit {
				next := child.Children[0]
				child.Name += " / " + next.Name
				child.Children = next.Children
			}
		}
	}
	decorate(root, map[string]bool{})
	return root
}

func Sidebar(root *Node) string {
	var b strings.Builder
	var walk func(*Node, int)
	walk = func(n *Node, level int) {
		if n.Path != "" {
			b.WriteString(strings.Repeat("  ", level) + "- [" + strings.ReplaceAll(n.Name, "]", "\\]") + "](/" + n.Path + ")\n")
			return
		}
		if n != root {
			b.WriteString(strings.Repeat("  ", level) + "- " + n.Name + "\n")
			level++
		}
		for _, child := range n.Children {
			walk(child, level)
		}
	}
	walk(root, 0)
	return b.String()
}
