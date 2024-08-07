package main

import (
	"context"
	"testing"

	client "github.com/caas-team/gokubedownscaler/internal/api/kubernetes"
	"github.com/caas-team/gokubedownscaler/internal/scalable"
	"github.com/caas-team/gokubedownscaler/internal/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockClient struct {
	client.Client
	mock.Mock
}

func (m *MockClient) GetNamespaceAnnotations(namespace string, ctx context.Context) (map[string]string, error) {
	args := m.Called(namespace, ctx)
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockClient) DownscaleWorkload(replicas int, workload scalable.Workload, ctx context.Context) error {
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

func TestScanWorkload(t *testing.T) {
	ctx := context.TODO()

	layerCli := values.NewLayer()
	layerEnv := values.NewLayer()

	layerCli.DownscaleReplicas = 0

	mockClient := new(MockClient)
	mockWorkload := new(MockWorkload)

	mockWorkload.On("GetNamespace").Return("test-namespace")
	mockWorkload.On("GetName").Return("test-workload")
	mockWorkload.On("GetAnnotations").Return(map[string]string{
		"downscaler/force-downtime": "true",
	})

	mockClient.On("GetNamespaceAnnotations", "test-namespace", ctx).Return(map[string]string{}, nil)
	mockClient.On("DownscaleWorkload", 0, mockWorkload, ctx).Return(nil)

	ok := scanWorkload(mockWorkload, mockClient, ctx, layerCli, layerEnv)

	assert.True(t, ok)

	mockClient.AssertExpectations(t)
	mockWorkload.AssertExpectations(t)
}
