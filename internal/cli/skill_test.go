package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillShowWithoutWorkspace(t *testing.T) {
	root := New()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"--workspace", filepath.Join(t.TempDir(), "missing"), "skill", "show"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.String() != string(kbSkill) || !strings.HasPrefix(out.String(), "---\nname: kb\n") {
		t.Fatalf("unexpected skill output: %q", out.String())
	}
}

func TestSkillInstallAndForce(t *testing.T) {
	base := t.TempDir()
	path, err := installSkill(base, false)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(base, ".agents", "skills", "kb", "SKILL.md") {
		t.Fatal(path)
	}
	if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, kbSkill) {
		t.Fatalf("installed content: %v", err)
	}
	custom := []byte("my edited skill")
	if err := os.WriteFile(path, custom, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := installSkill(base, false); err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("expected overwrite refusal, got %v", err)
	}
	if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, custom) {
		t.Fatalf("existing skill changed: %v", err)
	}
	if _, err := installSkill(base, true); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, kbSkill) {
		t.Fatalf("forced skill content: %v", err)
	}
}

func TestSkillInstallCLI(t *testing.T) {
	base := t.TempDir()
	t.Chdir(base)
	out, diagnostics := runCLI(t, "skill", "install")
	if diagnostics != "" || out != "installed .agents/skills/kb/SKILL.md\n" {
		t.Fatalf("install output: %q, stderr: %q", out, diagnostics)
	}
	if _, err := os.Stat(filepath.Join(base, ".agents", "skills", "kb", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	out, _ = runCLI(t, "skill", "install", "--global")
	if out != "installed "+filepath.Join(home, ".agents", "skills", "kb", "SKILL.md")+"\n" {
		t.Fatal(out)
	}
}

func TestSkillInstallDoesNotFollowSymlinks(t *testing.T) {
	for _, component := range []string{".agents", "skills", "kb", "SKILL.md"} {
		t.Run(component, func(t *testing.T) {
			base := t.TempDir()
			outside := t.TempDir()
			parts := map[string]string{
				".agents":  filepath.Join(base, ".agents"),
				"skills":   filepath.Join(base, ".agents", "skills"),
				"kb":       filepath.Join(base, ".agents", "skills", "kb"),
				"SKILL.md": filepath.Join(base, ".agents", "skills", "kb", "SKILL.md"),
			}
			path := parts[component]
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			target := outside
			if component == "SKILL.md" {
				target = filepath.Join(outside, "untouched")
				if err := os.WriteFile(target, []byte("original"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Symlink(target, path); err != nil {
				t.Fatal(err)
			}
			_, err := installSkill(base, true)
			if component == "SKILL.md" {
				if err != nil {
					t.Fatal(err)
				}
				if got, err := os.ReadFile(target); err != nil || string(got) != "original" {
					t.Fatalf("followed skill symlink: %q, %v", got, err)
				}
			} else if err == nil {
				t.Fatal("followed directory symlink")
			}
		})
	}
}
