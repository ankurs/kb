package render

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ankurs/kb/internal/config"
)

func TestLiveReloadAfterMarkdownEdit(t *testing.T) {
	root := t.TempDir()
	page := filepath.Join(root, "README.md")
	if err := os.WriteFile(page, []byte("# Home\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c := config.Workspace{Root: root, Name: "Live"}
	server := httptest.NewServer(Handler(c))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/_kb/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(response.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("events: %d %q", response.StatusCode, response.Header.Get("Content-Type"))
	}
	scanner := bufio.NewScanner(response.Body)
	if !scanner.Scan() || scanner.Text() != ": connected" {
		t.Fatal("missing initial SSE connection")
	}
	if err := os.WriteFile(page, []byte("# Home\n\nChanged content.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for scanner.Scan() {
		if scanner.Text() == "data: changed" {
			return
		}
	}
	t.Fatalf("missing reload event after Markdown edit: %v", scanner.Err())
}
