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
	AsInt32() (int32, error)
	AsBool() (bool, error)
}

type AbsoluteReplicas int32

func (a AbsoluteReplicas) String() string { return strconv.Itoa(int(a)) }

func (a AbsoluteReplicas) AsInt32() (int32, error) {
	return int32(a), nil
}

func (a AbsoluteReplicas) AsBool() (bool, error) {
	return false, newInvalidReplicaTypeError("absolute replicas cannot be converted to bool", a.String())
}

func (a AbsoluteReplicas) AsIntStr() intstr.IntOrString {
	return intstr.FromInt32(int32(a))
}

type PercentageReplicas int

func (p PercentageReplicas) String() string { return fmt.Sprintf("%d%%", p) }

func (p PercentageReplicas) AsInt32() (int32, error) {
	return 0, newInvalidReplicaTypeError("percentage replicas cannot be converted to int32", p.String())
}

func (p PercentageReplicas) AsBool() (bool, error) {
	return false, newInvalidReplicaTypeError("percentage replicas cannot be converted to bool", p.String())
}

func (p PercentageReplicas) AsIntStr() intstr.IntOrString {
	return intstr.FromString(strconv.Itoa(int(p)) + "%")
}

type BooleanReplicas bool

func (s BooleanReplicas) String() string {
	return strconv.FormatBool(bool(s))
}

func (s BooleanReplicas) AsInt32() (int32, error) {
	return 0, newInvalidReplicaTypeError("boolean replicas cannot be converted to int32", s.String())
}

func (s BooleanReplicas) AsBool() (bool, error) {
	return bool(s), nil
}

func (s BooleanReplicas) AsIntStr() intstr.IntOrString {
	return intstr.FromString(strconv.FormatBool(bool(s)))
}

type ReplicasValue struct {
	Replicas *Replicas
}

func (r *ReplicasValue) Set(value string) error {
	if v, err := strconv.ParseInt(value, 10, 32); err == nil {
		replica := AbsoluteReplicas(int32(v))

		if int(replica) < 0 && int(replica) != util.Undefined {
			return newInvalidReplicaTypeError(
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
				return newInvalidReplicaTypeError(
					"downscale replicas must be a percentage between 0% and 100%",
					value,
				)
			}

			*r.Replicas = replica

			return nil
		}
	}

	if isBooleanString(value) {
		valueLower := strings.ToLower(strings.TrimSpace(value))

		parsedBooleanValue, err := strconv.ParseBool(valueLower)
		if err != nil {
			return newInvalidReplicaTypeError("invalid boolean replica value", value)
		}

		replica := BooleanReplicas(parsedBooleanValue)

		*r.Replicas = replica

		return nil
	}

	return newInvalidReplicaTypeError("invalid replica value", value)
}

// NewReplicasFromIntOrStr parses a intstr.IntOrString to the correct specific replica type.
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
		return util.UndefinedString
	}

	return (*r.Replicas).String()
}

// isBooleanString checks if the string is a boolean value (true/false).
func isBooleanString(s string) bool {
	lower := strings.ToLower(s)
	return lower == "true" || lower == "false"
}
