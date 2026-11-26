package exit

import (
	"errors"
	"fmt"
	"testing"
)

func TestCode(t *testing.T) {
	if Code(nil) != 0 || Code(Refused("x")) != 1 || Code(Usage("x")) != 2 ||
		Code(errors.New("io")) != 2 || Code(fmt.Errorf("wrap: %w", Refused("x"))) != 1 {
		t.Fatal("wrong exit code mapping")
	}
}
