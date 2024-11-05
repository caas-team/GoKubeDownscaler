package kubernetes

import (
	"errors"
)

// Errors for kubernetes package
var (
	errRessourceNotSupported = errors.New("error: specified ressource type is not supported")
)
