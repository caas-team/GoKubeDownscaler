package values

import (
	"fmt"
	"strconv"

	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
)

// triStateBool represents a boolean with an additional isSet field.
type triStateBool struct {
	isSet bool
	value bool
}

// Set sets the value and sets isSet to true.
func (t *triStateBool) Set(value string) error {
	var err error

	t.value, err = strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("failed to parse boolean value: %w", err)
	}

	t.isSet = true

	return nil
}

func (t *triStateBool) String() string {
	if !t.isSet {
		return util.UndefinedString
	}

	return strconv.FormatBool(t.value)
}

// IsBoolFlag is there to make triStateBool implement flag.boolFlag.
// This lets users use the flag without needing to specify a value.
func (t *triStateBool) IsBoolFlag() bool { return true }
