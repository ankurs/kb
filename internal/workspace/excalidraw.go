package workspace

import (
	"fmt"
	"image/png"
	"os"
)

// ExcalidrawPreview verifies the required PNG sidecar without following an
// escaping symlink. The source scene remains editable; only the PNG is shown
// as an image in the viewers.
func (i *Inventory) ExcalidrawPreview(key string) (string, error) {
	preview := key + ".png"
	path, err := i.Source(preview)
	if err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("Excalidraw image %s needs PNG preview %s: %w", key, preview, err)
	}
	defer f.Close()
	if _, err := png.DecodeConfig(f); err != nil {
		return "", fmt.Errorf("invalid Excalidraw PNG preview %s: %w", preview, err)
	}
	return path, nil
}
