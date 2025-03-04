package util

import "fmt"

type Error struct {
	Message string
	Value   any
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %v", e.Message, e.Value)
}

type InvalidRegexError struct {
	ErrorType string
	Message   string
}

func (i *InvalidRegexError) Error() string {
	return fmt.Sprintf("%s: %s", i.ErrorType, i.Message)
}

// NewInvalidRegexError returns an error when a regex compilation fails.
func NewInvalidRegexError(errorType string, msg string) error {
	return &InvalidRegexError{errorType, msg}
}

type InvalidInt32Error struct {
	ErrorType string
	Message   string
}

func (i *InvalidInt32Error) Error() string {
	return fmt.Sprintf("%s: %s", i.ErrorType, i.Message)
}

// NewInvalidInt32Error returns an error when an int32 parsing fails.
func NewInvalidInt32Error(errorType string, msg string) error {
	return &InvalidInt32Error{errorType, msg}
}

type InvalidDurationError struct {
	ErrorType string
	Message   string
}

func (i *InvalidDurationError) Error() string {
	return fmt.Sprintf("%s: %s", i.ErrorType, i.Message)
}

// NewInvalidDurationError returns an error when a duration parsing fails.
func NewInvalidDurationError(errorType string, msg string) error {
	return &InvalidDurationError{errorType, msg}
}

type EnvValueError struct {
	ErrorType string
	Message   string
}

func (i *EnvValueError) Error() string {
	return fmt.Sprintf("%s: %s", i.ErrorType, i.Message)
}

// NewEnvValueError return an error when setting an environment variable value fails.
func NewEnvValueError(errorType string, msg string) error {
	return &EnvValueError{errorType, msg}
}
