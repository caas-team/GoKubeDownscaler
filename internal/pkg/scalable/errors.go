package scalable

import (
	"fmt"
)

type NoReplicasError struct {
	kind string
	name string
}

func newNoReplicasError(kind, name string) error {
	return &NoReplicasError{kind: kind, name: name}
}

func (n *NoReplicasError) Error() string {
	return fmt.Sprintf("error: %q %q has no replicas set", n.kind, n.name)
}

type InvalidResourceError struct {
	resource string
}

func newInvalidResourceError(resource string) error {
	return &InvalidResourceError{resource: resource}
}

func (i *InvalidResourceError) Error() string {
	return fmt.Sprintf("error: specified rescource type %q is not supported", i.resource)
}

type UnsupportedAPIVersionError struct {
	apiVersion string
}

func newUnsupportedAPIVersionError(apiVersion string) error {
	return &UnsupportedAPIVersionError{apiVersion: apiVersion}
}

func (e *UnsupportedAPIVersionError) Error() string {
	return fmt.Sprintf("error: unsupported apiVersion %q", e.apiVersion)
}

type OriginalReplicasUnsetError struct {
	reason string
}

func newOriginalReplicasUnsetError(reason string) error {
	return &OriginalReplicasUnsetError{reason: reason}
}

func (o *OriginalReplicasUnsetError) Error() string {
	return o.reason
}

type ExpectTypeGotTypeError struct {
	expected any
	actual   any
}

func newExpectTypeGotTypeError(expected, actual any) error {
	return &ExpectTypeGotTypeError{
		expected: expected,
		actual:   actual,
	}
}

func (e *ExpectTypeGotTypeError) Error() string {
	return fmt.Sprintf("expected type %T, got %T", e.expected, e.actual)
}

type NilUnderlyingObjectError struct {
	workloadType string
}

func newNilUnderlyingObjectError(workloadType string) error {
	return &NilUnderlyingObjectError{workloadType: workloadType}
}

func (o *NilUnderlyingObjectError) Error() string {
	return o.workloadType + " not found"
}

type UnexpectedReplicasTypeError struct {
	valType   string
	kind      string
	namespace string
	name      string
}

func newUnexpectedReplicasTypeError(val any, kind, namespace, name string) error {
	return &UnexpectedReplicasTypeError{
		valType:   fmt.Sprintf("%T", val),
		kind:      kind,
		namespace: namespace,
		name:      name,
	}
}

func (e *UnexpectedReplicasTypeError) Error() string {
	return fmt.Sprintf("unexpected type %s for spec.replicas on %s %s/%s", e.valType, e.kind, e.namespace, e.name)
}
