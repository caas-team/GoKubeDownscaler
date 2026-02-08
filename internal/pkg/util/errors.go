package util

import "fmt"

type InvalidWeekFrameValueError struct {
	reason string
	value  string
}

func newInvalidWeekFrameValue(reason, value string) error {
	return &InvalidWeekFrameValueError{reason: reason, value: value}
}

func (i *InvalidWeekFrameValueError) Error() string {
	return fmt.Sprintf("invalid weekframe value: %q got: %q", i.reason, i.value)
}

type NilWeekFrameError struct {
	reason string
}

func newNilWeekframe(reason string) error {
	return &NilWeekFrameError{reason: reason}
}

func (i *NilWeekFrameError) Error() string {
	return fmt.Sprintf("invalid weekframe: %q", i.reason)
}
