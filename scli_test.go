package scli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"reflect"
	"testing"
)

func TestCommand_ParseAndRun(t *testing.T) {
	var (
		emptyFlags = flag.NewFlagSet("emptyFlags", flag.ContinueOnError)
		rootFlags  = flag.NewFlagSet("rootFlags", flag.ContinueOnError)
		_          = rootFlags.String("string", "", "string flag")
		_          = rootFlags.Bool("bool", false, "bool flag")
		_          = rootFlags.Int64("int", 0, "int flag")
		subFlags   = flag.NewFlagSet("subFlags", flag.ContinueOnError)
		_          = subFlags.String("sub", "", "this is a flag in sub command")
	)

	tests := []struct {
		Name          string
		Usage         string
		Subcommands   []*Command
		FlagSet       *flag.FlagSet
		ArgsValidator ArgsValidator
		Exec          func(ctx context.Context, args []string) error
		PassedArgs    []string
		WantedError   error
	}{
		{
			Name:          "Root Help Flag",
			ArgsValidator: NoArgs,
			FlagSet:       emptyFlags,
			Exec:          returnsNil,
			PassedArgs:    []string{"-h"},
			WantedError:   flag.ErrHelp,
		},
		{
			Name:          "Invalid Args",
			ArgsValidator: ExactArgs(1),
			FlagSet:       emptyFlags,
			Exec:          returnsNil,
			PassedArgs:    []string{},
			WantedError:   ErrInvalidArguments,
		},
		{
			Name:          "Exec Invalid Args",
			ArgsValidator: NoArgs,
			FlagSet:       emptyFlags,
			Exec: func(ctx context.Context, args []string) error {
				return ErrInvalidArguments
			},
			PassedArgs:  []string{},
			WantedError: ErrInvalidArguments,
		},
		{
			Name:          "Help Requested",
			ArgsValidator: NoArgs,
			FlagSet:       emptyFlags,
			Exec: func(ctx context.Context, args []string) error {
				return flag.ErrHelp
			},
			PassedArgs:  []string{},
			WantedError: flag.ErrHelp,
		},
		{
			Name:          "Root",
			ArgsValidator: NoArgs,
			FlagSet:       emptyFlags,
			Exec:          returnsNil,
			PassedArgs:    []string{},
		},
		{
			Name:          "Root Flags",
			ArgsValidator: NoArgs,
			FlagSet:       rootFlags,
			Exec:          expectedFlags(rootFlags, fPair{"string", "bar"}, fPair{"bool", true}, fPair{"int", 42}),
			PassedArgs:    []string{"-string", "bar", "-bool", "-int", "42"},
		},
		{
			Name:          "Root Args",
			ArgsValidator: ExactArgs(3),
			FlagSet:       rootFlags,
			Exec:          expectsArgs("42", "foo", "bar"),
			PassedArgs:    []string{"42", "foo", "bar"},
		},
		{
			Name:          "Root Flags Args",
			ArgsValidator: ExactArgs(3),
			FlagSet:       rootFlags,
			Exec: combineExecs(
				expectedFlags(rootFlags, fPair{"string", "bar"}, fPair{"bool", true}, fPair{"int", 42}),
				expectsArgs("42", "foo", "bar"),
			),
			PassedArgs: []string{"-string", "bar", "-bool", "-int", "42", "42", "foo", "bar"},
		},
		{
			Name:          "Root Flags -- Args",
			ArgsValidator: ExactArgs(3),
			FlagSet:       rootFlags,
			Exec: combineExecs(
				expectedFlags(rootFlags, fPair{"string", "bar"}, fPair{"bool", true}, fPair{"int", 42}),
				expectsArgs("42", "foo", "bar"),
			),
			PassedArgs: []string{"-string", "bar", "-bool", "-int", "42", "--", "42", "foo", "bar"},
		},
		{
			Name:          "Root Sub",
			ArgsValidator: NoArgs,
			FlagSet:       emptyFlags,
			Subcommands: []*Command{
				{
					Usage:         "sub",
					ShortHelp:     "short help",
					LongHelp:      "long help",
					ArgsValidator: NoArgs,
					FlagSet:       emptyFlags,
					Exec:          returnsNil,
				},
			},
			PassedArgs: []string{"sub"},
		},
		{
			Name:          "Aliased Sub",
			Usage:         "AliasedSubCommand",
			ArgsValidator: NoArgs,
			FlagSet:       emptyFlags,
			Subcommands: []*Command{
				{
					Usage:         "sub",
					Aliases:       []string{"foobar"},
					ShortHelp:     "short help",
					LongHelp:      "long help",
					ArgsValidator: NoArgs,
					FlagSet:       emptyFlags,
					Exec:          returnsNil,
				},
			},
			PassedArgs: []string{"foobar"},
		},
		{
			Name:          "Root Flags Sub Flags",
			ArgsValidator: NoArgs,
			FlagSet:       rootFlags,
			Subcommands: []*Command{
				{
					Usage:         "sub",
					ShortHelp:     "short help",
					LongHelp:      "long help",
					ArgsValidator: NoArgs,
					FlagSet:       subFlags,
					Exec: combineExecs(
						expectedFlags(rootFlags, fPair{"string", "bar"}, fPair{"bool", true}, fPair{"int", 42}),
						expectedFlags(subFlags, fPair{"sub", "foobar"}),
					),
				},
			},
			PassedArgs: []string{"-string", "bar", "-bool", "-int", "42", "sub", "-sub", "foobar"},
		},
		{
			Name:          "Root Flags Sub Args",
			ArgsValidator: NoArgs,
			FlagSet:       rootFlags,
			Subcommands: []*Command{
				{
					Usage:         "sub",
					ShortHelp:     "short help",
					LongHelp:      "long help",
					ArgsValidator: ExactArgs(2),
					FlagSet:       subFlags,
					Exec: combineExecs(
						expectedFlags(rootFlags, fPair{"string", "bar"}, fPair{"bool", true}, fPair{"int", 42}),
						expectsArgs("42", "foobar"),
					),
				},
			},
			PassedArgs: []string{"-string", "bar", "-bool", "-int", "42", "sub", "42", "foobar"},
		},
		{
			Name:          "Root Flags -- Sub Args",
			ArgsValidator: NoArgs,
			FlagSet:       rootFlags,
			Subcommands: []*Command{
				{
					Usage:         "sub",
					ShortHelp:     "short help",
					LongHelp:      "long help",
					ArgsValidator: ExactArgs(2),
					FlagSet:       subFlags,
					Exec: combineExecs(
						expectedFlags(rootFlags, fPair{"string", "bar"}, fPair{"bool", true}, fPair{"int", 42}),
						expectsArgs("42", "foobar"),
					),
				},
			},
			PassedArgs: []string{"-string", "bar", "-bool", "-int", "42", "sub", "--", "42", "foobar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			cmd := Command{
				Usage:         tt.Usage,
				ShortHelp:     fmt.Sprintf("%s short help string", tt.Name),
				LongHelp:      fmt.Sprintf("%s long help string", tt.Name),
				Subcommands:   tt.Subcommands,
				FlagSet:       tt.FlagSet,
				ArgsValidator: tt.ArgsValidator,
				Exec:          tt.Exec,
			}

			if err := cmd.ParseAndRun(context.Background(), tt.PassedArgs); checkErr(err, tt.WantedError) {
				t.Errorf("RunAndParse() error %v, wantedError %v", err, tt.WantedError)
			}
		})
	}
}

func returnsNil(_ context.Context, _ []string) error {
	return nil
}

func combineExecs(execs ...func(ctx context.Context, args []string) error) func(ctx context.Context, args []string) error {
	return func(ctx context.Context, args []string) error {
		for _, f := range execs {
			if err := f(ctx, args); err != nil {
				return err
			}
		}
		return nil
	}
}

type fPair struct {
	Name  string
	Value any
}

func expectedFlags(flagSet *flag.FlagSet, flagPairs ...fPair) func(ctx context.Context, args []string) error {
	return func(ctx context.Context, args []string) error {
		fvps := make(map[string]flag.Value)

		flagSet.VisitAll(func(f *flag.Flag) {
			fvps[f.Name] = f.Value
		})

		for _, fp := range flagPairs {
			fv, ok := fvps[fp.Name]

			if !ok {
				return fmt.Errorf("missing flag for %s", fp.Name)
			}

			s := fmt.Sprintf("%v", fp.Value)
			if s != fv.String() {
				return fmt.Errorf("invalid flag value for %s", fp.Name)
			}
		}
		return nil
	}
}

func expectsArgs(wantedArgs ...string) func(ctx context.Context, args []string) error {
	return func(ctx context.Context, args []string) error {
		if !reflect.DeepEqual(wantedArgs, args) {
			return fmt.Errorf("expected args = %v, got = %v", wantedArgs, args)
		}
		return nil
	}
}

func checkErr(err, wantedErr error) bool {
	if wantedErr == nil && err == nil {
		return false
	}
	return !errors.Is(err, wantedErr)
}
