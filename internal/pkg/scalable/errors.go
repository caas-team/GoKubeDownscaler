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

type OriginalReplicasUnsetError struct {
	reason string
}

func newOriginalReplicasUnsetError(reason string) error {
	return &OriginalReplicasUnsetError{reason: reason}
}

func (o *OriginalReplicasUnsetError) Error() string {
	return o.reason
}

type UnexpectedOriginalReplicasError struct {
	allowedValues string
	actual        any
}

func newUnexpectedOriginalReplicasError(allowedValues string, actual any) error {
	return &UnexpectedOriginalReplicasError{allowedValues: allowedValues, actual: actual}
}

func (u *UnexpectedOriginalReplicasError) Error() string {
	return fmt.Sprintf("unexpected originalReplicas: allowed %q, got %v", u.allowedValues, u.actual)
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

type IngressClassNameNilError struct {
	namespace string
	name      string
}

func newIngressClassNameNilError(namespace, name string) error {
	return &IngressClassNameNilError{namespace: namespace, name: name}
}

func (e *IngressClassNameNilError) Error() string {
	return fmt.Sprintf("IngressClassName is nil for ingress %s / %s", e.namespace, e.name)
}
