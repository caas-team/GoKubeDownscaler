package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	client "github.com/caas-team/gokubedownscaler/internal/api/kubernetes"
	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

type MockClient struct {
	client.Client
	mock.Mock
}

func (m *MockClient) GetNamespaceScope(namespace string, ctx context.Context) (*values.Scope, error) {
	args := m.Called(namespace, ctx)
	return args.Get(0).(*values.Scope), args.Error(1)
}

func (m *MockClient) GetScaledObjects(namespace string, ctx context.Context) ([]scalable.Workload, error) {
	args := m.Called(namespace, ctx)
	return args.Get(0).([]scalable.Workload), args.Error(1)
}

type mockCertManager struct {
	Ready chan struct{}
}

//nolint:unparam //needed to simulate delay of this method
func (m *mockCertManager) AddCertificateRotation(_ context.Context, _ any) error {
	go func() {
		time.Sleep(10 * time.Millisecond)
		close(m.Ready)
	}()

	return nil
}

func TestCertRotationReady(t *testing.T) {
	t.Parallel()

	readyChannel := make(chan struct{})
	certManager := &mockCertManager{Ready: readyChannel}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := certManager.AddCertificateRotation(ctx, nil)
	require.NoError(t, err)

	select {
	case <-certManager.Ready:
		// success
	case <-ctx.Done():
		t.Fatal("certificate rotation did not signal ready")
	}
}

func TestServerConfigInitialization(t *testing.T) {
	mockKubeClient := &MockClient{}

	t.Parallel()

	serverConfiguration := &serverConfig{
		client:               mockKubeClient,
		clientNoDryRun:       mockKubeClient,
		scopeCli:             values.NewScope(),
		scopeEnv:             values.NewScope(),
		scopeDefault:         values.GetDefaultScope(),
		config:               &runtimeConfiguration{},
		includedResourcesSet: map[string]struct{}{"Deployment": {}},
	}

	require.NotNil(t, serverConfiguration.scopeCli)
	require.NotNil(t, serverConfiguration.scopeEnv)
	require.NotNil(t, serverConfiguration.scopeDefault)
	require.Contains(t, serverConfiguration.includedResourcesSet, "Deployment")
}

func TestToSetFunction(t *testing.T) {
	t.Parallel()

	inputSlice := []string{"a", "b", "c"}
	resultSet := toSet(inputSlice)

	require.Len(t, resultSet, 3)
	require.Contains(t, resultSet, "a")
	require.Contains(t, resultSet, "b")
	require.Contains(t, resultSet, "c")
}

func buildBadContentTypeRequest(t *testing.T) *http.Request {
	t.Helper()
	ctx := t.Context()
	request := httptest.NewRequestWithContext(ctx, http.MethodPost, "/validate-workloads", bytes.NewBufferString("not-a-json"))
	request.Header.Set("Content-Type", "text/plain")

	return request
}

func buildEmptyBodyRequest(t *testing.T) *http.Request {
	t.Helper()
	ctx := t.Context()
	request := httptest.NewRequestWithContext(ctx, http.MethodPost, "/validate-workloads", nil)
	request.Header.Set("Content-Type", "application/json")

	return request
}

func buildNilRequestField(t *testing.T) *http.Request {
	t.Helper()
	ctx := t.Context()
	admissionReview := admissionv1.AdmissionReview{}
	body, err := json.Marshal(admissionReview)
	require.NoError(t, err)

	request := httptest.NewRequestWithContext(ctx, http.MethodPost, "/validate-workloads", bytes.NewBuffer(body))
	request.Header.Set("Content-Type", "application/json")

	return request
}

func buildAdmissionRequest(t *testing.T, uid, kind string, rawJSON []byte) *http.Request {
	t.Helper()
	ctx := t.Context()
	admissionReview := admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			UID:  types.UID(uid),
			Kind: metav1.GroupVersionKind{Kind: kind},
			Object: k8sruntime.RawExtension{
				Raw: rawJSON,
			},
		},
	}
	body, err := json.Marshal(admissionReview)
	require.NoError(t, err)

	request := httptest.NewRequestWithContext(ctx, http.MethodPost, "/validate-workloads", bytes.NewBuffer(body))
	request.Header.Set("Content-Type", "application/json")

	return request
}

func buildBadWorkloadRequest(t *testing.T) *http.Request {
	t.Helper()
	return buildAdmissionRequest(t, "test-uid", "Pod", []byte(`{"metadata":{"name":"foo","namespace":"bar"}}`))
}

func buildValidDeploymentRequest(t *testing.T) *http.Request {
	t.Helper()

	return buildAdmissionRequest(t, "valid-uid", "Deployment", []byte(`{
		"apiVersion":"apps/v1",
		"kind":"Deployment",
		"metadata":{"name":"test-deploy","namespace":"default"},
		"spec":{
			"replicas":1,
			"selector":{"matchLabels":{"app":"demo"}},
			"template":{
				"metadata":{"labels":{"app":"demo"}},
				"spec":{"containers":[{"name":"nginx","image":"nginx:1.21"}]}
			}
		}
	}`))
}

func TestServeValidateWorkloads(t *testing.T) {
	t.Parallel()
	slog.SetLogLoggerLevel(slog.LevelDebug)

	testCases := []struct {
		name         string
		buildRequest func(t *testing.T) *http.Request
		wantCode     int
		wantContains string
		checkAllowed bool
	}{
		{"bad content type", buildBadContentTypeRequest, http.StatusBadRequest, "Content-Type", false},
		{"empty body", buildEmptyBodyRequest, http.StatusBadRequest, "empty", false},
		{"request field nil", buildNilRequestField, http.StatusBadRequest, "Request field is nil", false},
		{"parse workload error returns 500", buildBadWorkloadRequest, http.StatusInternalServerError, "", false},
		{"valid deployment request", buildValidDeploymentRequest, http.StatusOK, "", true},
	}

	for _, testCase := range testCases {
		// capture range variable
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockKubeClient := &MockClient{}
			mockKubeClient.On("GetNamespaceScope", "default", mock.Anything).Return(values.NewScope(), nil)
			mockKubeClient.On("GetScaledObjects", "default", mock.Anything).Return([]scalable.Workload{}, nil)

			serverConfiguration := &serverConfig{
				client:               mockKubeClient,
				clientNoDryRun:       mockKubeClient,
				scopeCli:             values.NewScope(),
				scopeEnv:             values.NewScope(),
				scopeDefault:         values.GetDefaultScope(),
				config:               &runtimeConfiguration{},
				includedResourcesSet: map[string]struct{}{},
			}

			request := testCase.buildRequest(t)
			responseRecorder := httptest.NewRecorder()

			serverConfiguration.serveValidateWorkloads(responseRecorder, request)

			require.Equal(t, testCase.wantCode, responseRecorder.Code)

			if testCase.wantContains != "" {
				require.Contains(t, responseRecorder.Body.String(), testCase.wantContains)
			}

			if testCase.checkAllowed && testCase.wantCode == http.StatusOK {
				var admissionResponse admissionv1.AdmissionReview
				require.NoError(t, json.Unmarshal(responseRecorder.Body.Bytes(), &admissionResponse))
				require.NotNil(t, admissionResponse.Response)
				require.True(t, admissionResponse.Response.Allowed)
			}
		})
	}
}
