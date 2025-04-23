package values

import (
	"fmt"
)

type IncompatibalFieldsError struct {
	field1 string
	field2 string
}

func newIncompatibalFieldsError(field1, field2 string) error {
	return &IncompatibalFieldsError{field1: field1, field2: field2}
}

func (i *IncompatibalFieldsError) Error() string {
	return fmt.Sprintf("error: the fields %q and %q are incompatible", i.field1, i.field2)
}

type ValueNotSetError struct {
	field string
}

func newValueNotSetError(field string) error {
	return &ValueNotSetError{field: field}
}

func (v *ValueNotSetError) Error() string {
	return fmt.Sprintf("error: no value set for field %q", v.field)
}

type InvalidSyntaxError struct {
	field  string
	reason string
}

func newInvalidSyntaxError(field, reason string) error {
	return &InvalidSyntaxError{field: field, reason: reason}
}

func (i *InvalidSyntaxError) Error() string {
	return fmt.Sprintf("error: %q is invalid. %q.", i.field, i.reason)
}
