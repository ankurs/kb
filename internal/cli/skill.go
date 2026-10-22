package cli

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed skill/SKILL.md
var kbSkill []byte

func skillCommand() *cobra.Command {
	var global, force bool
	skill := &cobra.Command{Use: "skill", Short: "Show or install the kb agent skill", Args: noArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	}}
	skill.AddCommand(&cobra.Command{Use: "show", Short: "Print the bundled kb skill", Args: noArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		_, err := cmd.OutOrStdout().Write(kbSkill)
		return err
	}})
	install := &cobra.Command{Use: "install", Short: "Install the kb skill into ./.agents/skills/kb", Args: noArgs, Example: "  kb skill install\n  kb skill install --global", RunE: func(cmd *cobra.Command, _ []string) error {
		base := "."
		if global {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			base = home
		}
		path, err := installSkill(base, force)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "installed", path)
		return nil
	}}
	install.Flags().BoolVar(&global, "global", false, "install in ~/.agents/skills/kb instead")
	install.Flags().BoolVarP(&force, "force", "f", false, "replace an existing skill file")
	skill.AddCommand(install)
	return skill
}

func installSkill(base string, force bool) (string, error) {
	agents := filepath.Join(base, ".agents")
	skills := filepath.Join(agents, "skills")
	dir := filepath.Join(skills, "kb")
	path := filepath.Join(dir, "SKILL.md")
	// Never follow an existing symlink into an unexpected destination.
	for _, part := range []string{agents, skills, dir} {
		info, err := os.Lstat(part)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(part, 0755); err != nil {
				return "", err
			}
			continue
		}
		if err != nil {
			return "", err
		}
		if !info.IsDir() {
			return "", fmt.Errorf("not a directory: %s", part)
		}
	}
	if _, err := os.Lstat(path); err == nil {
		if !force {
			return "", fmt.Errorf("skill already exists at %s (use --force to replace it)", path)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	// Rename replaces a skill-file symlink rather than writing through it.
	file, err := os.CreateTemp(dir, ".SKILL-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(kbSkill); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Chmod(0644); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(file.Name(), path); err != nil {
		return "", err
	}
	return path, nil
}
