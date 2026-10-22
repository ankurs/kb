package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/ankurs/kb/internal/config"
	"github.com/ankurs/kb/internal/render"
	"github.com/ankurs/kb/internal/workspace"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	var start, global string
	var verbose bool
	root := &cobra.Command{Use: "kb", Short: "Offline-first, Markdown-native knowledge base", SilenceUsage: true, SilenceErrors: true, Args: noArgs, RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	}}
	root.PersistentFlags().StringVarP(&start, "workspace", "w", "", "content repository path (default: discover kb.yaml)")
	root.PersistentFlags().StringVar(&global, "global-config", "", "user-global overrides (default: ~/.config/kb/config.yaml)")
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "show progress diagnostics on stderr")
	trace := func(cmd *cobra.Command, format string, args ...any) {
		if verbose {
			fmt.Fprintf(cmd.ErrOrStderr(), "kb: "+format+"\n", args...)
		}
	}
	load := func(cmd *cobra.Command) (config.Workspace, error) {
		if start == "" {
			start = "."
		}
		trace(cmd, "loading workspace from %s", start)
		c, err := config.Load(start, global)
		if err == nil {
			trace(cmd, "workspace %s (config %s)", c.Root, c.ConfigPath)
		}
		return c, err
	}

	root.AddCommand(skillCommand())
	root.AddCommand(&cobra.Command{Use: "init", Short: "Create a minimal kb.yaml", Args: noArgs, RunE: func(cmd *cobra.Command, args []string) error {
		dir := start
		if dir == "" {
			dir = "."
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			return err
		}
		if err := config.Init(abs); err != nil {
			return err
		}
		trace(cmd, "created workspace config %s", filepath.Join(abs, "kb.yaml"))
		fmt.Fprintln(cmd.OutOrStdout(), "created", filepath.Join(abs, "kb.yaml"))
		return nil
	}})
	var project string
	capture := &cobra.Command{Use: "capture TEXT", Short: "Append a dated inbox entry", Args: oneArg("TEXT"), Example: "  kb capture \"Ask about the loan policy\" --at tool-library", RunE: func(cmd *cobra.Command, args []string) error {
		c, err := load(cmd)
		if err != nil {
			return err
		}
		if err := workspace.Capture(c, project, args[0], time.Now()); err != nil {
			return err
		}
		trace(cmd, "capture: appended to %s", c.Capture.Inbox)
		return nil
	}}
	capture.Flags().StringVar(&project, "at", "", "project path to label the capture")
	root.AddCommand(capture)
	var at, title, dateString string
	newCmd := &cobra.Command{Use: "new meeting|page", Short: "Create a page from a template without overwriting", Example: "  kb new page --at tool-library --title \"Architecture\"\n  kb new meeting --at tool-library --title \"Design review\" --date 2026-10-10", Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return usagef(cmd, "missing page kind; expected \"meeting\" or \"page\"")
		}
		if err := oneArg("kind")(cmd, args); err != nil {
			return err
		}
		if args[0] != "meeting" && args[0] != "page" {
			return usagef(cmd, "unknown page kind %q; expected \"meeting\" or \"page\"", args[0])
		}
		return nil
	}, RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireFlags(cmd, "at", "title"); err != nil {
			return err
		}
		date := time.Now()
		if dateString != "" {
			var err error
			date, err = time.ParseInLocation("2006-01-02", dateString, time.Local)
			if err != nil {
				return usagef(cmd, "invalid --date %q; use YYYY-MM-DD, e.g. 2026-10-10", dateString)
			}
		}
		c, err := load(cmd)
		if err != nil {
			return err
		}
		path, err := workspace.New(c, args[0], at, title, date)
		if err == nil {
			trace(cmd, "new: created %s", path)
			fmt.Fprintln(cmd.OutOrStdout(), "created", path)
		}
		return err
	}}
	newCmd.Flags().StringVar(&at, "at", "", "project directory, relative to content root (required)")
	newCmd.Flags().StringVar(&title, "title", "", "page title (required)")
	newCmd.Flags().StringVar(&dateString, "date", "", "meeting date (YYYY-MM-DD)")
	root.AddCommand(newCmd)
	root.AddCommand(&cobra.Command{Use: "review", Short: "Show unprocessed inbox entries without changing them", Args: noArgs, RunE: func(cmd *cobra.Command, args []string) error {
		c, err := load(cmd)
		if err != nil {
			return err
		}
		text, err := workspace.Review(c)
		if err == nil {
			trace(cmd, "review: read %s", c.Capture.Inbox)
			fmt.Fprint(cmd.OutOrStdout(), text)
		}
		return err
	}})
	root.AddCommand(&cobra.Command{Use: "check", Short: "Validate authored headings and local links (read-only)", Args: noArgs, RunE: func(cmd *cobra.Command, args []string) error {
		c, err := load(cmd)
		if err != nil {
			return err
		}
		inv, err := workspace.Load(c)
		if err != nil {
			return err
		}
		trace(cmd, "check: validating %d pages", len(inv.Keys))
		count := 0
		for _, issue := range inv.Check() {
			level := "error"
			if issue.Warning {
				level = "warning"
			} else {
				count++
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s:%d: %s: %s\n", issue.Page, issue.Line, level, issue.Message)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "checked %d pages; %d error(s)\n", len(inv.Keys), count)
		if count > 0 {
			return errors.New("content check failed")
		}
		return nil
	}})
	var scope string
	find := &cobra.Command{Use: "find QUERY", Short: "Search local Markdown without models", Args: oneArg("QUERY"), Example: "  kb find \"borrowing policy\"\n  kb find ladder --scope tool-library", RunE: func(cmd *cobra.Command, args []string) error {
		c, err := load(cmd)
		if err != nil {
			return err
		}
		inv, err := workspace.Load(c)
		if err != nil {
			return err
		}
		trace(cmd, "find: searching %d pages (scope %q)", len(inv.Keys), scope)
		result := inv.Find(args[0], scope)
		trace(cmd, "find: %d matching lines", strings.Count(result, "\n"))
		fmt.Fprint(cmd.OutOrStdout(), result)
		return nil
	}}
	find.Flags().StringVar(&scope, "scope", "", "path prefix")
	root.AddCommand(find)
	var collection string
	search := &cobra.Command{Use: "search QUERY", Short: "Hybrid search through the configured qmd provider", Args: oneArg("QUERY"), Example: "  kb search \"how do members borrow tools\"", RunE: func(cmd *cobra.Command, args []string) error {
		c, err := load(cmd)
		if err != nil {
			return err
		}
		if c.Search.Provider != "qmd" {
			return fmt.Errorf("unsupported search provider %q", c.Search.Provider)
		}
		trace(cmd, "search: provider qmd (collection %q)", collection)
		out, err := (workspace.QMD{Root: c.Root, ConfigDir: c.Search.QMDConfig, Logf: func(format string, args ...any) { trace(cmd, format, args...) }}).Search(args[0], collection)
		if err == nil {
			fmt.Fprint(cmd.OutOrStdout(), out)
		}
		return err
	}}
	search.Flags().StringVar(&collection, "scope", "", "qmd collection")
	root.AddCommand(search)
	var force bool
	index := &cobra.Command{Use: "index", Short: "Run qmd update and embed", Args: noArgs, RunE: func(cmd *cobra.Command, args []string) error {
		c, err := load(cmd)
		if err != nil {
			return err
		}
		if c.Search.Provider != "qmd" {
			return fmt.Errorf("unsupported search provider %q", c.Search.Provider)
		}
		trace(cmd, "index: provider qmd (force %t)", force)
		return (workspace.QMD{Root: c.Root, ConfigDir: c.Search.QMDConfig, Logf: func(format string, args ...any) { trace(cmd, format, args...) }}).Index(force)
	}}
	index.Flags().BoolVar(&force, "force", false, "force re-embedding")
	root.AddCommand(index)
	root.AddCommand(&cobra.Command{Use: "verify", Short: "Check content and current offline outputs without writing", Args: noArgs, RunE: func(cmd *cobra.Command, args []string) error {
		c, err := load(cmd)
		if err != nil {
			return err
		}
		inv, err := workspace.Load(c)
		if err != nil {
			return err
		}
		trace(cmd, "verify: validating %d pages", len(inv.Keys))
		for _, issue := range inv.Check() {
			if !issue.Warning {
				return fmt.Errorf("%s:%d: %s", issue.Page, issue.Line, issue.Message)
			}
		}
		trace(cmd, "verify: comparing offline outputs")
		if err := render.Verify(c); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "content and offline outputs current")
		return nil
	}})
	root.AddCommand(&cobra.Command{Use: "build", Short: "Build offline site and tracked single-file snapshots", Args: noArgs, RunE: func(cmd *cobra.Command, args []string) error {
		c, err := load(cmd)
		if err != nil {
			return err
		}
		trace(cmd, "build: compiling workspace")
		r, err := render.BuildWithLog(c, func(format string, args ...any) { trace(cmd, format, args...) })
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "site:", r.Site)
		for _, p := range r.Bundles {
			fmt.Fprintln(cmd.OutOrStdout(), "bundle:", p)
		}
		for _, w := range r.Warnings {
			fmt.Fprintln(cmd.OutOrStdout(), "warning:", w)
		}
		return nil
	}})
	var publishOutput, publishWiki string
	var include, exclude []string
	var includeRoot bool
	publish := &cobra.Command{Use: "publish", Short: "Build HTML-only wiki bundles in a distribution directory", Args: noArgs, Example: "  kb publish --output dist --include tool-library", RunE: func(cmd *cobra.Command, args []string) error {
		c, err := load(cmd)
		if err != nil {
			return err
		}
		if cmd.Flags().Changed("output") {
			c.Publish.Dir = publishOutput
		}
		if cmd.Flags().Changed("wiki") {
			c.Publish.Wiki = publishWiki
		}
		if cmd.Flags().Changed("include") {
			c.Publish.Include = include
		}
		if cmd.Flags().Changed("exclude") {
			c.Publish.Exclude = exclude
		}
		if cmd.Flags().Changed("root-pages") {
			c.Publish.Root = includeRoot
		}
		r, err := render.PublishWithLog(c, func(format string, args ...any) { trace(cmd, format, args...) })
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "published:", r.Dir)
		for _, p := range r.Bundles {
			fmt.Fprintln(cmd.OutOrStdout(), "bundle:", p)
		}
		for _, w := range r.Warnings {
			fmt.Fprintln(cmd.OutOrStdout(), "warning:", w)
		}
		return nil
	}}
	publish.Flags().StringVarP(&publishOutput, "output", "o", "", "distribution directory (relative to workspace or absolute)")
	publish.Flags().StringVar(&publishWiki, "wiki", "", "filename for the combined wiki")
	publish.Flags().StringSliceVar(&include, "include", nil, "top-level folders to include; repeat or comma-separate")
	publish.Flags().StringSliceVar(&exclude, "exclude", nil, "top-level folders to omit; repeat or comma-separate")
	publish.Flags().BoolVar(&includeRoot, "root-pages", false, "include root pages with folder filters")
	root.AddCommand(publish)
	var showSync bool
	syncCommand := &cobra.Command{Use: "sync", Short: "Run the configured sync command", Args: noArgs, RunE: func(cmd *cobra.Command, args []string) error {
		c, err := load(cmd)
		if err != nil {
			return err
		}
		if strings.TrimSpace(c.Sync.Command) == "" {
			return errors.New("sync.command is not configured in kb.yaml or user settings")
		}
		if showSync {
			dir, err := c.PublishDirectory()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "cwd:", c.Root)
			fmt.Fprintln(cmd.OutOrStdout(), "publish:", dir)
			fmt.Fprintln(cmd.OutOrStdout(), "command:", c.Sync.Command)
			return nil
		}
		trace(cmd, "sync: running configured command")
		return workspace.RunSync(cmd.Context(), c, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
	}}
	syncCommand.Flags().BoolVar(&showSync, "show", false, "print command and working directory without running")
	root.AddCommand(syncCommand)
	var host string
	var port int
	serve := &cobra.Command{Use: "serve", Short: "Serve a local Docsify viewer with live navigation and reload", Args: noArgs, Example: "  kb serve --port 3000", RunE: func(cmd *cobra.Command, args []string) error {
		c, err := load(cmd)
		if err != nil {
			return err
		}
		if cmd.Flags().Changed("host") {
			c.Server.Host = host
		}
		if cmd.Flags().Changed("port") {
			c.Server.Port = port
		}
		trace(cmd, "serve: listening on %s:%d", c.Server.Host, c.Server.Port)
		return render.ServeWithLog(c, func(format string, args ...any) { trace(cmd, format, args...) })
	}}
	serve.Flags().StringVar(&host, "host", "127.0.0.1", "bind address")
	serve.Flags().IntVarP(&port, "port", "p", 3000, "listen port")
	root.AddCommand(serve)
	root.AddCommand(&cobra.Command{Use: "doctor", Short: "Check optional dependencies and workspace configuration", Args: noArgs, RunE: func(cmd *cobra.Command, args []string) error {
		c, err := load(cmd)
		if err != nil {
			return err
		}
		trace(cmd, "doctor: checking optional qmd configuration")
		fmt.Fprintln(cmd.OutOrStdout(), "workspace:", c.Root)
		fmt.Fprintln(cmd.OutOrStdout(), "config:", c.ConfigPath)
		for _, item := range workspace.Doctor(c.Root, c.Search.QMDConfig) {
			fmt.Fprintln(cmd.OutOrStdout(), "optional:", item)
		}
		return nil
	}})
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error { return usageError{cmd: cmd, err: err} })

	return root
}
