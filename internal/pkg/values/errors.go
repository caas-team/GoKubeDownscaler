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
	reason string
	value  string
}

func newInvalidSyntaxError(reason, value string) error {
	return &InvalidSyntaxError{reason: reason, value: value}
}

func (i *InvalidSyntaxError) Error() string {
	return fmt.Sprintf("error: %q, got %s.", i.reason, i.value)
}

type InvalidValueError struct {
	reason string
	value  string
}

func newInvalidValueError(reason, value string) error {
	return &InvalidSyntaxError{reason: reason, value: value}
}

func (i *InvalidValueError) Error() string {
	return fmt.Sprintf("error: %q, got %s.", i.reason, i.value)
}
