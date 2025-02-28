package main

import (
	"context"
	"log/slog"
	"testing"
	"time"

	client "github.com/caas-team/gokubedownscaler/internal/api/kubernetes"
	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MockClient struct {
	client.Client
	mock.Mock
}

func (m *MockClient) GetNamespaceAnnotations(namespace string, ctx context.Context) (map[string]string, error) {
	args := m.Called(namespace, ctx)
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockClient) DownscaleWorkload(replicas int32, workload scalable.Workload, ctx context.Context) error {
	args := m.Called(replicas, workload, ctx)
	return args.Error(0)
}

func (m *MockClient) UpscaleWorkload(workload scalable.Workload, ctx context.Context) error {
	args := m.Called(workload, ctx)
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

func TestScanWorkload(t *testing.T) {
	t.Parallel()

	slog.SetLogLoggerLevel(slog.LevelDebug)

	ctx := t.Context()

	layerCli := values.NewLayer()
	layerEnv := values.NewLayer()
	config := &util.RuntimeConfiguration{}

	layerCli.DownscaleReplicas = 0
	layerCli.GracePeriod = 15 * time.Minute

	mockClient := new(MockClient)
	mockWorkload := new(MockWorkload)

	mockWorkload.On("GetNamespace").Return("test-namespace")
	mockWorkload.On("GetName").Return("test-workload")
	mockWorkload.On("GetCreationTimestamp").Return(time.Now().Add(-layerCli.GracePeriod))
	mockWorkload.On("GetAnnotations").Return(map[string]string{
		"downscaler/force-downtime": "true",
	})

	mockClient.On("GetNamespaceAnnotations", "test-namespace", ctx).Return(map[string]string{}, nil)
	mockClient.On("DownscaleWorkload", int32(0), mockWorkload, ctx).Return(nil)

	err := scanWorkload(mockWorkload, mockClient, ctx, values.GetDefaultLayer(), &layerCli, &layerEnv, config)

	require.NoError(t, err)

	mockClient.AssertExpectations(t)
	mockWorkload.AssertExpectations(t)
}
