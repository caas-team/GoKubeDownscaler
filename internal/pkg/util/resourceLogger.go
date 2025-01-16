package util

import "context"

type ResourceLogger interface {
	// ErrorInvalidAnnotation adds an invalid annotation error on a resource
	ErrorInvalidAnnotation(id string, message string, ctx context.Context)
	// ErrorIncompatibleFields adds an incompatible fields error on a resource
	ErrorIncompatibleFields(message string, ctx context.Context)
}
