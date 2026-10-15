package render

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ankurs/kb/internal/workspace"
)

func TestServeSearchFiltersQMDHits(t *testing.T) {
	c := testWorkspace(t)
	// An indexed file may no longer be visible, even though qmd still returns it.
	hidden := filepath.Join(c.Root, ".private.md")
	if err := os.WriteFile(hidden, []byte("# Secret"), 0644); err != nil {
		t.Fatal(err)
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/query" || r.Method != "POST" {
			t.Errorf("qmd request: %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			Searches []struct{ Type, Query string } `json:"searches"`
			Rerank   bool                           `json:"rerank"`
			Limit    int                            `json:"limit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body.Searches) != 2 || body.Searches[0].Type != "lex" || body.Searches[1].Type != "vec" || body.Searches[0].Query != "borrowed tools" || body.Rerank || body.Limit != 40 {
			t.Errorf("qmd request: %+v", body)
		}
		json.NewEncoder(w).Encode(map[string]any{"results": []qmdResult{
			{File: "qmd://wiki/tool-library/kb/borrowing-policy.md", Snippet: "12: return a borrowed tool"},
			{File: "qmd://handbook/check-in/README.md", Snippet: "7: check-in"},
			{File: "qmd://wiki/.private.md", Snippet: "secret"},
			{File: "qmd://wiki/../../outside.md", Snippet: "outside"},
			{File: "qmd://unknown/tool-library/kb/borrowing-policy.md", Snippet: "wrong collection"},
			{File: "qmd://wiki/tool-library/kb/borrowing-policy.md", Snippet: "duplicate"},
		}})
	}))
	defer upstream.Close()
	backend := &qmdSearch{endpoint: upstream.URL, client: upstream.Client(), collections: map[string]string{"wiki": ".", "handbook": "reference"}}
	h := handler(c, backend)
	rec := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/_kb/search?q=borrowed+tools", nil)
	request.Host = "127.0.0.1:3000"
	h.ServeHTTP(rec, request)
	if rec.Code != 200 || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("search status: %d: %s", rec.Code, rec.Body.String())
	}
	var results struct {
		Results []searchHit `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatal(err)
	}
	if len(results.Results) != 2 || results.Results[0].Path != "tool-library/kb/borrowing-policy.md" || results.Results[1].Path != "handbook/check-in/README.md" {
		t.Fatalf("results: %+v", results)
	}
	// qmd commonly configures collections with absolute workspace paths.
	absolute := &qmdSearch{endpoint: upstream.URL, client: upstream.Client(), collections: map[string]string{
		"wiki": c.Root, "handbook": filepath.Join(c.Root, "reference"),
	}}
	inv, err := workspace.Load(c)
	if err != nil {
		t.Fatal(err)
	}
	absoluteHits, err := absolute.query(httptest.NewRequest("GET", "/", nil).Context(), "borrowed tools", inv)
	if err != nil || len(absoluteHits) != 2 || absoluteHits[0].Path != results.Results[0].Path || absoluteHits[1].Path != results.Results[1].Path {
		t.Fatalf("absolute collection paths: %+v %v", absoluteHits, err)
	}
	for _, request := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/_kb/search?q=x", 400}, {"GET", "/_kb/search?q=" + strings.Repeat("x", 501), 400}, {"POST", "/_kb/search?q=test", 405},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(request.method, request.path, nil)
		req.Host = "127.0.0.1:3000"
		h.ServeHTTP(rec, req)
		if rec.Code != request.status {
			t.Errorf("%s %s: %d", request.method, request.path, rec.Code)
		}
	}
	for _, host := range []string{"attacker.example:3000", "127.0.0.1:3000"} {
		req := httptest.NewRequest("GET", "/_kb/search?q=borrowed+tools", nil)
		req.Host = host
		if host == "127.0.0.1:3000" {
			req.Header.Set("Origin", "https://attacker.example")
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != 403 {
			t.Errorf("host %s origin %s: %d", host, req.Header.Get("Origin"), rec.Code)
		}
	}
	rec = httptest.NewRecorder()
	Handler(c).ServeHTTP(rec, httptest.NewRequest("GET", "/_kb/search?q=test", nil))
	if rec.Code != 404 {
		t.Fatalf("disabled search: %d", rec.Code)
	}
	page := httptest.NewRecorder()
	h.ServeHTTP(page, httptest.NewRequest("GET", "/", nil))
	if !strings.Contains(page.Body.String(), `"semanticSearch":true`) {
		t.Fatal("live viewer not enabled")
	}
	if strings.Contains(LiveHTML(c, false), `"semanticSearch":true`) {
		t.Fatal("static viewer must remain offline")
	}
}

func TestServeSearchUpstreamFailure(t *testing.T) {
	c := testWorkspace(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "private upstream error", 500) }))
	defer upstream.Close()
	h := handler(c, &qmdSearch{endpoint: upstream.URL, client: upstream.Client()})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_kb/search?q=borrowed", nil)
	req.Host = "127.0.0.1:3000"
	h.ServeHTTP(rec, req)
	if rec.Code != 502 || strings.Contains(rec.Body.String(), "private upstream error") {
		t.Fatalf("response: %d %s", rec.Code, rec.Body.String())
	}
}

func TestServeCollectionsRequiresIndex(t *testing.T) {
	c := testWorkspace(t)
	q := workspace.QMD{Root: c.Root, ConfigDir: c.Search.QMDConfig}
	if _, err := q.ServeCollections(); err == nil || !strings.Contains(err.Error(), "qmd init") {
		t.Fatalf("uninitialized qmd: %v", err)
	}
}

func TestBrowserLiveQMDSearchAndFallback(t *testing.T) {
	browser := headlessBrowser(t)
	c := testWorkspace(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/query" {
			var request struct {
				Searches []struct {
					Query string `json:"query"`
				} `json:"searches"`
			}
			json.NewDecoder(r.Body).Decode(&request)
			if len(request.Searches) > 0 && request.Searches[0].Query == "borrowing policy" {
				http.Error(w, "qmd failed", 500)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"results": []qmdResult{{File: "qmd://wiki/tool-library/kb/borrowing-policy.md", Snippet: "9: semantic evidence"}}})
			return
		}
		http.NotFound(w, r)
	}))
	defer upstream.Close()
	backend := &qmdSearch{endpoint: upstream.URL, client: upstream.Client(), collections: map[string]string{"wiki": "."}}
	probe := `<script>(function(){let tries=0,timer=setInterval(function(){
  if(!window.kbViewerUI){if(++tries>300)clearInterval(timer);return}
  clearInterval(timer);
  let input=document.getElementById('filter');input.value='conceptual borrowing';input.dispatchEvent(new Event('input',{bubbles:true}));
  setTimeout(function(){
   let first=document.querySelector('#results .result');
   let result={path:first?.dataset.path,status:document.querySelector('#results .result-count')?.textContent,
    snippet:first?.querySelector('.result-snippet')?.textContent};
   input.value='borrowing policy';input.dispatchEvent(new Event('input',{bubbles:true}));
   setTimeout(function(){
    result.fallback=document.querySelector('#results .result-count')?.textContent;
    result.fallbackPath=document.querySelector('#results .result')?.dataset.path;
    let out=document.createElement('output');out.id='qmd-probe';out.textContent=JSON.stringify(result);document.body.append(out)
   },850)
  },1500)
 },20)})();</script>`
	h := handler(c, backend)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(injectBrowserProbe(t, liveHTML(c, false, true), probe)))
			return
		}
		h.ServeHTTP(w, r)
	}))
	defer server.Close()
	dom := dumpDOM(t, browser, server.URL+"/#/README.md")
	var got struct{ Path, Status, Snippet, Fallback, FallbackPath string }
	browserProbeResult(t, dom, "qmd-probe", &got)
	if got.Path != "tool-library/kb/borrowing-policy.md" || got.Status != "1 result from qmd" || !strings.Contains(got.Snippet, "semantic evidence") || got.Fallback == "" || strings.Contains(got.Fallback, "from qmd") || got.FallbackPath == "" {
		t.Fatalf("qmd browser result: %+v", got)
	}
}

func TestServeSearchRejectsLinkedQMDOutsideWorkspace(t *testing.T) {
	c := testWorkspace(t)
	dir := filepath.Join(c.Root, c.Search.QMDConfig)
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "index.sqlite"), []byte("not an index"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "index.sqlite"), filepath.Join(dir, "index.sqlite")); err != nil {
		t.Fatal(err)
	}
	q := workspace.QMD{Root: c.Root, ConfigDir: c.Search.QMDConfig}
	if _, err := q.ServeCollections(); err == nil || !strings.Contains(err.Error(), "escapes content root") {
		t.Fatalf("external index: %v", err)
	}
	if err := os.Remove(filepath.Join(dir, "index.sqlite")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.sqlite"), []byte("placeholder"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "index.yml")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "index.yml"), []byte("collections:\n  private:\n    path: .\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "index.yml"), filepath.Join(dir, "index.yml")); err != nil {
		t.Fatal(err)
	}
	if _, err := q.ServeCollections(); err == nil || !strings.Contains(err.Error(), "escapes content root") {
		t.Fatalf("external config: %v", err)
	}
}

func TestQMDShortQueryUsesLexicalSearch(t *testing.T) {
	c := testWorkspace(t)
	inv, err := workspace.Load(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		query string
		types []string
	}{
		{"policy", []string{"lex"}},
		{"반납", []string{"lex", "vec"}},
	} {
		t.Run(tc.query, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					Searches []struct {
						Type string `json:"type"`
					} `json:"searches"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
				}
				if len(request.Searches) != len(tc.types) {
					t.Errorf("query types: %+v", request)
					return
				}
				for i, search := range request.Searches {
					if search.Type != tc.types[i] {
						t.Errorf("query types: %+v", request)
					}
				}
				json.NewEncoder(w).Encode(map[string]any{"results": []qmdResult{{File: "qmd://wiki/tool-library/kb/borrowing-policy.md", Snippet: "policy"}}})
			}))
			defer upstream.Close()
			backend := &qmdSearch{endpoint: upstream.URL, client: upstream.Client(), collections: map[string]string{"wiki": c.Root}}
			hits, err := backend.query(httptest.NewRequest("GET", "/", nil).Context(), tc.query, inv)
			if err != nil || len(hits) != 1 {
				t.Fatalf("hits: %+v %v", hits, err)
			}
		})
	}
}
