package values

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type Replicas interface {
	String() string
	AsIntStr() intstr.IntOrString
	AsInt32() int32
	AsInt() int
}

type AbsoluteReplicas int32

func (a AbsoluteReplicas) String() string { return strconv.Itoa(int(a)) }

func (a AbsoluteReplicas) AsInt32() int32 {
	return int32(a)
}

func (a AbsoluteReplicas) AsIntStr() intstr.IntOrString {
	return intstr.FromInt32(int32(a))
}

func (a AbsoluteReplicas) AsInt() int {
	return int(a)
}

type PercentageReplicas int

func (p PercentageReplicas) String() string { return fmt.Sprintf("%d%%", p) }

func (p PercentageReplicas) AsInt32() int32 {
	const (
		minPercentage = 0
		maxPercentage = 100
	)

	percentage := p.AsInt() // AsInt returns int

	// Clamp the value between min and max
	if percentage < minPercentage {
		return int32(minPercentage)
	}

	if percentage > maxPercentage {
		return int32(maxPercentage)
	}

	return int32(percentage)
}

func (p PercentageReplicas) AsIntStr() intstr.IntOrString {
	return intstr.FromString(strconv.Itoa(int(p)) + "%")
}

func (p PercentageReplicas) AsInt() int {
	return int(p)
}

type ReplicasValue struct {
	Replicas *Replicas
}

// nolint: err113 // this is a value type, not a pointer
func (r *ReplicasValue) Set(value string) error {
	if v, err := strconv.ParseInt(value, 10, 32); err == nil {
		replica := AbsoluteReplicas(int32(v))

		if int(replica) < 0 && int(replica) != util.Undefined {
			return newInvalidValueError(
				"downscale replicas has to be a positive integer",
				value,
			)
		}

		*r.Replicas = replica

		return nil
	}

	if strings.HasSuffix(value, "%") {
		trimmed := strings.TrimSuffix(value, "%")
		if p, err := strconv.Atoi(trimmed); err == nil {
			replica := PercentageReplicas(p)

			if p < 0 || p > 100 {
				return newInvalidValueError(
					"downscale replicas must be a percentage between 0% and 100%",
					value,
				)
			}

			*r.Replicas = replica

			return nil
		}
	}

	return fmt.Errorf("invalid replica value: %s", value)
}

// NewReplicasFromIntOrStr
//
// nolint: ireturn // needed to return an interface
func NewReplicasFromIntOrStr(intOrString *intstr.IntOrString) Replicas {
	if intOrString == nil {
		return nil
	}

	switch intOrString.Type {
	case intstr.Int:
		return AbsoluteReplicas(intOrString.IntVal)
	case intstr.String:
		str := strings.TrimSuffix(intOrString.StrVal, "%")
		val, _ := strconv.Atoi(str)

		return PercentageReplicas(val)
	}

	return nil
}

func (r *ReplicasValue) String() string {
	if r.Replicas == nil || *r.Replicas == nil {
		return ""
	}

	return (*r.Replicas).String()
}
