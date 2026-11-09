package policy

import (
	"errors"
	"testing"
	"time"

	"github.com/caas-team/gokubedownscaler/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var errTest = errors.New("test error")

type testTarget struct {
	name   string
	labels map[string]string
	gvk    schema.GroupVersionKind
}

func (t testTarget) GetName() string                           { return t.name }
func (t testTarget) GetLabels() map[string]string              { return t.labels }
func (t testTarget) GroupVersionKind() schema.GroupVersionKind { return t.gvk }

func newObject(name string, created time.Time, spec *v1alpha1.DownscalePolicySpec) *v1alpha1.DownscalePolicy {
	return &v1alpha1.DownscalePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "team-a",
			CreationTimestamp: metav1.NewTime(created),
			Generation:        1,
		},
		Spec: *spec,
	}
}

func TestPolicy_Matches(t *testing.T) {
	t.Parallel()

	deployment := testTarget{
		name:   "api",
		labels: map[string]string{"tier": "backend"},
		gvk:    schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
	}
	service := testTarget{
		name: "api",
		gvk:  schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Service"},
	}

	tests := []struct {
		name   string
		spec   v1alpha1.DownscalePolicySpec
		target testTarget
		want   bool
	}{
		{
			name:   "no targeting matches everything",
			spec:   v1alpha1.DownscalePolicySpec{},
			target: deployment,
			want:   true,
		},
		{
			name: "matching targetRef",
			spec: v1alpha1.DownscalePolicySpec{
				TargetRefs: []gatewayv1.LocalPolicyTargetReference{{Group: "apps", Kind: "Deployment", Name: "api"}},
			},
			target: deployment,
			want:   true,
		},
		{
			name: "targetRef with other name",
			spec: v1alpha1.DownscalePolicySpec{
				TargetRefs: []gatewayv1.LocalPolicyTargetReference{{Group: "apps", Kind: "Deployment", Name: "web"}},
			},
			target: deployment,
			want:   false,
		},
		{
			name: "targetRef with other group",
			spec: v1alpha1.DownscalePolicySpec{
				TargetRefs: []gatewayv1.LocalPolicyTargetReference{{Group: "apps.kruise.io", Kind: "Deployment", Name: "api"}},
			},
			target: deployment,
			want:   false,
		},
		{
			name: "targetRef with core group",
			spec: v1alpha1.DownscalePolicySpec{
				TargetRefs: []gatewayv1.LocalPolicyTargetReference{{Group: "", Kind: "Service", Name: "api"}},
			},
			target: service,
			want:   true,
		},
		{
			name: "empty selector matches everything",
			spec: v1alpha1.DownscalePolicySpec{
				Selector: &v1alpha1.DownscalePolicySelector{},
			},
			target: service,
			want:   true,
		},
		{
			name: "selector by kind",
			spec: v1alpha1.DownscalePolicySpec{
				Selector: &v1alpha1.DownscalePolicySelector{Kinds: []v1alpha1.DownscalePolicyKind{{Group: "apps", Kind: "Deployment"}}},
			},
			target: deployment,
			want:   true,
		},
		{
			name: "selector by other kind",
			spec: v1alpha1.DownscalePolicySpec{
				Selector: &v1alpha1.DownscalePolicySelector{Kinds: []v1alpha1.DownscalePolicyKind{{Group: "apps", Kind: "StatefulSet"}}},
			},
			target: deployment,
			want:   false,
		},
		{
			name: "selector by labels",
			spec: v1alpha1.DownscalePolicySpec{
				Selector: &v1alpha1.DownscalePolicySelector{
					LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{"tier": "backend"}},
				},
			},
			target: deployment,
			want:   true,
		},
		{
			name: "selector by other labels",
			spec: v1alpha1.DownscalePolicySpec{
				Selector: &v1alpha1.DownscalePolicySelector{
					LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{"tier": "frontend"}},
				},
			},
			target: deployment,
			want:   false,
		},
		{
			name: "selector by kind and labels requires both",
			spec: v1alpha1.DownscalePolicySpec{
				Selector: &v1alpha1.DownscalePolicySelector{
					Kinds:         []v1alpha1.DownscalePolicyKind{{Group: "", Kind: "Service"}},
					LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{"tier": "backend"}},
				},
			},
			target: deployment,
			want:   false,
		},
		{
			name: "targetRef or selector",
			spec: v1alpha1.DownscalePolicySpec{
				TargetRefs: []gatewayv1.LocalPolicyTargetReference{{Group: "apps", Kind: "Deployment", Name: "web"}},
				Selector: &v1alpha1.DownscalePolicySelector{
					LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{"tier": "backend"}},
				},
			},
			target: deployment,
			want:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			policy := New(newObject("policy", time.Now(), &test.spec), nil, nil)
			require.True(t, policy.Valid())
			assert.Equal(t, test.want, policy.Matches(test.target))
		})
	}
}

func TestNew_InvalidSelector(t *testing.T) {
	t.Parallel()

	policy := New(newObject("policy", time.Now(), &v1alpha1.DownscalePolicySpec{
		Selector: &v1alpha1.DownscalePolicySelector{
			LabelSelector: metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{Key: "tier", Operator: "Invalid"}},
			},
		},
	}), nil, nil)

	require.False(t, policy.Valid())
}

func TestResolve(t *testing.T) {
	t.Parallel()

	now := time.Now()
	oldest := New(newObject("z-oldest", now.Add(-2*time.Hour), &v1alpha1.DownscalePolicySpec{}), nil, nil)
	tieA := New(newObject("a-tie", now, &v1alpha1.DownscalePolicySpec{}), nil, nil)
	tieB := New(newObject("b-tie", now, &v1alpha1.DownscalePolicySpec{}), nil, nil)

	winner, losers := Resolve([]*Policy{tieB, tieA, oldest})
	assert.Equal(t, oldest, winner)
	assert.Equal(t, []*Policy{tieA, tieB}, losers)

	winner, losers = Resolve([]*Policy{tieB, tieA})
	assert.Equal(t, tieA, winner)
	assert.Equal(t, []*Policy{tieB}, losers)

	winner, losers = Resolve(nil)
	assert.Nil(t, winner)
	assert.Nil(t, losers)
}

func TestFindApplicable_IgnoresInvalidPolicies(t *testing.T) {
	t.Parallel()

	now := time.Now()
	invalid := New(newObject("invalid", now.Add(-time.Hour), &v1alpha1.DownscalePolicySpec{}), nil, errTest)
	valid := New(newObject("valid", now, &v1alpha1.DownscalePolicySpec{}), nil, nil)

	winner, losers := FindApplicable([]*Policy{invalid, valid}, testTarget{name: "api"})
	assert.Equal(t, valid, winner)
	assert.Empty(t, losers)
}

func TestTracker_PendingUpdates(t *testing.T) {
	t.Parallel()

	now := time.Now()
	older := New(newObject("older", now.Add(-time.Hour), &v1alpha1.DownscalePolicySpec{}), nil, nil)
	newer := New(newObject("newer", now, &v1alpha1.DownscalePolicySpec{}), nil, nil)
	invalid := New(newObject("invalid", now, &v1alpha1.DownscalePolicySpec{}), nil, errTest)

	tracker := NewTracker([]*Policy{older, newer, invalid})
	tracker.Record(older, []*Policy{newer}, "Deployment/api")
	tracker.Record(older, nil, "Deployment/web")

	updates := tracker.PendingUpdates()
	require.Len(t, updates, 3)

	statuses := map[string]v1alpha1.DownscalePolicyStatus{}
	for _, update := range updates {
		statuses[update.Policy.Object.Name] = update.Status
	}

	assert.Equal(t, int32(2), statuses["older"].MatchedWorkloads)
	assert.True(t, meta.IsStatusConditionTrue(statuses["older"].Conditions, v1alpha1.ConditionTypeAccepted))
	assert.Equal(t, int64(1), statuses["older"].ObservedGeneration)

	assert.Equal(t, int32(0), statuses["newer"].MatchedWorkloads)
	accepted := meta.FindStatusCondition(statuses["newer"].Conditions, v1alpha1.ConditionTypeAccepted)
	require.NotNil(t, accepted)
	assert.Equal(t, v1alpha1.ReasonConflicted, accepted.Reason)
	assert.Contains(t, accepted.Message, "Deployment/api (applied: older)")

	assert.True(t, meta.IsStatusConditionFalse(statuses["invalid"].Conditions, v1alpha1.ConditionTypeValid))
	assert.True(t, meta.IsStatusConditionFalse(statuses["invalid"].Conditions, v1alpha1.ConditionTypeAccepted))

	// Once the status is written, the same scan result must not produce another update.
	older.Object.Status = statuses["older"]
	newer.Object.Status = statuses["newer"]
	invalid.Object.Status = statuses["invalid"]

	tracker = NewTracker([]*Policy{older, newer, invalid})
	tracker.Record(older, []*Policy{newer}, "Deployment/api")
	tracker.Record(older, nil, "Deployment/web")

	assert.Empty(t, tracker.PendingUpdates())
}
