package scli

import (
	"errors"
	"fmt"
)

var (
	ErrUnparsed         = errors.New("command tree is unparsed, can't run")
	ErrInvalidArguments = errors.New("invalid arguments")
)

type NoExecError struct {
	Command *Command
}

func (e NoExecError) Error() string {
	return fmt.Sprintf("terminal command (%s) does not define a Exec function", e.Command.Name())
}
