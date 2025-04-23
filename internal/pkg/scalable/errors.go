package scalable

import (
	"fmt"
)

type NoReplicasError struct {
	Kind string
	Name string
}

func newNoReplicasError(kind, name string) error {
	return &NoReplicasError{Kind: kind, Name: name}
}

func (e *NoReplicasError) Error() string {
	return fmt.Sprintf("error: %s %s has no replicas set", e.Kind, e.Name)
}

type InvalidResourceError struct {
	Resource string
}

func newInvalidResourceError(resource string) error {
	return &InvalidResourceError{Resource: resource}
}

func (e *InvalidResourceError) Error() string {
	return fmt.Sprintf("error: specified rescource type %s is not supported", e.Resource)
}

type OriginalReplicasUnsetError struct {
	reason string
}

func newOriginalReplicasUnsetError(reason string) error {
	return &OriginalReplicasUnsetError{reason: reason}
}

func (e *OriginalReplicasUnsetError) Error() string {
	return e.reason
}
