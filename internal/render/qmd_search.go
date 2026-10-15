package render

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ankurs/kb/internal/config"
	"github.com/ankurs/kb/internal/workspace"
)

type qmdResult struct {
	File    string `json:"file"`
	Snippet string `json:"snippet"`
}
type searchHit struct {
	Path    string `json:"path"`
	Snippet string `json:"snippet"`
}
type qmdSearch struct {
	endpoint    string
	client      *http.Client
	collections map[string]string
}

// startQMD owns a private loopback HTTP process, never a shared daemon. The
// browser talks only to kb; the qmd port is not published or configurable.
func startQMD(ctx context.Context, c config.Workspace) (*qmdSearch, func(), error) {
	if c.Search.Provider != "qmd" {
		return nil, nil, errors.New("qmd search is not configured")
	}
	q := workspace.QMD{Root: c.Root, ConfigDir: c.Search.QMDConfig}
	collections, err := q.ServeCollections()
	if err != nil {
		return nil, nil, err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	childCtx, cancel := context.WithCancel(ctx)
	args := []string{"mcp", "--http", "--host", "127.0.0.1", "--port", fmt.Sprint(port)}
	if c.Search.QMDConfig != "" && c.Search.QMDConfig != ".qmd" {
		// Custom config directories are not discovered as project-local by
		// qmd. Use that directory and its paired SQLite index explicitly.
		args = append([]string{"--index", "index"}, args...)
	}
	cmd := exec.CommandContext(childCtx, "qmd", args...)
	cmd.Dir = c.Root
	cmd.Env = q.CommandEnv()
	// A file-backed stderr avoids a pipe held open by qmd's model workers
	// after the parent process exits (which would otherwise hang cmd.Wait).
	cmd.Stderr = os.Stderr
	cmd.WaitDelay = 2 * time.Second
	if err = cmd.Start(); err != nil {
		cancel()
		return nil, nil, err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	stop := func() { cancel(); <-done }
	backend := &qmdSearch{endpoint: fmt.Sprintf("http://127.0.0.1:%d", port), client: &http.Client{Timeout: 45 * time.Second}, collections: collections}
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(10 * time.Second)
	defer timeout.Stop()
	for {
		req, _ := http.NewRequestWithContext(ctx, "GET", backend.endpoint+"/health", nil)
		probe := &http.Client{Timeout: 200 * time.Millisecond}
		if resp, e := probe.Do(req); e == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return backend, stop, nil
			}
		}
		select {
		case err := <-done:
			cancel()
			return nil, nil, fmt.Errorf("qmd mcp --http exited: %w", err)
		case <-timeout.C:
			cancel()
			<-done
			return nil, nil, errors.New("qmd mcp --http did not start within 10 seconds")
		case <-ctx.Done():
			cancel()
			<-done
			return nil, nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *qmdSearch) query(ctx context.Context, text string, inv *workspace.Inventory) ([]searchHit, error) {
	searches := []map[string]string{{"type": "lex", "query": text}}
	// Short ASCII terms should not load an embedding model on every keystroke.
	// Keep semantic retrieval for multiword and non-Latin queries.
	if len(strings.Fields(text)) > 1 || strings.IndexFunc(text, func(r rune) bool { return r > 127 }) >= 0 {
		searches = append(searches, map[string]string{"type": "vec", "query": text})
	}
	body, _ := json.Marshal(map[string]any{"searches": searches, "rerank": false, "limit": 40})
	req, err := http.NewRequestWithContext(ctx, "POST", s.endpoint+"/query", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("qmd query returned HTTP %d", resp.StatusCode)
	}
	var output struct {
		Results []qmdResult `json:"results"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(nil, resp.Body, 1<<20)).Decode(&output); err != nil {
		return nil, err
	}
	// Only indexed documents that are also inventoried pages can appear in the
	// viewer. This excludes old, hidden, symlinked, or out-of-workspace hits.
	files := make(map[string]string, len(inv.Pages))
	for key, page := range inv.Pages {
		if resolved, err := filepath.EvalSymlinks(page.File); err == nil {
			files[resolved] = key
		}
	}
	hits := make([]searchHit, 0, len(output.Results))
	seen := map[string]bool{}
	for _, result := range output.Results {
		u, err := url.Parse(result.File)
		if err != nil || u.Scheme != "qmd" || u.Host == "" || u.RawQuery != "" || u.Fragment != "" {
			continue
		}
		base, ok := s.collections[u.Host]
		if !ok {
			continue
		}
		relative := strings.TrimPrefix(u.Path, "/")
		if !config.SafeRelative(relative) || relative == "." || strings.Contains(relative, "\\") {
			continue
		}
		if !filepath.IsAbs(base) {
			base = filepath.Join(inv.Config.Root, base)
		}
		source, err := filepath.EvalSymlinks(filepath.Join(base, filepath.FromSlash(relative)))
		if err != nil {
			continue
		}
		key, ok := files[source]
		if !ok || seen[key] {
			continue
		}
		seen[key] = true
		hits = append(hits, searchHit{Path: key, Snippet: result.Snippet})
	}
	return hits, nil
}

func serveSearch(w http.ResponseWriter, r *http.Request, c config.Workspace, s *qmdSearch) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if s == nil {
		http.NotFound(w, r)
		return
	}
	// Do not turn the local qmd proxy into a DNS-rebinding or cross-origin
	// endpoint. A deliberately non-loopback kb bind still needs access controls.
	if !allowedSearchRequest(r, c) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) < 2 || len(query) > 500 {
		http.Error(w, "query must be 2–500 bytes", http.StatusBadRequest)
		return
	}
	inv, err := workspace.Load(c)
	if err != nil {
		http.Error(w, "viewer unavailable", http.StatusInternalServerError)
		return
	}
	hits, err := s.query(r.Context(), query, inv)
	if err != nil {
		http.Error(w, "qmd search unavailable", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(struct {
		Results []searchHit `json:"results"`
	}{hits})
}

func allowedSearchRequest(r *http.Request, c config.Workspace) bool {
	if c.Server.Host == "127.0.0.1" || c.Server.Host == "localhost" || c.Server.Host == "::1" {
		host := r.Host
		if name, _, err := net.SplitHostPort(host); err == nil {
			host = name
		}
		if host != "127.0.0.1" && host != "localhost" && host != "::1" {
			return false
		}
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host != r.Host || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return false
		}
	}
	return true
}
