package kubernetes

import "errors"

type ResourceNotSupported struct {
	reason string
	cause  string
}

func (r *ResourceNotSupported) create() error {
	return errors.New(r.reason)
}
