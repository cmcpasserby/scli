package scli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"
	"text/tabwriter"
)

type Command struct {
	// Usage is a one liner usage message. First word of Usage is used for the Command's name.
	// Required for sub-commands.
	// Recommend syntax is:
	//
	//     cmd [flags] subcmd [flags] <required> [<optional> ...]
	Usage string

	// Aliases is a slice of alternate names that can be used for a sub command instead of the first word of usage.
	// Optional.
	Aliases []string

	// ShortHelp is a short description that is displayed in the global help -h output.
	// Optional, but recommended.
	ShortHelp string

	// LongHelp is a longer description that is displayed int the '<this-command> -h' output.
	// If not provided ShortHelp will be used in its place. Optional.
	LongHelp string

	// Subcommands is a slice of commands supported by Command.
	// Subcommands are optional and only needed if you application needs multiple commands.
	Subcommands []*Command

	// TODO
	UsageFunc func(c *Command) string

	// FlagSet for this command. Optional, but if none is provided,
	// an empty FlagSet will be defined to ensure -h works as expected.
	FlagSet *flag.FlagSet

	// ArgsValidator provides a validation function for arguments. There are multiple builtin validators as the
	// XArgs functions in this package.
	// Any error returned by ArgsValidator gets wrapped by an ErrInvalidArguments then is returned by Run or ParseAndRun.
	// When ArgsValidator returns an error the commands usage will be printed as well as the body of the error message.
	ArgsValidator ArgsValidator

	// Exec is the function that does the actual work, most Command's will implement this, unless they are just a
	// namespace for Subcommands.
	// The error returned by Exec will be bubble up and be returned by Run and ParseAndRun.
	// If flag.ErrHelp or ErrInvalidArguments is returned the commands usage will be printed to the output.
	Exec func(ctx context.Context, args []string) error

	selected *Command // the command that was selected by parse

	args []string // remaining args after flag parsing that should be passed to Exec function
}

// Name of the command is derived from first word of Usage
func (c *Command) Name() string {
	name := c.Usage
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

// Parse the command line arguments for this command and all sub-commands
func (c *Command) Parse(args []string) error {
	if c.selected != nil {
		return nil
	}

	if c.FlagSet == nil {
		c.FlagSet = flag.NewFlagSet(c.Name(), flag.ExitOnError)
	}

	if c.UsageFunc == nil {
		c.UsageFunc = defaultUsageFunc
	}

	c.FlagSet.Usage = func() {
		_, _ = fmt.Fprintln(c.FlagSet.Output(), c.UsageFunc(c))
	}

	if err := c.FlagSet.Parse(args); err != nil {
		return err
	}

	c.args = c.FlagSet.Args()
	if len(c.args) > 0 {
		for _, cmd := range c.Subcommands {
			if cmd.selectedBy(c.args[0]) {
				c.selected = cmd
				return cmd.Parse(c.args[1:])
			}
		}
	}

	c.selected = c

	if c.Exec == nil {
		return NoExecError{Command: c}
	}

	if c.ArgsValidator != nil {
		if err := c.ArgsValidator(c.args); err != nil {
			c.FlagSet.Usage()
			return fmt.Errorf("%w: %s", ErrInvalidArguments, err.Error())
		}
	}

	return nil
}

// Run executes the previously selected command from a parsed Command.
func (c *Command) Run(ctx context.Context) (err error) {
	if c.selected == nil {
		return ErrUnparsed
	}

	if c.selected == c && c.Exec == nil {
		return NoExecError{Command: c}
	}

	if c.selected == c && c.Exec != nil {
		defer func() {
			if errors.Is(err, flag.ErrHelp) || errors.Is(err, ErrInvalidArguments) {
				c.FlagSet.Usage()
			}
		}()

		return c.Exec(ctx, c.args)
	}

	if err = c.selected.Run(ctx); err != nil {
		return err
	}

	return nil
}

// ParseAndRun is a helper function to execute parse and run in a single invocation.
func (c *Command) ParseAndRun(ctx context.Context, args []string) error {
	if err := c.Parse(args); err != nil {
		return err
	}

	if err := c.Run(ctx); err != nil {
		return err
	}

	return nil
}

func (c *Command) selectedBy(name string) bool {
	aliases := append([]string{c.Name()}, c.Aliases...)

	for _, s := range aliases {
		if strings.EqualFold(name, s) {
			return true
		}
	}
	return false
}

//goland:noinspection GoUnhandledErrorResult
func defaultUsageFunc(c *Command) string {
	var b strings.Builder

	fmt.Fprintln(&b, "USAGE")
	if c.Usage != "" {
		fmt.Fprintf(&b, " %s\n", c.Usage)
	} else {
		fmt.Fprintf(&b, " %s\n", c.Name())
	}
	fmt.Fprintln(&b)

	if c.LongHelp != "" {
		fmt.Fprintf(&b, "%s\n\n", c.LongHelp)
	} else if c.ShortHelp != "" {
		fmt.Fprintf(&b, "%s\n\n", c.ShortHelp)
	}

	if len(c.Subcommands) > 0 {
		fmt.Fprintln(&b, "SUBCOMMANDS")
		tw := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)

		for _, subcommand := range c.Subcommands {
			fmt.Fprintf(tw, "  %s\t%s\n", subcommand.Name(), subcommand.ShortHelp)
		}
		tw.Flush()
		fmt.Fprintln(&b)
	}

	if countFlags(c.FlagSet) > 0 {
		fmt.Fprintln(&b, "FLAGS")

		tw := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
		c.FlagSet.VisitAll(func(f *flag.Flag) {
			space := " "
			if isBoolFlag(f) {
				space = "="
			}

			def := f.DefValue
			if def == "" {
				def = "..."
			}

			fmt.Fprintf(tw, "  -%s%s%s\t%s\n", f.Name, space, def, f.Usage)
		})
		tw.Flush()
		fmt.Fprintln(&b)
	}

	return strings.TrimSpace(b.String()) + "\n"
}

func countFlags(fs *flag.FlagSet) (n int) {
	fs.VisitAll(func(f *flag.Flag) {
		n++
	})
	return n
}

func isBoolFlag(f *flag.Flag) bool {
	b, ok := f.Value.(interface {
		IsBoolFlag() bool
	})
	return ok && b.IsBoolFlag()
}
