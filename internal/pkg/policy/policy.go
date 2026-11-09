// Package policy implements the matching and conflict resolution of DownscalePolicy resources.
package policy

import (
	"fmt"
	"slices"
	"strings"

	"github.com/caas-team/gokubedownscaler/api/v1alpha1"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Target is a workload a policy can be applied to.
type Target interface {
	GetName() string
	GetLabels() map[string]string
	GroupVersionKind() schema.GroupVersionKind
}

// Policy is a parsed DownscalePolicy, ready to be matched against workloads.
type Policy struct {
	// Object is the DownscalePolicy resource.
	Object *v1alpha1.DownscalePolicy
	// Scope holds the parsed scaling values of the policy. It is only usable when Err is nil.
	Scope *values.Scope
	// Err is set when the policy is invalid and must not be applied.
	Err error

	selector labels.Selector
}

// New creates a Policy from its resource, the scope parsed from its values and the error that occurred while parsing them.
func New(object *v1alpha1.DownscalePolicy, scope *values.Scope, parseErr error) *Policy {
	policy := &Policy{Object: object, Scope: scope, Err: parseErr}

	if object.Spec.Selector == nil {
		return policy
	}

	selector, err := metav1.LabelSelectorAsSelector(&object.Spec.Selector.LabelSelector)
	if err != nil {
		if policy.Err == nil {
			policy.Err = fmt.Errorf("invalid label selector: %w", err)
		}

		return policy
	}

	policy.selector = selector

	return policy
}

// Key returns the "namespace/name" identifier of the policy.
func (p *Policy) Key() string {
	return p.Object.Namespace + "/" + p.Object.Name
}

// Valid reports whether the policy can be applied.
func (p *Policy) Valid() bool {
	return p.Err == nil
}

// Matches reports whether the policy applies to the target.
// The target is matched if it is listed in targetRefs or matches the selector.
// If both are omitted, every target is matched.
func (p *Policy) Matches(target Target) bool {
	spec := &p.Object.Spec

	if len(spec.TargetRefs) == 0 && spec.Selector == nil {
		return true
	}

	gvk := target.GroupVersionKind()

	for _, ref := range spec.TargetRefs {
		if string(ref.Group) == gvk.Group && string(ref.Kind) == gvk.Kind && string(ref.Name) == target.GetName() {
			return true
		}
	}

	return p.selectorMatches(target, gvk)
}

func (p *Policy) selectorMatches(target Target, gvk schema.GroupVersionKind) bool {
	if p.Object.Spec.Selector == nil || p.selector == nil {
		return false
	}

	if kinds := p.Object.Spec.Selector.Kinds; len(kinds) > 0 {
		kindMatches := slices.ContainsFunc(kinds, func(kind v1alpha1.DownscalePolicyKind) bool {
			return string(kind.Group) == gvk.Group && string(kind.Kind) == gvk.Kind
		})
		if !kindMatches {
			return false
		}
	}

	return p.selector.Matches(labels.Set(target.GetLabels()))
}

// Resolve returns the policy that applies to a workload matched by all given policies, together with the policies that lost.
// Conflicts are resolved following GEP-713: the oldest policy wins, on a tie the first one in alphabetical
// "namespace/name" order wins.
//
//nolint:nonamedreturns // named returns document which value is which
func Resolve(matching []*Policy) (winner *Policy, losers []*Policy) {
	if len(matching) == 0 {
		return nil, nil
	}

	sorted := slices.Clone(matching)
	slices.SortStableFunc(sorted, comparePrecedence)

	return sorted[0], sorted[1:]
}

func comparePrecedence(left, right *Policy) int {
	leftTime, rightTime := left.Object.CreationTimestamp.Time, right.Object.CreationTimestamp.Time
	if !leftTime.Equal(rightTime) {
		if leftTime.Before(rightTime) {
			return -1
		}

		return 1
	}

	return strings.Compare(left.Key(), right.Key())
}

// FindApplicable returns the valid policies matching the target, the one to apply and the ones that lost against it.
//
//nolint:nonamedreturns // named returns document which value is which
func FindApplicable(policies []*Policy, target Target) (winner *Policy, losers []*Policy) {
	matching := make([]*Policy, 0, len(policies))

	for _, policy := range policies {
		if !policy.Valid() || !policy.Matches(target) {
			continue
		}

		matching = append(matching, policy)
	}

	return Resolve(matching)
}
