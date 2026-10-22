package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// usageError reports a mistake in how a command was invoked and appends the
// command's usage line, examples, and a pointer to its help.
type usageError struct {
	cmd *cobra.Command
	err error
}

func (e usageError) Error() string {
	var b strings.Builder
	b.WriteString(e.err.Error())
	b.WriteString("\n\nUsage:\n  ")
	what := "flags"
	if e.cmd.HasAvailableSubCommands() {
		b.WriteString(e.cmd.CommandPath() + " COMMAND [flags]")
		what = "commands"
	} else {
		b.WriteString(e.cmd.UseLine())
	}
	if e.cmd.Example != "" {
		b.WriteString("\n\nExamples:\n")
		b.WriteString(e.cmd.Example)
	}
	fmt.Fprintf(&b, "\n\nRun '%s --help' for all %s.", e.cmd.CommandPath(), what)
	return b.String()
}

func (e usageError) Unwrap() error { return e.err }

func usagef(cmd *cobra.Command, format string, args ...any) error {
	return usageError{cmd: cmd, err: fmt.Errorf(format, args...)}
}

// noArgs rejects positional arguments, naming any unknown subcommand.
func noArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	if cmd.HasAvailableSubCommands() {
		return unknownCommand(cmd, args[0])
	}
	return usagef(cmd, "%q takes no arguments; got %q", cmd.CommandPath(), strings.Join(args, " "))
}

// oneArg requires exactly one positional argument named name.
func oneArg(name string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		switch len(args) {
		case 1:
			return nil
		case 0:
			return usagef(cmd, "missing %s argument", name)
		default:
			return usagef(cmd, "expected one %s argument, got %d; quote multi-word text", name, len(args))
		}
	}
}

func unknownCommand(cmd *cobra.Command, name string) error {
	if cmd.SuggestionsMinimumDistance <= 0 {
		cmd.SuggestionsMinimumDistance = 2
	}
	msg := fmt.Sprintf("unknown command %q for %q", name, cmd.CommandPath())
	if suggestions := cmd.SuggestionsFor(name); len(suggestions) > 0 {
		msg += "; did you mean " + strings.Join(suggestions, " or ") + "?"
	}
	return usageError{cmd: cmd, err: fmt.Errorf("%s", msg)}
}

// requireFlags reports every unset flag in names along with its help text.
func requireFlags(cmd *cobra.Command, names ...string) error {
	var missing, help []string
	for _, name := range names {
		f := cmd.Flags().Lookup(name)
		if f == nil || f.Changed {
			continue
		}
		missing = append(missing, "--"+name)
		help = append(help, fmt.Sprintf("  --%-8s %s", name, f.Usage))
	}
	switch len(missing) {
	case 0:
		return nil
	case 1:
		return usagef(cmd, "missing required flag %s\n%s", missing[0], strings.Join(help, "\n"))
	default:
		list := strings.Join(missing[:len(missing)-1], ", ") + " and " + missing[len(missing)-1]
		return usagef(cmd, "missing required flags %s\n%s", list, strings.Join(help, "\n"))
	}
}
