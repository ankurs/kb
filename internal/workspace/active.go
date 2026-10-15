package workspace

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ActiveLinkedFile identifies local files that could be interpreted as active
// browser content. SVG is checked separately by the renderer's SVG validator.
// Inspect the bytes as well as the extension so renaming an HTML document to
// .txt does not make it safe to copy into a static site.
func ActiveLinkedFile(path string) (bool, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".html", ".htm", ".xhtml", ".xml", ".xsl", ".xslt", ".mht", ".mhtml", ".js", ".mjs", ".css", ".wasm":
		return true, nil
	}
	if strings.EqualFold(filepath.Ext(path), ".svg") {
		return false, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	var header [512]byte
	n, err := io.ReadFull(f, header[:])
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return false, err
	}
	if strings.HasPrefix(http.DetectContentType(header[:n]), "text/html") {
		return true, nil
	}
	trimmed := strings.ToLower(strings.TrimSpace(string(header[:n])))
	return strings.HasPrefix(trimmed, "<?xml") || strings.HasPrefix(trimmed, "<svg") || strings.HasPrefix(trimmed, "<xsl:") || strings.HasPrefix(trimmed, "<script"), nil
}
