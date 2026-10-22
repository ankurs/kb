package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestServeLoadsBrowserAssets(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "kb")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build kb: %v\n%s", err, output)
	}

	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "kb.yaml"), []byte("version: 1\nsearch:\n  serve_qmd: false\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "README.md"), []byte("# Serve smoke test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	log, err := os.Create(filepath.Join(t.TempDir(), "serve.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	serve := exec.CommandContext(ctx, binary, "--workspace", workspace, "--global-config", filepath.Join(workspace, "absent.yaml"), "serve", "--port", strconv.Itoa(port))
	serve.Stdout, serve.Stderr = log, log
	if err := serve.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- serve.Wait() }()
	t.Cleanup(func() {
		_ = serve.Process.Signal(os.Interrupt)
		select {
		case err := <-done:
			if err != nil && ctx.Err() == nil {
				t.Errorf("kb serve exited: %v", err)
			}
		case <-time.After(3 * time.Second):
			_ = serve.Process.Kill()
			<-done
			t.Error("kb serve did not stop after interrupt")
		}
		if t.Failed() {
			b, _ := os.ReadFile(log.Name())
			t.Logf("kb serve output:\n%s", b)
		}
	})

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	client := &http.Client{Timeout: time.Second}
	var page []byte
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		response, err := client.Get(base + "/")
		if err != nil {
			continue
		}
		page, err = io.ReadAll(response.Body)
		response.Body.Close()
		if err == nil && response.StatusCode == http.StatusOK {
			break
		}
	}
	if len(page) == 0 {
		t.Fatal("kb serve did not return the viewer")
	}

	resources := regexp.MustCompile(`(?:src|href)="((?:vendor/|_kb/)[^"]+)"`).FindAllSubmatch(page, -1)
	vendorCount := 0
	for _, match := range resources {
		path := string(match[1])
		if strings.HasPrefix(path, "vendor/") {
			vendorCount++
		}
		response, err := client.Get(base + "/" + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil || response.StatusCode != http.StatusOK || len(body) == 0 {
			t.Fatalf("GET %s: status %d, %d bytes, %v", path, response.StatusCode, len(body), err)
		}
	}
	if vendorCount < 8 || !strings.Contains(string(page), "vendor/docsify/docsify-4.13.1.min.js") {
		t.Fatalf("viewer missing pinned browser resources: %d vendor URLs", vendorCount)
	}
	response, err := client.Get(base + "/README.md")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || response.StatusCode != http.StatusOK || !strings.Contains(string(body), "# Serve smoke test") {
		t.Fatalf("served Markdown: status %d, %q, %v", response.StatusCode, body, err)
	}
}
