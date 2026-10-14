package render

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ankurs/kb/internal/config"
	"github.com/ankurs/kb/internal/workspace"
)

const publishMarker = ".kb-published"
const markerContent = "kb publish output\n"

type PublishResult struct {
	Dir      string
	Bundles  []string
	Warnings []string
}

func Publish(c config.Workspace) (PublishResult, error) { return PublishWithLog(c, nil) }

func PublishWithLog(c config.Workspace, logf func(string, ...any)) (PublishResult, error) {
	var result PublishResult
	log := func(format string, args ...any) {
		if logf != nil {
			logf(format, args...)
		}
	}
	if err := c.Validate(); err != nil {
		return result, err
	}
	dir, err := c.PublishDirectory()
	if err != nil {
		return result, err
	}
	inv, err := workspace.Load(c)
	if err != nil {
		return result, err
	}
	known := map[string]bool{}
	for _, key := range inv.Keys {
		if parts := strings.SplitN(key, "/", 2); len(parts) == 2 {
			known[parts[0]] = true
		}
	}
	chosen := map[string]bool{}
	if len(c.Publish.Include) == 0 {
		for group := range known {
			chosen[group] = true
		}
	} else {
		for _, group := range c.Publish.Include {
			if !known[group] {
				return result, fmt.Errorf("publish.include: unknown folder %q", group)
			}
			chosen[group] = true
		}
	}
	for _, group := range c.Publish.Exclude {
		if !known[group] {
			return result, fmt.Errorf("publish.exclude: unknown folder %q", group)
		}
		delete(chosen, group)
	}
	rootPages := (len(c.Publish.Include) == 0 && len(c.Publish.Exclude) == 0) || c.Publish.Root
	var filter func(string) bool
	if !rootPages || len(chosen) != len(known) {
		filter = func(key string) bool {
			parts := strings.SplitN(key, "/", 2)
			if len(parts) == 1 {
				return rootPages
			}
			return chosen[parts[0]]
		}
	}
	log("publish: compiling HTML for %d folders", len(chosen))
	compiled, warnings, err := compileBundles(c, dir, filter, false)
	if err != nil {
		return result, err
	}
	files := map[string][]byte{}
	files[c.Publish.WikiName()] = compiled["wiki.html"]
	for group := range chosen {
		filename := group + ".html"
		if custom, ok := c.Publish.Names[group]; ok {
			filename = custom
		}
		if !config.ValidHTMLName(filename) {
			return result, fmt.Errorf("invalid publish filename %q", filename)
		}
		if _, exists := files[filename]; exists {
			return result, fmt.Errorf("duplicate publish filename %q", filename)
		}
		contents, ok := compiled[group+".html"]
		if !ok {
			return result, fmt.Errorf("no bundle for folder %q", group)
		}
		files[filename] = contents
	}
	if err := validateDestination(dir); err != nil {
		return result, err
	}
	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return result, err
	}
	staging, err := os.MkdirTemp(parent, ".kb-publish-")
	if err != nil {
		return result, err
	}
	defer os.RemoveAll(staging)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		log("publish: writing %s (%d bytes)", name, len(files[name]))
		if err := os.WriteFile(filepath.Join(staging, name), files[name], 0644); err != nil {
			return result, err
		}
	}
	if err := os.WriteFile(filepath.Join(staging, publishMarker), []byte(markerContent), 0644); err != nil {
		return result, err
	}
	backup := ""
	if _, err := os.Lstat(dir); err == nil {
		backup, err = os.MkdirTemp(parent, ".kb-publish-previous-")
		if err != nil {
			return result, err
		}
		if err := os.Remove(backup); err != nil {
			return result, err
		}
		if err := os.Rename(dir, backup); err != nil {
			return result, err
		}
	} else if !os.IsNotExist(err) {
		return result, err
	}
	if err := os.Rename(staging, dir); err != nil {
		if backup != "" {
			if restoreErr := os.Rename(backup, dir); restoreErr != nil {
				return result, fmt.Errorf("publish: %w; restore previous output: %v", err, restoreErr)
			}
		}
		return result, err
	}
	if backup != "" {
		if err := os.RemoveAll(backup); err != nil {
			warnings = append(warnings, "could not remove previous publish output: "+err.Error())
		}
	}
	result.Dir = dir
	result.Warnings = warnings
	for _, name := range names {
		result.Bundles = append(result.Bundles, filepath.Join(dir, name))
	}
	log("publish: wrote %d bundles to %s", len(names), dir)
	return result, nil
}

func validateDestination(dir string) error {
	info, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("refusing to replace non-directory publish output %s", dir)
	}
	marker, err := os.ReadFile(filepath.Join(dir, publishMarker))
	if err != nil || string(marker) != markerContent {
		return errors.New("refusing to replace directory without kb publish marker: " + dir)
	}
	return nil
}
