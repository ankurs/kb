package render

import (
	"fmt"

	"github.com/ankurs/kb/internal/config"
	"github.com/ankurs/kb/internal/workspace"
)

func checkActiveResource(c config.Workspace, source, key string) error {
	if c.Render.TrustHTML {
		return nil
	}
	active, err := workspace.ActiveLinkedFile(source)
	if err != nil {
		return err
	}
	if active {
		return fmt.Errorf("active linked file %s requires render.trust_html: true", key)
	}
	return nil
}
