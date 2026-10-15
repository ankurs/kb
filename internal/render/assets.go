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
	names := []string{}
	for _, m := range c.Mounts {
		names = append(names, m.Name)
	}
	mounts, _ := json.Marshal(names)
	viewerName, _ := json.Marshal(c.Name) // JSON escapes </script in inline JavaScript.
	reload := ""
	if live {
		reload = `<script>let source=new EventSource('/_kb/events');source.onmessage=()=>location.reload()</script>`
	}
	return fmt.Sprintf(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s</title><link rel="stylesheet" href="/vendor/docsify/dark-4.13.1.min.css"><link rel="stylesheet" href="/_kb/live.css"></head><body><div id="app"></div><script src="/vendor/bundle/purify-3.2.6.min.js"></script><script src="/vendor/mermaid/mermaid-11.17.2.min.js"></script><script>const mounts=%s,trustHTML=%t;mermaid.initialize({startOnLoad:false,theme:'dark',securityLevel:'strict'});window.$docsify={name:%s,homepage:'README.md',loadSidebar:true,alias:{'/.*/_sidebar.md':'/_sidebar.md'},sidebarDisplayLevel:1,relativePath:true,subMaxLevel:3,search:{placeholder:'Search',depth:3},markdown:{renderer:{code:function(code,lang){if(lang==='mermaid'){let safe=trustHTML?code:code.replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));return '<div class="mermaid">'+safe+'</div>'}return this.origin.code.apply(this,arguments)}}},plugins:[function(hook,vm){hook.beforeEach(function(md){let mount=mounts.find(m=>vm.route.path.startsWith('/'+m+'/'));if(!mount)return md;let fence=false;return md.split('\n').map(line=>{let t=line.trim();if(t.startsWith('~~~')||t.charCodeAt(0)===96)fence=!fence;if(fence)return line;return line.replace(/\]\(\/([^)]+)/g,(all,dest)=>dest.startsWith(mount+'/')?all:'](/'+mount+'/'+dest)}).join('\n')});hook.afterEach(function(html){return trustHTML?html:DOMPurify.sanitize(html)});hook.doneEach(function(){mermaid.run({querySelector:'.mermaid'}).catch(console.error)})}]};</script><script src="/vendor/docsify/docsify-4.13.1.min.js"></script><script src="/vendor/docsify/search-4.13.1.min.js"></script><script src="/vendor/docsify/sidebar-collapse-1.3.5.min.js"></script><script src="/vendor/prism/prism-python-1.30.0.min.js"></script><script src="/vendor/prism/prism-bash-1.30.0.min.js"></script><script src="/vendor/prism/prism-json-1.30.0.min.js"></script><script src="/vendor/prism/prism-yaml-1.30.0.min.js"></script>%s</body></html>`, html.EscapeString(c.Name), mounts, c.Render.TrustHTML, viewerName, reload)
}
