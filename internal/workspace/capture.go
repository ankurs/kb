package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ankurs/kb/internal/config"
)

func Capture(c config.Workspace, at, text string, now time.Time) error {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	if text == "" {
		return errors.New("capture text is empty")
	}
	if at != "" && !config.SafeRelative(at) {
		return errors.New("project must be a safe relative path")
	}
	file, err := c.WritablePath(c.Capture.Inbox)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		return err
	}
	line := "- " + now.Format("2006-01-02")
	if at != "" {
		line += " [" + at + "]"
	}
	line += " " + text + "\n"
	mode := os.FileMode(0644)
	if info, err := os.Lstat(file); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("inbox is not a regular file: %s", file)
		}
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return err
	}
	existing, err := os.ReadFile(file)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	content := string(existing)
	switch {
	case content == "":
		content = "# Inbox\n\n## Unprocessed\n\n" + line
	default:
		heading := regexp.MustCompile(`(?m)^## Unprocessed[^\n]*\n`)
		loc := heading.FindStringIndex(content)
		if loc == nil {
			content = strings.TrimRight(content, "\n") + "\n\n## Unprocessed\n\n" + line
		} else {
			next := regexp.MustCompile(`(?m)^## `).FindStringIndex(content[loc[1]:])
			end := len(content)
			if next != nil {
				end = loc[1] + next[0]
			}
			block := strings.TrimRight(content[loc[1]:end], "\n")
			if block == "" {
				block = "\n"
			} else {
				block += "\n"
			}
			content = content[:loc[1]] + block + line + "\n" + content[end:]
		}
	}
	tmp, err := os.CreateTemp(filepath.Dir(file), ".kb-inbox-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), file)
}

var badSlug = regexp.MustCompile(`[^a-z0-9-]+`)

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	return strings.Trim(badSlug.ReplaceAllString(s, "-"), "-")
}
func New(c config.Workspace, kind, at, title string, date time.Time) (string, error) {
	if kind != "meeting" && kind != "page" {
		return "", errors.New("kind must be meeting or page")
	}
	if !config.SafeRelative(at) || at == "." {
		return "", errors.New("--at must be a project directory within the workspace")
	}
	if strings.TrimSpace(title) == "" || strings.ContainsAny(title, "\r\n") {
		return "", errors.New("title must be a single nonempty line")
	}
	name := slug(title)
	if name == "" {
		return "", errors.New("title has no filename-safe characters")
	}
	template := "meeting.md"
	subdir := "meetings"
	filename := date.Format("2006-01-02") + "-" + name + ".md"
	header := "# " + date.Format("2006-01-02") + " — " + title
	if kind == "page" {
		template = "kb-page.md"
		subdir = "kb"
		filename = name + ".md"
		header = "# " + title
	}
	path, err := c.WritablePath(filepath.Join(at, subdir, filename))
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	source, _ := c.Path(filepath.Join(c.Capture.Templates, template))
	body, err := os.ReadFile(source)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if len(body) == 0 {
		body = []byte(header + "\n\n")
	}
	lines := strings.SplitN(string(body), "\n", 2)
	if len(lines) > 1 {
		body = []byte(header + "\n" + lines[1])
	} else {
		body = []byte(header + "\n\n")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("refusing to overwrite %s: %w", path, err)
	}
	defer f.Close()
	_, err = f.Write(body)
	return path, err
}
func Review(c config.Workspace) (string, error) {
	path, err := c.WritablePath(c.Capture.Inbox)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	parts := strings.SplitN(string(data), "## Unprocessed", 2)
	if len(parts) == 1 {
		return "Inbox has no Unprocessed section\n", nil
	}
	lines := []string{}
	for _, line := range strings.Split(parts[1], "\n") {
		if strings.HasPrefix(line, "## ") {
			break
		}
		if strings.HasPrefix(line, "- ") {
			lines = append(lines, line)
		}
	}
	return fmt.Sprintf("%d unprocessed item(s) in %s\n%s\n", len(lines), c.Capture.Inbox, strings.Join(lines, "\n")), nil
}
