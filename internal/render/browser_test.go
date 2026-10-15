package render

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ankurs/kb/internal/config"
)

func headlessBrowser(t *testing.T) string {
	t.Helper()
	if name := os.Getenv("KB_BROWSER"); name != "" {
		return name
	}
	// Hosted CI images may expose a browser executable that cannot start a
	// headless session. Require an explicitly provisioned browser in CI rather
	// than letting every browser test stall until its per-page timeout.
	if os.Getenv("CI") != "" {
		t.Skip("browser integration tests require an explicitly provisioned KB_BROWSER in CI")
	}
	for _, name := range []string{"chromium", "chromium-browser", "google-chrome"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	if runtime.GOOS == "darwin" {
		for _, path := range []string{"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser"} {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}
	t.Skip("no headless Chromium available (set KB_BROWSER to run browser integration test)")
	return ""
}

func TestBrowserViewerNavigationAndSearch(t *testing.T) {
	browser := headlessBrowser(t)
	root := t.TempDir()
	for name, content := range map[string]string{
		"README.md":       "# Home\n\nWelcome to the wiki.\n",
		"alpha/README.md": "# Clockwork Guide\n\nThe clockwork guide explains the mechanism.\n",
		"beta/README.md":  "# General Notes\n\nClockwork clockwork clockwork clockwork.\n",
	} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	c := config.Workspace{Root: root, Name: "Example", Output: config.Output{Site: ".kb/site", Bundles: ".kb/snapshots"}}
	build, err := Build(c)
	if err != nil {
		t.Fatal(err)
	}
	probe := `<script>(function(){let attempts=0,timer=setInterval(function(){
  if(!window.kbViewerUI||!document.querySelector('#article h1,.markdown-section h1')){
    if(++attempts>200){clearInterval(timer);let out=document.createElement('output');out.id='probe-result';out.textContent='timeout';document.body.append(out)}
    return
  }
  clearInterval(timer);
  let nav=document.getElementById('nav'),input=document.getElementById('filter');
  let active=nav.querySelector('[aria-current="page"]');
  let initiallyClosed=nav.querySelectorAll('details').length===2 &&
    nav.querySelectorAll('details[open]').length===1 &&
    !nav.querySelector('a[data-path="beta/README.md"]').closest('details').open;
  let section=active.closest('details');section.open=false;kbViewerUI.setActive('alpha/README.md');
  let expanded=section.open,highlightedActive=active.classList.contains('kb-active'),customNav=!nav.classList.contains('app-nav')&&
    getComputedStyle(section.querySelector('ul')).display!=='none';
  input.value='clockwork';input.dispatchEvent(new Event('input',{bubbles:true}));
  let first=document.querySelector('#results .result');
  let result={active:active.dataset.path,initiallyClosed:initiallyClosed,expanded:expanded,highlightedActive:highlightedActive,customNav:customNav,first:first?.dataset.path,
    count:document.querySelector('#results .result-count')?.textContent,
    highlighted:!!document.querySelector('#results .result mark'),hiddenNav:nav.hidden};
  input.value='clockwork guide';input.dispatchEvent(new Event('input',{bubbles:true}));
  result.allTerms=document.querySelectorAll('#results .result').length;
  document.dispatchEvent(new KeyboardEvent('keydown',{key:'k',ctrlKey:true,bubbles:true,cancelable:true}));
  result.shortcut=document.activeElement===input;
  input.dispatchEvent(new KeyboardEvent('keydown',{key:'Escape',bubbles:true}));
  result.restored=!nav.hidden&&document.getElementById('results').hidden;
  document.getElementById('kb-sidebar').classList.remove('open');
  document.getElementById('menu-toggle').setAttribute('aria-expanded','false');
  document.getElementById('menu-toggle').click();
  result.menu=document.getElementById('kb-sidebar').classList.contains('open');
  result.menuAttr=document.getElementById('menu-toggle').getAttribute('aria-expanded');
  result.sidebarClass=document.getElementById('kb-sidebar').className;
  input.value='clockwork';input.dispatchEvent(new Event('input',{bubbles:true}));
  document.querySelector('#results .result[data-path="beta/README.md"]').click();
  input.value='';input.dispatchEvent(new Event('input',{bubbles:true}));
  setTimeout(function(){
    let heading=document.querySelector('#article h1,.markdown-section h1');
    result.navigated=location.hash.includes('beta/README.md')&&
      nav.querySelector('[aria-current="page"]')?.dataset.path==='beta/README.md'&&
      nav.querySelector('[aria-current="page"]')?.classList.contains('kb-active')&&
      nav.querySelector('[aria-current="page"]')?.closest('details')?.open&&
      heading?.textContent.includes('General Notes');
    let out=document.createElement('output');out.id='probe-result';out.textContent=JSON.stringify(result);document.body.append(out)
  },200)
},20)})();</script>`
	injected := func(source string) string {
		at := strings.LastIndex(source, "</body>")
		if at < 0 {
			t.Fatal("viewer has no closing body tag")
		}
		return source[:at] + probe + source[at:]
	}
	staticFS := http.FileServer(http.Dir(build.Site))
	static := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(injected(LiveHTML(c, false))))
			return
		}
		staticFS.ServeHTTP(w, r)
	}))
	defer static.Close()
	served := Handler(c)
	live := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(injected(LiveHTML(c, false))))
			return
		}
		served.ServeHTTP(w, r)
	}))
	defer live.Close()
	bundle, err := os.ReadFile(filepath.Join(root, ".kb/snapshots/wiki.html"))
	if err != nil {
		t.Fatal(err)
	}
	probeBundle := filepath.Join(t.TempDir(), "wiki.html")
	if err := os.WriteFile(probeBundle, []byte(injected(string(bundle))), 0644); err != nil {
		t.Fatal(err)
	}
	for _, viewer := range []struct{ name, url string }{
		{"bundle", "file://" + probeBundle + "#/alpha/README.md"},
		{"serve", live.URL + "/#/alpha/README.md"},
		{"site", static.URL + "/#/alpha/README.md"},
	} {
		t.Run(viewer.name, func(t *testing.T) {
			dom := dumpDOM(t, browser, viewer.url)
			match := regexp.MustCompile(`<output id="probe-result">([^<]*)</output>`).FindStringSubmatch(dom)
			if len(match) != 2 {
				t.Fatalf("viewer probe did not run (%d bytes)", len(dom))
			}
			var got struct {
				Active, First, Count, MenuAttr, SidebarClass                                                                         string
				InitiallyClosed, Expanded, HighlightedActive, CustomNav, Highlighted, HiddenNav, Shortcut, Restored, Menu, Navigated bool
				AllTerms                                                                                                             int
			}
			if err := json.Unmarshal([]byte(html.UnescapeString(match[1])), &got); err != nil {
				t.Fatalf("probe %q: %v", match[1], err)
			}
			if got.Active != "alpha/README.md" || got.First != "alpha/README.md" || got.Count != "2 results" || got.AllTerms != 1 || !got.InitiallyClosed || !got.Expanded || !got.HighlightedActive || !got.CustomNav || !got.Highlighted || !got.HiddenNav || !got.Shortcut || !got.Restored || !got.Menu || !got.Navigated {
				t.Fatalf("viewer interaction: %+v", got)
			}
		})
	}
	focusedBody, err := os.ReadFile(filepath.Join(root, ".kb/snapshots/alpha.html"))
	if err != nil {
		t.Fatal(err)
	}
	focusedProbe := `<script>(function(){let timer=setInterval(function(){if(!window.kbViewerUI)return;clearInterval(timer);
  let input=document.getElementById('filter');input.value='clockwork';input.dispatchEvent(new Event('input',{bubbles:true}));
  let output=document.createElement('output');output.id='focused-result';
  output.textContent=document.querySelectorAll('#results .result').length+':'+
    (document.querySelector('#results .result')?.dataset.path||'');document.body.append(output)
},20)})();</script>`
	at := strings.LastIndex(string(focusedBody), "</body>")
	if at < 0 {
		t.Fatal("focused bundle lacks body")
	}
	focusedFile := filepath.Join(t.TempDir(), "alpha.html")
	if err := os.WriteFile(focusedFile, []byte(string(focusedBody[:at])+focusedProbe+string(focusedBody[at:])), 0644); err != nil {
		t.Fatal(err)
	}
	focusedDOM := dumpDOM(t, browser, "file://"+focusedFile+"#/alpha/README.md")
	if !strings.Contains(focusedDOM, `<output id="focused-result">1:alpha/README.md</output>`) {
		t.Fatal("focused bundle search included another folder")
	}
}

func TestBrowserExternalLinksPreference(t *testing.T) {
	browser := headlessBrowser(t)
	for _, newTab := range []bool{true, false} {
		name := "new-tab"
		if !newTab {
			name = "same-tab"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			md := "# Home\n\n[External](https://example.org/path)\n\n" +
				"[Protocol](//example.net/path)\n\n" +
				"[Internal](other/README.md)\n\n" +
				"[Mail](mailto:friend@example.org)\n\n" +
				`<a href="https://example.com/raw">Raw external</a>` + "\n"
			if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(md), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(root, "other"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "other", "README.md"), []byte("# Other\n"), 0644); err != nil {
				t.Fatal(err)
			}
			c := config.Workspace{Root: root, Name: "Links", Output: config.Output{Site: ".kb/site", Bundles: ".kb/snapshots"}}
			if !newTab {
				c.Render.ExternalLinksNewTab = &newTab
			}
			build, err := Build(c)
			if err != nil {
				t.Fatal(err)
			}
			probe := `<script>(function(){let attempts=0,timer=setInterval(function(){
  let article=document.querySelector('#article,.markdown-section');
  if(!article||!article.querySelector('a[href="https://example.org/path"]')){
    if(++attempts>200){clearInterval(timer);let out=document.createElement('output');out.id='links-result';out.textContent='timeout';document.body.append(out)}
    return
  }
  clearInterval(timer);
  let labels=['External','Protocol','Raw external','Internal','Mail'];
  let result={};labels.forEach(label=>{
    let a=Array.from(article.querySelectorAll('a')).find(x=>x.textContent.trim()===label);
    result[label]=a?{target:a.getAttribute('target')||'',rel:a.getAttribute('rel')||'',href:a.getAttribute('href')||''}:null
  });
  let out=document.createElement('output');out.id='links-result';out.textContent=JSON.stringify(result);document.body.append(out)
},20)})();</script>`
			inject := func(content string) string {
				at := strings.LastIndex(content, "</body>")
				if at < 0 {
					t.Fatal("viewer has no closing body tag")
				}
				return content[:at] + probe + content[at:]
			}
			staticFiles := http.FileServer(http.Dir(build.Site))
			static := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/" || r.URL.Path == "/index.html" {
					w.Header().Set("Content-Type", "text/html")
					w.Write([]byte(inject(LiveHTML(c, false))))
					return
				}
				staticFiles.ServeHTTP(w, r)
			}))
			defer static.Close()
			served := Handler(c)
			live := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/" || r.URL.Path == "/index.html" {
					w.Header().Set("Content-Type", "text/html")
					w.Write([]byte(inject(LiveHTML(c, false))))
					return
				}
				served.ServeHTTP(w, r)
			}))
			defer live.Close()
			bundle, err := os.ReadFile(filepath.Join(root, ".kb", "snapshots", "wiki.html"))
			if err != nil {
				t.Fatal(err)
			}
			bundlePath := filepath.Join(t.TempDir(), "wiki.html")
			if err := os.WriteFile(bundlePath, []byte(inject(string(bundle))), 0644); err != nil {
				t.Fatal(err)
			}
			for _, viewer := range []struct{ name, url string }{
				{"bundle", "file://" + bundlePath + "#/README.md"},
				{"serve", live.URL + "/#/README.md"},
				{"site", static.URL + "/#/README.md"},
			} {
				t.Run(viewer.name, func(t *testing.T) {
					dom := dumpDOM(t, browser, viewer.url)
					match := regexp.MustCompile(`<output id="links-result">([^<]*)</output>`).FindStringSubmatch(dom)
					if len(match) != 2 {
						t.Fatalf("link probe did not run (%d bytes)", len(dom))
					}
					var got map[string]*struct{ Target, Rel, Href string }
					if err := json.Unmarshal([]byte(html.UnescapeString(match[1])), &got); err != nil {
						t.Fatal(err)
					}
					for _, label := range []string{"External", "Protocol", "Raw external"} {
						a := got[label]
						if a == nil {
							t.Fatalf("%s missing: %+v", label, got)
						}
						if newTab {
							if a.Target != "_blank" || !strings.Contains(" "+a.Rel+" ", " noopener ") || !strings.Contains(" "+a.Rel+" ", " noreferrer ") {
								t.Fatalf("%s should open safely in a new tab: %+v", label, *a)
							}
						} else if a.Target != "" {
							t.Fatalf("%s should open in the same tab: %+v", label, *a)
						}
					}
					if got["Internal"] == nil || got["Internal"].Target != "" || got["Mail"] == nil {
						t.Fatalf("internal or mail link modified: %+v", got)
					}
				})
			}
		})
	}
}

func TestBrowserMountAndFallbackSearch(t *testing.T) {
	browser := headlessBrowser(t)
	c := testWorkspace(t)
	built, err := Build(c)
	if err != nil {
		t.Fatal(err)
	}
	static := httptest.NewServer(http.FileServer(http.Dir(built.Site)))
	defer static.Close()
	server := Handler(c)
	live := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(LiveHTML(c, false)))
			return
		}
		server.ServeHTTP(w, r)
	}))
	defer live.Close()
	for _, viewer := range []struct{ name, url string }{
		{"bundle", "file://" + filepath.Join(c.Root, ".kb/snapshots/wiki.html") + "#/handbook/check-in/README.md"},
		{"serve", live.URL + "/#/handbook/check-in/README.md"},
		{"site", static.URL + "/#/handbook/check-in/README.md"},
	} {
		t.Run(viewer.name, func(t *testing.T) {
			dom := dumpDOM(t, browser, viewer.url)
			if !regexp.MustCompile(`data-path="handbook/check-in/README.md"[^>]*aria-current="page"`).MatchString(dom) ||
				!strings.Contains(dom, "Check-in Checklist") {
				t.Fatal("mounted route or active navigation did not render")
			}
		})
	}
	if err := os.RemoveAll(filepath.Join(c.Root, "reference")); err != nil {
		t.Fatal(err)
	}
	c.Publish.Include = []string{"handbook"}
	published, err := Publish(c)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(published.Dir, "handbook.html"))
	if err != nil {
		t.Fatal(err)
	}
	probe := `<script>(function(){let timer=setInterval(function(){if(!window.kbViewerUI)return;clearInterval(timer);
  let input=document.getElementById('filter');input.value='inspect';input.dispatchEvent(new Event('input',{bubbles:true}));
  let output=document.createElement('output');output.id='fallback-result';output.textContent=
    Array.from(document.querySelectorAll('#results .result')).some(a=>a.dataset.path==='handbook/check-in/README.md');
  document.body.append(output)
},20)})();</script>`
	at := strings.LastIndex(string(body), "</body>")
	if at < 0 {
		t.Fatal("fallback bundle lacks body")
	}
	fallbackFile := filepath.Join(t.TempDir(), "handbook.html")
	if err := os.WriteFile(fallbackFile, []byte(string(body[:at])+probe+string(body[at:])), 0644); err != nil {
		t.Fatal(err)
	}
	dom := dumpDOM(t, browser, "file://"+fallbackFile+"#/handbook/check-in/README.md")
	if !strings.Contains(dom, `<output id="fallback-result">true</output>`) {
		t.Fatal("fallback pages were not searchable in focused bundle")
	}
}

func dumpDOM(t *testing.T, browser, url string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	args := []string{"--headless", "--disable-gpu", "--no-sandbox", "--disable-background-networking", "--disable-extensions", "--no-first-run", "--no-default-browser-check", "--user-data-dir=" + t.TempDir(), "--host-resolver-rules=MAP * ~NOTFOUND, EXCLUDE 127.0.0.1", "--virtual-time-budget=5000", "--dump-dom", url}
	cmd := exec.CommandContext(ctx, browser, args...)
	b, err := cmd.Output()
	if ctx.Err() != nil {
		t.Fatalf("browser timed out at %s", url)
	}
	if err != nil && !bytes.Contains(b, []byte("<!DOCTYPE html>")) {
		t.Fatalf("browser %s: %v (%d bytes)", url, err, len(b))
	}
	return string(b)
}

func TestBrowserRenderingOffline(t *testing.T) {
	browser := headlessBrowser(t)
	for _, trusted := range []bool{false, true} {
		name := "sanitized"
		if trusted {
			name = "trusted"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			md := "# Home\n\n<br><table><tr><td>benign</td></tr></table>\n\n" +
				`<img src="missing" onerror="document.body.setAttribute('data-executed','1')">` + "\n" +
				`<script id="attack">document.body.setAttribute('data-executed','1')</script>` + "\n\n" +
				"[bad](javascript:alert(1))\n\n![photo](pic.png)\n\n```js\nconst answer = 42;\n```\n\n```mermaid\ngraph LR\n A-->B\n```\n"
			if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(md), 0644); err != nil {
				t.Fatal(err)
			}
			png, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/lF0AAAAASUVORK5CYII=")
			os.WriteFile(filepath.Join(root, "pic.png"), png, 0644)
			c := config.Workspace{Root: root, Name: "Browser Test", Output: config.Output{Site: ".kb/site", Bundles: ".kb/snapshots"}, Render: config.Render{TrustHTML: trusted}}
			result, err := Build(c)
			if err != nil {
				t.Fatal(err)
			}
			bundle := filepath.Join(root, ".kb/snapshots/wiki.html")
			served := Handler(c)
			live := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/" {
					w.Header().Set("Content-Type", "text/html")
					w.Write([]byte(LiveHTML(c, false)))
					return
				}
				served.ServeHTTP(w, r)
			}))
			defer live.Close()
			static := httptest.NewServer(http.FileServer(http.Dir(result.Site)))
			defer static.Close()
			for _, viewer := range []struct{ name, url string }{
				{"bundle", "file://" + bundle}, {"serve", live.URL + "/#/README.md"}, {"site", static.URL + "/#/README.md"},
			} {
				t.Run(viewer.name, func(t *testing.T) {
					dom := dumpDOM(t, browser, viewer.url)
					if !strings.Contains(dom, "benign</td>") || !strings.Contains(dom, "<br>") || !strings.Contains(dom, "answer") || !strings.Contains(dom, "mermaid") || !strings.Contains(dom, "pic.png") && !strings.Contains(dom, "data:image/png") {
						t.Fatalf("missing benign render in %s (DOM %d bytes): %.1000s", viewer.name, len(dom), dom)
					}
					if !strings.Contains(dom, `data-processed="true"`) || !strings.Contains(dom, "<svg") {
						t.Fatal("Mermaid diagram did not render")
					}
					if viewer.name == "bundle" && !strings.Contains(dom, `<img src="data:image/png`) {
						t.Fatal("bundle image did not resolve to embedded data")
					}
					if trusted {
						if !strings.Contains(dom, `onerror="document.body.setAttribute`) {
							t.Fatal("trusted raw HTML removed")
						}
					} else if strings.Contains(dom, `onerror="document.body.setAttribute`) || strings.Contains(dom, `<script id="attack"`) || strings.Contains(dom, `data-executed="1"`) || strings.Contains(dom, `href="javascript:`) {
						t.Fatal("unsanitized executable Markdown in DOM")
					}
				})
			}
		})
	}
}
