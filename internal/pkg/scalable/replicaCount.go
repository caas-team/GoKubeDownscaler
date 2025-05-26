package scalable

import (
	"errors"
	"strconv"
	"strings"
)

type ReplicaCount interface {
	String() string
}

type AbsoluteReplicas struct {
	Value int32
}

func (a AbsoluteReplicas) String() string { return strconv.Itoa(int(a.Value)) }

type PercentageReplicas struct {
	Value string
}

func (p PercentageReplicas) String() string { return p.Value }

type ReplicaCountValue struct {
	ReplicaCount *ReplicaCount
}

// nolint: err113 // this is a value type, not a pointer
func (r *ReplicaCountValue) Set(value string) error {
	if v, err := strconv.ParseInt(value, 10, 32); err == nil {
		*r.ReplicaCount = AbsoluteReplicas{Value: int32(v)}
		return nil
	}

	if strings.HasSuffix(value, "%") {
		*r.ReplicaCount = PercentageReplicas{Value: value}
		return nil
	}

	return errors.New("invalid value for ReplicaCount: must be int or percentage (e.g. 3 or 50%%)")
}

func (r *ReplicaCountValue) String() string {
	if r.ReplicaCount == nil || *r.ReplicaCount == nil {
		return ""
	}

	return (*r.ReplicaCount).String()
}
