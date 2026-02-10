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

func newNilWeekFrame(reason string) error {
	return &NilWeekFrameError{reason: reason}
}

func (n *NilWeekFrameError) Error() string {
	return fmt.Sprintf("invalid weekframe: %q", n.reason)
}

type NilTimezoneError struct {
	reason string
}

func NewNilTimezoneError(reason string) error {
	return &NilTimezoneError{reason: reason}
}

func (n *NilTimezoneError) Error() string {
	return fmt.Sprintf("invalid timezone: %q", n.reason)
}
