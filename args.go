package scli

import (
	"fmt"
	"strings"
)

// ArgsValidator is the function signature for representing an argument validator for Command's.
type ArgsValidator func(args []string) error

// NoArgs returns an error if any args are included.
func NoArgs(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("was expecting to receive no arguemetns received %d", len(args))
	}
	return nil
}

// MinArgs returns an error if there are not at least N args.
func MinArgs(n int) ArgsValidator {
	return func(args []string) error {
		if len(args) < n {
			return fmt.Errorf("requires at least %d arg(s), only received %d", n, len(args))
		}
		return nil
	}
}

// MaxArgs returns an error if there are more than N args.
func MaxArgs(n int) ArgsValidator {
	return func(args []string) error {
		if len(args) > n {
			return fmt.Errorf("requires at most %d arg(s), received %d", n, len(args))
		}
		return nil
	}
}

// ExactArgs returns an error unless there are exactly N args.
func ExactArgs(n int) ArgsValidator {
	return func(args []string) error {
		if len(args) != n {
			return fmt.Errorf("requires exactly %d arg(s), received %d", n, len(args))
		}
		return nil
	}
}

// RangeArgs returns an error if the number of args is not in the expected range.
func RangeArgs(min, max int) ArgsValidator {
	return func(args []string) error {
		l := len(args)
		if l < min || l > max {
			return fmt.Errorf("requires between %d and %d arg(s), received %d", min, max, l)
		}
		return nil
	}
}

// OnlyValidArgs returns an error if there are any args that are not contained in the validArgs slice.
func OnlyValidArgs(validArgs []string) ArgsValidator {
	if len(validArgs) == 0 {
		return nil
	}

	validSet := make(map[string]struct{}, len(validArgs))
	for _, arg := range validArgs {
		validSet[arg] = struct{}{}
	}

	return func(args []string) error {
		for _, arg := range args {
			if _, ok := validSet[arg]; !ok {
				// TODO: this wording needs to be improved
				return fmt.Errorf("requires valid arugmenets of %s, received %s", strings.Join(validArgs, ", "), arg)
			}
		}
		return nil
	}
}

// CombineValidator is used for combining multiple ArgsValidator's into one.
// It accepts multiple ArgsValidator functions and returns a single ArgsValidator,
// that checks all conditions in order they are passed.
func CombineValidator(validators ...ArgsValidator) ArgsValidator {
	return func(args []string) error {
		for _, v := range validators {
			if err := v(args); err != nil {
				return err
			}
		}
		return nil
	}
}
