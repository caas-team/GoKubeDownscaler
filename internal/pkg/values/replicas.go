package values

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

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

type StatusReplicas string

func (s StatusReplicas) AsBool() (bool, error) {
	return false, newInvalidReplicaTypeError("status replicas cannot be converted to bool", s.String())
}

func (s StatusReplicas) String() string { return string(s) }

func (s StatusReplicas) AsInt32() (int32, error) {
	return 0, newInvalidReplicaTypeError("status replicas cannot be converted to int32", s.String())
}

func (s StatusReplicas) AsIntStr() intstr.IntOrString {
	return intstr.FromString(string(s))
}

type ReplicasValue struct {
	Replicas *Replicas
}

func (r *ReplicasValue) Set(value string) error {
	if replica, ok, err := parseAbsoluteReplicas(value); ok {
		if err != nil {
			return err
		}

		*r.Replicas = replica

		return nil
	}

	if replica, ok, err := parsePercentageReplicas(value); ok {
		if err != nil {
			return err
		}

		*r.Replicas = replica

		return nil
	}

	if replica, ok, err := parseBooleanReplicas(value); ok {
		if err != nil {
			return err
		}

		*r.Replicas = replica

		return nil
	}

	if isAlpha(value) {
		*r.Replicas = StatusReplicas(value)
		return nil
	}

	return newInvalidReplicaTypeError("invalid replica value", value)
}

// parseAbsoluteReplicas tries to parse value as an AbsoluteReplicas.
// Returns (replica, true, nil) on success, (nil, true, err) on validation error, (nil, false, nil) if not an integer.
func parseAbsoluteReplicas(value string) (AbsoluteReplicas, bool, error) {
	v, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, false, nil
	}

	replica := AbsoluteReplicas(int32(v))
	if int(replica) < 0 && int(replica) != util.Undefined {
		return 0, true, newInvalidReplicaTypeError("downscale replicas has to be a positive integer", value)
	}

	return replica, true, nil
}

// parsePercentageReplicas tries to parse value as a PercentageReplicas.
// Returns (replica, true, nil) on success, (nil, true, err) on validation error, (nil, false, nil) if not a percentage.
func parsePercentageReplicas(value string) (PercentageReplicas, bool, error) {
	if !strings.HasSuffix(value, "%") {
		return 0, false, nil
	}

	p, err := strconv.Atoi(strings.TrimSuffix(value, "%"))
	if err != nil {
		return 0, false, nil
	}

	if p < 0 || p > 100 {
		return 0, true, newInvalidReplicaTypeError("downscale replicas must be a percentage between 0% and 100%", value)
	}

	return PercentageReplicas(p), true, nil
}

// parseBooleanReplicas tries to parse value as a BooleanReplicas.
// Returns (replica, true, nil) on success, (nil, true, err) on parse error, (nil, false, nil) if not a boolean.
func parseBooleanReplicas(value string) (BooleanReplicas, bool, error) {
	if !isBooleanString(value) {
		return false, false, nil
	}

	parsed, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(value)))
	if err != nil {
		return false, true, newInvalidReplicaTypeError("invalid boolean replica value", value)
	}

	return BooleanReplicas(parsed), true, nil
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

func isAlpha(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}

	return true
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
