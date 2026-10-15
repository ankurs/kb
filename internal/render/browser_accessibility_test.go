package render

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ankurs/kb/internal/config"
)

func injectBrowserProbe(t *testing.T, page, probe string) string {
	t.Helper()
	at := strings.LastIndex(page, "</body>")
	if at < 0 {
		t.Fatal("viewer has no closing body tag")
	}
	return page[:at] + probe + page[at:]
}

func browserProbeResult(t *testing.T, dom, id string, result any) {
	t.Helper()
	pattern := regexp.MustCompile(`<output id="` + regexp.QuoteMeta(id) + `">([^<]*)</output>`)
	match := pattern.FindStringSubmatch(dom)
	if len(match) != 2 {
		t.Fatalf("%s did not run (%d DOM bytes)", id, len(dom))
	}
	if err := json.Unmarshal([]byte(html.UnescapeString(match[1])), result); err != nil {
		t.Fatalf("%s: %v", id, err)
	}
}

func TestBrowserMobileKeyboardNavigation(t *testing.T) {
	browser := headlessBrowser(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "chapter"), 0755); err != nil {
		t.Fatal(err)
	}
	for name, md := range map[string]string{
		"README.md":         "# Home\n\nWelcome to the guide.\n",
		"chapter/README.md": "# Chapter\n\nChapter body.\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(md), 0644); err != nil {
			t.Fatal(err)
		}
	}
	c := config.Workspace{Root: root, Name: "Mobile", Output: config.Output{Site: ".kb/site", Bundles: ".kb/snapshots"}}
	built, err := Build(c)
	if err != nil {
		t.Fatal(err)
	}
	probe := `<script>(function(){let attempts=0,timer=setInterval(function(){
  if(!window.kbViewerUI||!document.querySelector('#article h1,.markdown-section h1')){
    if(++attempts>200){clearInterval(timer);let out=document.createElement('output');out.id='mobile-probe';out.textContent='timeout';document.body.append(out)}
    return
  }
  clearInterval(timer);
  let sidebar=document.getElementById('kb-sidebar'),menu=document.getElementById('menu-toggle');
  let input=document.getElementById('filter');
  let result={width:innerWidth,mobile:matchMedia('(max-width:800px)').matches,
    menuVisible:getComputedStyle(menu).display!=='none',closedInert:sidebar.inert,
    collapsed:menu.getAttribute('aria-expanded')==='false'};
  input.focus();result.noOffscreenFocus=document.activeElement!==input;
  menu.focus();menu.click();
  result.opened=!sidebar.inert&&sidebar.classList.contains('open')&&
    menu.getAttribute('aria-expanded')==='true'&&document.activeElement===input;
  input.value='chapter';input.dispatchEvent(new Event('input',{bubbles:true}));
  result.searchAccessible=!!document.querySelector('#results [role="status"]')&&
    input.getAttribute('aria-controls')==='results';
  input.dispatchEvent(new KeyboardEvent('keydown',{key:'Escape',bubbles:true,cancelable:true}));
  result.escapeClosed=sidebar.inert&&!sidebar.classList.contains('open')&&
    menu.getAttribute('aria-expanded')==='false'&&document.activeElement===menu&&
    input.value===''&&document.getElementById('results').hidden;
  menu.click();
  document.querySelector('#nav a[data-path="chapter/README.md"]').click();
  setTimeout(function(){
    result.linkClosed=sidebar.inert&&!sidebar.classList.contains('open')&&
      menu.getAttribute('aria-expanded')==='false'&&document.activeElement===menu&&
      document.querySelector('#nav a[aria-current="page"]')?.dataset.path==='chapter/README.md'&&
      document.querySelector('#nav a[aria-current="page"]')?.closest('details')?.open;
    result.noHorizontalOverflow=document.documentElement.scrollWidth<=innerWidth+1;
    let out=document.createElement('output');out.id='mobile-probe';out.textContent=JSON.stringify(result);document.body.append(out)
  },200)
},20)})();</script>`
	staticFiles := http.FileServer(http.Dir(built.Site))
	static := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(injectBrowserProbe(t, LiveHTML(c, false), probe)))
			return
		}
		staticFiles.ServeHTTP(w, r)
	}))
	defer static.Close()
	served := Handler(c)
	live := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(injectBrowserProbe(t, LiveHTML(c, false), probe)))
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
	if err := os.WriteFile(bundlePath, []byte(injectBrowserProbe(t, string(bundle), probe)), 0644); err != nil {
		t.Fatal(err)
	}
	for _, viewer := range []struct{ name, url string }{
		{"bundle", "file://" + bundlePath + "#/README.md"},
		{"serve", live.URL + "/#/README.md"},
		{"site", static.URL + "/#/README.md"},
	} {
		t.Run(viewer.name, func(t *testing.T) {
			dom := dumpDOMWithSize(t, browser, viewer.url, 600, 850)
			var got struct {
				Width                                                                    int
				Mobile, MenuVisible, ClosedInert, Collapsed, NoOffscreenFocus            bool
				Opened, SearchAccessible, EscapeClosed, LinkClosed, NoHorizontalOverflow bool
			}
			browserProbeResult(t, dom, "mobile-probe", &got)
			if got.Width > 800 || !got.Mobile || !got.MenuVisible || !got.ClosedInert || !got.Collapsed || !got.NoOffscreenFocus ||
				!got.Opened || !got.SearchAccessible || !got.EscapeClosed || !got.LinkClosed || !got.NoHorizontalOverflow {
				t.Fatalf("mobile interaction: %+v", got)
			}
		})
	}
}

func TestBrowserLargeCollectionSearch(t *testing.T) {
	browser := headlessBrowser(t)
	pages := make(map[string]string, 1201)
	pages["README.md"] = "# Home\n\nIntroduction.\n"
	for n := 0; n < 1200; n++ {
		key := fmt.Sprintf("entries/note-%04d.md", n)
		pages[key] = fmt.Sprintf("# Note %04d\n\nRare catalog entry %04d.\n", n, n)
	}
	pages["entries/note-1199.md"] = "# Rare Catalog Guide\n\nOne rare catalog entry.\n"
	c := config.Workspace{Name: "Scale"}
	bundle, err := Bundle(BundleData{Title: "Scale", Pages: pages, Assets: map[string]string{}, Files: map[string]string{}, Nav: navFromPages(pages, c)})
	if err != nil {
		t.Fatal(err)
	}
	probe := `<script>(function(){let timer=setInterval(function(){if(!window.kbViewerUI)return;clearInterval(timer);
  let input=document.getElementById('filter');
  let start=performance.now();input.value='rare catalog';input.dispatchEvent(new Event('input',{bubbles:true}));
  let result={pages:Object.keys(JSON.parse(document.getElementById('page-data').textContent).pages).length,
    count:document.querySelectorAll('#results .result').length,
    first:document.querySelector('#results .result')?.dataset.path,
    status:document.querySelector('#results [role="status"]')?.textContent,
    elapsed:Math.round(performance.now()-start)};
  let out=document.createElement('output');out.id='scale-probe';out.textContent=JSON.stringify(result);document.body.append(out)
},20)})();</script>`
	path := filepath.Join(t.TempDir(), "scale.html")
	if err := os.WriteFile(path, []byte(injectBrowserProbe(t, string(bundle), probe)), 0644); err != nil {
		t.Fatal(err)
	}
	dom := dumpDOM(t, browser, "file://"+path)
	var got struct {
		Pages, Count, Elapsed int
		First, Status         string
	}
	browserProbeResult(t, dom, "scale-probe", &got)
	if got.Pages != 1201 || got.Count != 40 || got.First != "entries/note-1199.md" || got.Status != "Showing top 40 results" || got.Elapsed > 2000 {
		t.Fatalf("large collection search: %+v", got)
	}
}
