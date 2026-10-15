package workspace

import (
	"fmt"
	"image/png"
	"os"
)

const maxPreviewBytes = 20 << 20
const maxPreviewPixels = 32_000_000

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
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	if info.Size() > maxPreviewBytes {
		return "", fmt.Errorf("Excalidraw PNG preview %s exceeds %d bytes", preview, maxPreviewBytes)
	}
	config, err := png.DecodeConfig(f)
	if err != nil {
		return "", fmt.Errorf("invalid Excalidraw PNG preview %s: %w", preview, err)
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > maxPreviewPixels {
		return "", fmt.Errorf("Excalidraw PNG preview %s exceeds %d pixels", preview, maxPreviewPixels)
	}
	if _, err := f.Seek(0, 0); err != nil {
		return "", err
	}
	if _, err := png.Decode(f); err != nil {
		return "", fmt.Errorf("invalid Excalidraw PNG preview %s: %w", preview, err)
	}
	return path, nil
}
