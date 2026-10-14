package workspace

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/ankurs/kb/internal/config"
)

func RunSync(ctx context.Context, c config.Workspace, stdin io.Reader, stdout, stderr io.Writer) error {
	if strings.TrimSpace(c.Sync.Command) == "" {
		return errors.New("sync.command is not configured in kb.yaml or user settings")
	}
	dir, err := c.PublishDirectory()
	if err != nil {
		return err
	}
	command := exec.CommandContext(ctx, "sh", "-c", c.Sync.Command)
	command.Dir = c.Root
	command.Env = append(os.Environ(), "KB_WORKSPACE_ROOT="+c.Root, "KB_PUBLISH_DIR="+dir)
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("sync command: %w", err)
	}
	return nil
}
