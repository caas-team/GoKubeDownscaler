package main

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/caas-team/gokubedownscaler/api/v1alpha1"
	client "github.com/caas-team/gokubedownscaler/internal/api/kubernetes"
	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/policy"
	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type MockClient struct {
	client.Client
	mock.Mock
}

func (m *MockClient) GetNamespaceAnnotations(namespace string, ctx context.Context) (map[string]string, error) {
	args := m.Called(namespace, ctx)
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockClient) DownscaleWorkload(
	replicas values.Replicas,
	workload scalable.Workload,
	ctx context.Context,
	logger *slog.Logger,
) (*metrics.SavedResources, error) {
	args := m.Called(replicas, workload, ctx, logger)
	return args.Get(0).(*metrics.SavedResources), args.Error(1)
}

func (m *MockClient) UpscaleWorkload(workload scalable.Workload, ctx context.Context, logger *slog.Logger) error {
	args := m.Called(workload, ctx, logger)
	return args.Error(0)
}

type MockWorkload struct {
	scalable.Workload
	mock.Mock
}

func (m *MockWorkload) GetNamespace() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockWorkload) GetName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockWorkload) GetAnnotations() map[string]string {
	args := m.Called()
	return args.Get(0).(map[string]string)
}

func (m *MockWorkload) GetCreationTimestamp() v1.Time {
	args := m.Called()
	return v1.Time{Time: args.Get(0).(time.Time)}
}

func (m *MockWorkload) GetLabels() map[string]string {
	args := m.Called()
	return args.Get(0).(map[string]string)
}

func (m *MockWorkload) GroupVersionKind() schema.GroupVersionKind {
	args := m.Called()
	return args.Get(0).(schema.GroupVersionKind)
}

type noopResourceLogger struct{}

func (noopResourceLogger) ErrorInvalidAnnotation(string, string, context.Context) {}

func (noopResourceLogger) ErrorIncompatibleFields(string, context.Context) {}

func newTestPolicy(t *testing.T, name string, created time.Time, policyValues *v1alpha1.DownscalePolicyValues) *policy.Policy {
	t.Helper()

	object := &v1alpha1.DownscalePolicy{
		ObjectMeta: v1.ObjectMeta{Name: name, Namespace: "test-namespace", CreationTimestamp: v1.NewTime(created)},
		Spec:       v1alpha1.DownscalePolicySpec{DownscalePolicyValues: *policyValues},
	}

	scope := values.NewScope()
	err := scope.GetScopeFromAnnotations(values.PolicyValuesToAnnotations(policyValues), noopResourceLogger{}, nil, t.Context())
	require.NoError(t, err)

	return policy.New(object, scope, nil)
}

func TestScanWorkload_DownscalePolicy(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	now := time.Now()

	olderPolicy := newTestPolicy(t, "older", now.Add(-time.Hour), &v1alpha1.DownscalePolicyValues{
		ForceDowntime:     "true",
		DownscaleReplicas: "1",
	})
	newerPolicy := newTestPolicy(t, "newer", now, &v1alpha1.DownscalePolicyValues{
		ForceDowntime:     "true",
		DownscaleReplicas: "2",
	})
	policies := &downscalePolicies{
		enabled:     true,
		byNamespace: map[string][]*policy.Policy{"test-namespace": {newerPolicy, olderPolicy}},
		tracker:     policy.NewTracker([]*policy.Policy{newerPolicy, olderPolicy}),
	}

	mockClient := new(MockClient)
	mockWorkload := new(MockWorkload)

	mockWorkload.On("GetNamespace").Return("test-namespace")
	mockWorkload.On("GetName").Return("test-workload")
	mockWorkload.On("GetLabels").Return(map[string]string{})
	mockWorkload.On("GroupVersionKind").Return(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"})
	mockWorkload.On("GetCreationTimestamp").Return(now.Add(-time.Hour))
	mockWorkload.On("GetAnnotations").Return(map[string]string{})
	mockClient.On(
		"DownscaleWorkload",
		values.AbsoluteReplicas(1),
		mockWorkload,
		ctx,
		mock.AnythingOfType("*slog.Logger"),
	).Return(metrics.NewSavedResources(0, 0), nil)

	err := scanWorkload(
		mockWorkload,
		mockClient,
		ctx,
		values.GetDefaultScope(),
		values.NewScope(),
		values.NewScope(),
		map[string]*values.Scope{"test-namespace": values.NewScope()},
		policies,
		&metrics.NamespaceMetricsHolder{},
		&runtimeConfiguration{},
		workloadLogger(mockWorkload),
	)
	require.NoError(t, err)

	mockClient.AssertExpectations(t)

	updates := policies.tracker.PendingUpdates()
	require.Len(t, updates, 2)

	statuses := map[string]v1alpha1.DownscalePolicyStatus{}
	for _, update := range updates {
		statuses[update.Policy.Object.Name] = update.Status
	}

	require.Equal(t, int32(1), statuses["older"].MatchedWorkloads)
	require.Equal(t, int32(0), statuses["newer"].MatchedWorkloads)

	accepted := meta.FindStatusCondition(statuses["newer"].Conditions, v1alpha1.ConditionTypeAccepted)
	require.NotNil(t, accepted)
	require.Equal(t, v1alpha1.ReasonConflicted, accepted.Reason)
}

func TestScanWorkload(t *testing.T) {
	t.Parallel()

	slog.SetLogLoggerLevel(slog.LevelDebug)

	ctx := t.Context()

	scopeCli := values.NewScope()
	scopeEnv := values.NewScope()
	config := &runtimeConfiguration{}

	scopeCli.DownscaleReplicas = values.AbsoluteReplicas(0)
	scopeCli.GracePeriod = 15 * time.Minute

	namespaceScopes := map[string]*values.Scope{
		"test-namespace": {
			DownscaleReplicas: values.AbsoluteReplicas(0),
			GracePeriod:       15 * time.Minute,
		},
	}

	namespaceMetrics := &metrics.NamespaceMetricsHolder{}

	mockClient := new(MockClient)
	mockWorkload := new(MockWorkload)

	mockWorkload.On("GetNamespace").Return("test-namespace")
	mockWorkload.On("GetName").Return("test-workload")
	mockWorkload.On("GetCreationTimestamp").Return(time.Now().Add(-scopeCli.GracePeriod))
	mockWorkload.On("GetAnnotations").Return(map[string]string{
		"downscaler/force-downtime": "true",
	})
	mockClient.On(
		"DownscaleWorkload",
		values.AbsoluteReplicas(0),
		mockWorkload,
		ctx,
		mock.AnythingOfType("*slog.Logger")).Return(metrics.NewSavedResources(0, 0),
		nil,
	)
	err := scanWorkload(
		mockWorkload,
		mockClient,
		ctx,
		values.GetDefaultScope(),
		scopeCli,
		scopeEnv,
		namespaceScopes,
		&downscalePolicies{},
		namespaceMetrics,
		config,
		workloadLogger(mockWorkload),
	)

	require.NoError(t, err)

	mockClient.AssertExpectations(t)
	mockWorkload.AssertExpectations(t)
}
