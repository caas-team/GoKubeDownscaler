package kubernetes

import (
	"errors"
)

type InvalidConfiguration struct {
	reason string
	cause  string
}

func (e *InvalidConfiguration) create() error {
	return errors.New(e.reason)
}

func (e *InvalidConfiguration) modifyReason(r string) {
	e.reason = r
}

func (e *InvalidConfiguration) modifyCause(c string) {
	e.cause = c
}

func newInvalidConfiguration(r string, c string) *InvalidConfiguration {
	nIC := InvalidConfiguration{reason: r, cause: c}
	return &nIC
}
