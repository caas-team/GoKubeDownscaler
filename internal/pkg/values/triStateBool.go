package values

import (
	"fmt"
	"strconv"
)

// triStateBool represents a boolean with an additional isSet field
type triStateBool struct {
	isSet bool
	value bool
}

// Set sets the value and sets isSet to true
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
		return "undefined"
	}
	return fmt.Sprint(t.value)
}

// IsBoolFlag is there to make triStateBool implement flag.boolFlag, which lets the cli argument not need a value resulting in a "true" value
func (b *triStateBool) IsBoolFlag() bool { return true }
