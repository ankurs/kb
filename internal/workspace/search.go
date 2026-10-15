package workspace

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ankurs/kb/internal/config"
	"gopkg.in/yaml.v3"
)

// SearchProvider defines query and indexing operations.
type SearchProvider interface {
	Search(query, scope string) (string, error)
	Index(force bool) error
}
type QMD struct {
	Root, ConfigDir string
	Logf            func(string, ...any)
}

func (q QMD) logf(format string, args ...any) {
	if q.Logf != nil {
		q.Logf(format, args...)
	}
}

// qmd discovers the project-local .qmd index from its working directory.
// Global overrides must not redirect a kb command to another index or trust store.
func qmdBaseEnv() []string {
	env := make([]string, 0, len(os.Environ()))
	for _, item := range os.Environ() {
		if !strings.HasPrefix(item, "QMD_CONFIG_DIR=") && !strings.HasPrefix(item, "INDEX_PATH=") {
			env = append(env, item)
		}
	}
	return env
}

func (q QMD) env() []string {
	env := qmdBaseEnv()
	if q.ConfigDir == "" || q.ConfigDir == ".qmd" {
		return env
	}
	dir := filepath.Join(q.Root, q.ConfigDir)
	return append(env, "QMD_CONFIG_DIR="+dir, "INDEX_PATH="+filepath.Join(dir, "index.sqlite"))
}

func (q QMD) projectConfig() (string, error) {
	dir := q.ConfigDir
	if dir == "" {
		dir = ".qmd"
	}
	for _, name := range []string{"index.yaml", "index.yml"} {
		path := filepath.Join(q.Root, dir, name)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path, nil
		}
	}
	return "", fmt.Errorf("project qmd config not found in %s; run `qmd init` in this workspace to use `kb search` and `kb index`", filepath.Join(q.Root, dir))
}

type qmdConfig struct {
	Collections map[string]struct {
		Path   string `yaml:"path"`
		Update string `yaml:"update"`
	} `yaml:"collections"`
	Models struct {
		Embed    string `yaml:"embed"`
		Rerank   string `yaml:"rerank"`
		Generate string `yaml:"generate"`
	} `yaml:"models"`
}

func readQMDConfig(configPath string) (qmdConfig, error) {
	contents, err := os.ReadFile(configPath)
	if err != nil {
		return qmdConfig{}, err
	}
	var config qmdConfig
	if err := yaml.Unmarshal(contents, &config); err != nil {
		return qmdConfig{}, fmt.Errorf("qmd config %s: %w", configPath, err)
	}
	return config, nil
}

func (q QMD) configuredCollections() (qmdConfig, error) {
	configPath, err := q.projectConfig()
	if err != nil {
		return qmdConfig{}, err
	}
	config, err := readQMDConfig(configPath)
	if err != nil {
		return qmdConfig{}, err
	}
	if len(config.Collections) == 0 {
		return qmdConfig{}, fmt.Errorf("no qmd collections configured in %s; run `qmd collection add . --name wiki` in this workspace, then `kb index`", configPath)
	}
	return config, nil
}

func (q QMD) needsTrust(configPath string) (bool, error) {
	config, err := readQMDConfig(configPath)
	if err != nil {
		return false, err
	}
	// qmd's built-in model URIs do not require project-config approval.
	defaults := map[string]string{
		"embed":    "hf:ggml-org/embeddinggemma-300M-GGUF/embeddinggemma-300M-Q8_0.gguf",
		"rerank":   "hf:ggml-org/Qwen3-Reranker-0.6B-Q8_0-GGUF/qwen3-reranker-0.6b-q8_0.gguf",
		"generate": "hf:tobil/qmd-query-expansion-1.7B-gguf/qmd-query-expansion-1.7B-q4_k_m.gguf",
	}
	models := map[string]string{"embed": config.Models.Embed, "rerank": config.Models.Rerank, "generate": config.Models.Generate}
	for slot, uri := range models {
		if uri != "" && uri != defaults[slot] {
			return true, nil
		}
	}
	root, err := filepath.EvalSymlinks(q.Root)
	if err != nil {
		return false, err
	}
	for _, collection := range config.Collections {
		if collection.Update != "" {
			return true, nil
		}
		raw := collection.Path
		if raw == "" {
			continue
		}
		if raw == "~" || strings.HasPrefix(raw, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				return false, err
			}
			if raw == "~" {
				raw = home
			} else {
				raw = filepath.Join(home, strings.TrimPrefix(raw, "~/"))
			}
		}
		if !filepath.IsAbs(raw) {
			raw = filepath.Join(q.Root, raw)
		}
		target := filepath.Clean(raw)
		if resolved, err := filepath.EvalSymlinks(target); err == nil {
			target = resolved
		}
		relative, err := filepath.Rel(root, target)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return true, nil
		}
	}
	return false, nil
}

func (q QMD) trusted() error {
	configPath, err := q.projectConfig()
	if err != nil {
		return err
	}
	if _, err := exec.LookPath("qmd"); err != nil {
		return errors.New("qmd is not installed; `kb find` works without it")
	}
	gated, err := q.needsTrust(configPath)
	if err != nil {
		return err
	}
	if !gated {
		return nil
	}
	q.logf("qmd: checking configuration trust")
	cmd := exec.Command("qmd", "trust", "list")
	cmd.Dir = q.Root
	cmd.Env = qmdBaseEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("qmd trust list: %w: %s", err, out)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, configPath+" (trusted ") {
			return nil
		}
	}
	return errors.New("project qmd config has no trust record; run `qmd trust` interactively in this workspace")
}
func (q QMD) run(args ...string) (string, error) {
	name := args[0]
	q.logf("qmd: running %s", name)
	start := time.Now()
	cmd := exec.Command("qmd", args...)
	cmd.Dir = q.Root
	cmd.Env = q.env()
	out, err := cmd.CombinedOutput()
	if err != nil {
		q.logf("qmd: %s failed after %s", name, time.Since(start).Round(time.Millisecond))
		return "", fmt.Errorf("qmd %s: %w\n%s", strings.Join(args, " "), err, out)
	}
	q.logf("qmd: %s completed in %s", name, time.Since(start).Round(time.Millisecond))
	return string(out), nil
}
func (q QMD) Search(query, scope string) (string, error) {
	config, err := q.configuredCollections()
	if err != nil {
		return "", err
	}
	if scope != "" {
		if _, ok := config.Collections[scope]; !ok {
			return "", fmt.Errorf("qmd collection %q is not configured; check .qmd/index.yml", scope)
		}
	}
	if err := q.trusted(); err != nil {
		return "", err
	}
	args := []string{"query"}
	if scope != "" {
		args = append(args, "-c", scope)
	}
	out, err := q.run(append(args, query)...)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(out) == "" {
		return "", errors.New("qmd returned no results; check `qmd status` and run `kb index` if documents have not been indexed")
	}
	return out, nil
}

// ServeCollections reports project-local collections suitable for the live viewer.
// An index created by qmd init is required; a checked-in config alone is not.
func (q QMD) ServeCollections() (map[string]string, error) {
	configPath, err := q.projectConfig()
	if err != nil {
		return nil, err
	}
	root := config.Workspace{Root: q.Root}
	rel, err := filepath.Rel(q.Root, configPath)
	if err != nil {
		return nil, err
	}
	if _, err := root.ReadablePath(rel); err != nil {
		return nil, err
	}
	settings, err := q.configuredCollections()
	if err != nil {
		return nil, err
	}
	dir := q.ConfigDir
	if dir == "" {
		dir = ".qmd"
	}
	index := filepath.Join(dir, "index.sqlite")
	if info, err := os.Stat(filepath.Join(q.Root, index)); err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("qmd index not initialized in this workspace; run `qmd init` and `kb index`")
	}
	if _, err := root.ReadablePath(index); err != nil {
		return nil, err
	}
	if err := q.trusted(); err != nil {
		return nil, err
	}
	paths := make(map[string]string, len(settings.Collections))
	for name, collection := range settings.Collections {
		paths[name] = collection.Path
	}
	return paths, nil
}

// CommandEnv applies the same project configuration to long-lived qmd processes.
func (q QMD) CommandEnv() []string { return q.env() }

func (q QMD) Index(force bool) error {
	if _, err := q.configuredCollections(); err != nil {
		return err
	}
	if err := q.trusted(); err != nil {
		return err
	}
	if _, err := q.run("update"); err != nil {
		return err
	}
	args := []string{"embed"}
	if force {
		args = append(args, "-f")
	}
	_, err := q.run(args...)
	return err
}
func (i *Inventory) Find(query, scope string) string {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return ""
	}
	var b strings.Builder
	count := 0
	for _, key := range i.Keys {
		if scope != "" && key != scope && !strings.HasPrefix(key, scope+"/") {
			continue
		}
		for n, line := range strings.Split(i.Pages[key].Markdown, "\n") {
			if strings.Contains(strings.ToLower(line), needle) {
				fmt.Fprintf(&b, "%s:%d: %s\n", filepath.ToSlash(key), n+1, strings.TrimSpace(line))
				count++
				if count >= 100 {
					return b.String()
				}
			}
		}
	}
	return b.String()
}
func Doctor(cRoot, configDir string) []string {
	q := QMD{Root: cRoot, ConfigDir: configDir}
	if _, err := q.projectConfig(); err != nil {
		return []string{err.Error()}
	}
	if _, err := q.configuredCollections(); err != nil {
		return []string{err.Error()}
	}
	if err := q.trusted(); err != nil {
		return []string{err.Error()}
	}
	return nil
}
