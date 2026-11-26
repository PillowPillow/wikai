package exit

import (
	"errors"
	"fmt"
)

type refusedError struct{ msg string }

func (e *refusedError) Error() string { return e.msg }

func Refused(format string, a ...any) error {
	return &refusedError{msg: fmt.Sprintf(format, a...)}
}

func Usage(format string, a ...any) error {
	return fmt.Errorf(format, a...)
}

func Code(err error) int {
	if err == nil {
		return 0
	}
	var r *refusedError
	if errors.As(err, &r) {
		return 1
	}
	return 2
}
