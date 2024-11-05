package kubernetes

import (
	"errors"
)

var errRessourceNotSupported = errors.New("error: specified ressource type is not supported")
