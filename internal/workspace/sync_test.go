package workspace

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSyncFromWorkspace(t *testing.T) {
	c, _ := fixture(t)
	c.Publish.Dir = filepath.Join(t.TempDir(), "output")
	c.Sync.Command = `printf 'root=%s\npublish=%s\ncwd=%s\n' "$KB_WORKSPACE_ROOT" "$KB_PUBLISH_DIR" "$PWD"`
	var out, errors bytes.Buffer
	if err := RunSync(context.Background(), c, strings.NewReader(""), &out, &errors); err != nil {
		t.Fatal(err)
	}
	physicalRoot, err := filepath.EvalSymlinks(c.Root)
	if err != nil {
		t.Fatal(err)
	}
	if errors.Len() != 0 || !strings.Contains(out.String(), "root="+c.Root) || !strings.Contains(out.String(), "publish="+c.Publish.Dir) || !strings.Contains(out.String(), "cwd="+physicalRoot) {
		t.Fatalf("stdout=%q stderr=%q", out.String(), errors.String())
	}
	c.Sync.Command = "exit 23"
	if err := RunSync(context.Background(), c, nil, &out, &errors); err == nil || !strings.Contains(err.Error(), "23") {
		t.Fatalf("expected exit failure: %v", err)
	}
	c.Sync.Command = ""
	if err := RunSync(context.Background(), c, nil, &out, &errors); err == nil {
		t.Fatal("accepted missing sync command")
	}
}
