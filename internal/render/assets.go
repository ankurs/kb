package render

import (
	"embed"
	"encoding/json"
	"fmt"
	"html"
	"io/fs"
	"strings"

	"github.com/ankurs/kb/assets"
	"github.com/ankurs/kb/internal/config"
)

//go:embed web/*
var web embed.FS

func asset(name string) ([]byte, error) { return fs.ReadFile(assets.FS, "vendor/"+name) }
func webFile(name string) string        { b, _ := web.ReadFile("web/" + name); return string(b) }
func script(b []byte) string            { return strings.ReplaceAll(string(b), "</script", "<\\/script") }
func LiveHTML(c config.Workspace, live bool) string {
	mounts := []string{}
	for _, m := range c.Mounts {
		mounts = append(mounts, m.Name)
	}
	settings, _ := json.Marshal(struct {
		Name                string   `json:"name"`
		Mounts              []string `json:"mounts"`
		TrustHTML           bool     `json:"trustHTML"`
		ExternalLinksNewTab bool     `json:"externalLinksNewTab"`
	}{c.Name, mounts, c.Render.TrustHTML, c.Render.OpenExternalLinksNewTab()})
	reload := ""
	if live {
		reload = `<script>let source=new EventSource('_kb/events');source.onmessage=()=>location.reload()</script>`
	}
	return fmt.Sprintf(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s</title><link rel="stylesheet" href="vendor/docsify/dark-4.13.1.min.css"><link rel="stylesheet" href="_kb/live.css"><link rel="stylesheet" href="_kb/ui.css"></head><body class="kb-viewer kb-live"><header id="topbar"><button id="menu-toggle" type="button" aria-label="Toggle sidebar" aria-controls="kb-sidebar" aria-expanded="false">&#9776;</button><span class="brand">%s</span><span class="spacer"></span><span class="tag">local wiki</span></header><aside id="kb-sidebar"><input id="filter" type="search" placeholder="Search pages and content…" aria-label="Search pages and content" aria-controls="results"><div id="results" aria-live="polite" hidden></div><div id="nav" role="navigation" aria-label="Pages"></div></aside><div id="app"></div><script type="application/json" id="viewer-config">%s</script><script src="vendor/bundle/purify-3.2.6.min.js"></script><script src="vendor/mermaid/mermaid-11.17.2.min.js"></script><script src="_kb/ui.js"></script><script src="_kb/live.js"></script><script src="vendor/docsify/docsify-4.13.1.min.js"></script><script src="vendor/prism/prism-python-1.30.0.min.js"></script><script src="vendor/prism/prism-bash-1.30.0.min.js"></script><script src="vendor/prism/prism-json-1.30.0.min.js"></script><script src="vendor/prism/prism-yaml-1.30.0.min.js"></script>%s</body></html>`, html.EscapeString(c.Name), html.EscapeString(c.Name), settings, reload)
}
