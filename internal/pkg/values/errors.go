package values

import (
	"fmt"
)

type IncompatibalFieldsError struct {
	Field1 string
	Field2 string
}

func newIncompatibalFieldsError(field1, field2 string) error {
	return &IncompatibalFieldsError{Field1: field1, Field2: field2}
}

func (e *IncompatibalFieldsError) Error() string {
	return fmt.Sprintf("error: the fields %s and %s are incompatible", e.Field1, e.Field2)
}

type ValueNotSetError struct {
	Field string
}

func newValueNotSetError(field string) error {
	return &ValueNotSetError{Field: field}
}

func (e *ValueNotSetError) Error() string {
	return "error: no value set for field " + e.Field
}

type InvalidConfigError struct {
	Field string
}

func newInvalidConfigError(field string) error {
	return &InvalidConfigError{Field: field}
}

func (e *InvalidConfigError) Error() string {
	return fmt.Sprintf("error: specified %s is invalid", e.Field)
}
