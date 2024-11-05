package scalable

import (
	"errors"
)

// Errors for scalable package
var (
	errMinReplicasBoundsExceeded = errors.New("error: a HPAs minReplicas can only be set to int32 values larger than 1")
	errNoReplicasSpecified       = errors.New("error: workload has no replicas set")
)
