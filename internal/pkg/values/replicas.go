package values

import (
	"errors"
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

type StringReplicas string

func (s StringReplicas) AsBool() (bool, error) {
	return false, newInvalidReplicaTypeError("string replicas cannot be converted to bool", s.String())
}

func (s StringReplicas) String() string { return string(s) }

func (s StringReplicas) AsInt32() (int32, error) {
	return 0, newInvalidReplicaTypeError("string replicas cannot be converted to int32", s.String())
}

func (s StringReplicas) AsIntStr() intstr.IntOrString {
	return intstr.FromString(string(s))
}

type ReplicasValue struct {
	Replicas *Replicas
}

func (r *ReplicasValue) Set(value string) error {
	absoluteReplica, absoluteMatched, err := parseAbsoluteReplicas(value)
	if isUnexpectedReplicaParseError(err) {
		return err
	}

	if absoluteMatched {
		*r.Replicas = absoluteReplica
		return nil
	}

	percentageReplica, percentageMatched, err := parsePercentageReplicas(value)
	if isUnexpectedReplicaParseError(err) {
		return err
	}

	if percentageMatched {
		*r.Replicas = percentageReplica
		return nil
	}

	booleanReplica, booleanMatched, err := parseBooleanReplicas(value)
	if isUnexpectedReplicaParseError(err) {
		return err
	}

	if booleanMatched {
		*r.Replicas = booleanReplica
		return nil
	}

	if isAlpha(value) {
		*r.Replicas = StringReplicas(value)
		return nil
	}

	return newInvalidReplicaTypeError("invalid replica value", value)
}

// isUnexpectedReplicaParseError filters out parser no-match sentinel errors so Set can continue trying the next parser.
func isUnexpectedReplicaParseError(err error) bool {
	return err != nil && !errors.Is(err, ErrReplicaFormatNotMatched)
}

// parseAbsoluteReplicas tries to parse value as an AbsoluteReplicas.
// Returns (replica, true, nil) on success, (0, true, err) on validation error, (0, false, ErrReplicaFormatNotMatched) if not an integer.
func parseAbsoluteReplicas(value string) (AbsoluteReplicas, bool, error) {
	if parsedValue, err := strconv.ParseInt(value, 10, 32); err == nil {
		replica := AbsoluteReplicas(int32(parsedValue))
		if int(replica) < 0 && int(replica) != util.Undefined {
			return 0, true, newInvalidReplicaTypeError("downscale replicas has to be a positive integer", value)
		}

		return replica, true, nil
	}

	return 0, false, newReplicaFormatNotMatchedError()
}

// parsePercentageReplicas tries to parse value as a PercentageReplicas.
// Returns (replica, true, nil) on success, (0, true, err) on validation error, (0, false, ErrReplicaFormatNotMatched) if not a percentage.
func parsePercentageReplicas(value string) (PercentageReplicas, bool, error) {
	if !strings.HasSuffix(value, "%") {
		return 0, false, newReplicaFormatNotMatchedError()
	}

	if percentageValue, err := strconv.Atoi(strings.TrimSuffix(value, "%")); err == nil {
		if percentageValue < 0 || percentageValue > 100 {
			return 0, true, newInvalidReplicaTypeError("downscale replicas must be a percentage between 0% and 100%", value)
		}

		return PercentageReplicas(percentageValue), true, nil
	}

	return 0, false, newReplicaFormatNotMatchedError()
}

// parseBooleanReplicas tries to parse value as a BooleanReplicas.
// Returns (replica, true, nil) on success, (false, true, err) on parse error, (false, false, ErrReplicaFormatNotMatched) if not a boolean.
func parseBooleanReplicas(value string) (BooleanReplicas, bool, error) {
	if !isBooleanString(value) {
		return false, false, newReplicaFormatNotMatchedError()
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
