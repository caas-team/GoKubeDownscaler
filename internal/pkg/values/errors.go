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
	return fmt.Sprintf("error: the fields %s and %s are incompatible", i.field1, i.field2)
}

type ValueNotSetError struct {
	field string
}

func newValueNotSetError(field string) error {
	return &ValueNotSetError{field: field}
}

func (v *ValueNotSetError) Error() string {
	return "error: no value set for field " + v.field
}

type InvalidConfigError struct {
	field string
}

func newInvalidConfigError(field string) error {
	return &InvalidConfigError{field: field}
}

func (i *InvalidConfigError) Error() string {
	return fmt.Sprintf("error: specified %s is invalid", i.field)
}
