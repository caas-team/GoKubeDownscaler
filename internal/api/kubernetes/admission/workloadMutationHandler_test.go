package admission

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	client "github.com/caas-team/gokubedownscaler/internal/api/kubernetes"
	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
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

func newAdmissionRequests(t *testing.T, uid, kind, namespace string, rawJSON []byte) *http.Request {
	t.Helper()

	admissionReview := admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			UID:       types.UID(uid),
			Kind:      metav1.GroupVersionKind{Kind: kind},
			Namespace: namespace,
			Object: k8sruntime.RawExtension{
				Raw: rawJSON,
			},
		},
	}
	body, err := json.Marshal(admissionReview)
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodPost, "/validate-workloads", bytes.NewBuffer(body))
	request.Header.Set("Content-Type", "application/json")

	return request
}

func newDeploymentRequestWithoutLabels(t *testing.T, namespace string) *http.Request {
	t.Helper()

	return newAdmissionRequests(t, "valid-uid", "Deployment", namespace, []byte(`{
		"apiVersion":"apps/v1",
		"kind":"Deployment",
		"metadata":{
			"name":"test-deploy",
			"namespace":"`+namespace+`"
		},
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

// nolint: unparam//in the future it could be useful to have this parameter
func newDeploymentRequestWithLabels(t *testing.T, namespace string) *http.Request {
	t.Helper()

	return newAdmissionRequests(t, "valid-uid", "Deployment", namespace, []byte(`{
		"apiVersion":"apps/v1",
		"kind":"Deployment",
		"metadata":{
			"name":"test-deploy",
			"namespace":"`+namespace+`",
			"labels": {"app":"demo"}
		},
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

func newDeploymentRequestWithExcludeAnnotationTrue(t *testing.T, namespace string) *http.Request {
	t.Helper()

	return newAdmissionRequests(t, "valid-uid", "Deployment", namespace, []byte(`{
		"apiVersion":"apps/v1",
		"kind":"Deployment",
		"metadata":{
			"name":"test-deploy",
			"namespace":"`+namespace+`",
			"annotations":{"downscaler/exclude":"true"},
			"labels": {"app":"demo"}
		},
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

// buildScaledObjectFromBytes returns a scalable.Workload from raw JSON bytes for tests.
// nolint: ireturn //interface necessary to realize the test case
func buildScaledObjectFromBytes(t *testing.T, namespace string) scalable.Workload {
	t.Helper()

	rawSo := []byte(`{
		"apiVersion": "keda.sh/v1alpha1",
		"kind": "ScaledObject",
		"metadata": {
			"name": "test-deploy",
			"namespace": "` + namespace + `"
		},
		"spec": {
			"scaleTargetRef": {
				"name": "test-deploy",
				"kind": "Deployment",
				"apiVersion": "apps/v1"
			},
			"minReplicaCount": 1,
			"maxReplicaCount": 5
		}
	}`)

	soWorkload, err := scalable.ParseWorkloadFromRawObject("scaledobject", rawSo)
	if err != nil {
		t.Fatalf("failed to parse scaledobject from bytes: %v", err)
	}

	return soWorkload
}

func newHandlerWithMocks(mockClient *MockClient) *WorkloadMutationHandler {
	return NewWorkloadMutationHandler(
		mockClient,
		values.NewScope(), values.NewScope(), values.GetDefaultScope(),
		false, nil, &util.RegexList{regexp.MustCompile(".*")}, &util.RegexList{}, &util.RegexList{},
		map[string]struct{}{"deployments": {}, "scaledobjects": {}}, false,
		nil,
	)
}

//nolint:dupl // duplication could be improved later
func TestEvaluateMutation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		setupMocks      func(*MockClient)
		setupHandler    func(*WorkloadMutationHandler)
		request         func(*testing.T) *http.Request
		expectedMessage string
		expectedCode    int32
	}{
		{
			name: "Workload excluded because uptime",
			setupMocks: func(mockClient *MockClient) {
				scope := values.NewScope()
				scope.DownscaleReplicas = values.AbsoluteReplicas(0)
				_ = scope.ForceUptime.Set("always")

				mockClient.On("GetScaledObjects", "default", mock.Anything).Return([]scalable.Workload{}, nil)
				mockClient.On("GetNamespaceScope", "default", mock.Anything).Return(scope, nil)
			},
			setupHandler: func(h *WorkloadMutationHandler) { h.includeNamespaces = &[]string{"default"}; h.dryRun = true },
			request: func(t *testing.T) *http.Request {
				t.Helper()
				return newDeploymentRequestWithLabels(t, "default")
			},
			expectedMessage: "workload matches scaling up conditions",
			expectedCode:    http.StatusAccepted,
		},
		{
			name: "Workload mutated because downtime",
			setupMocks: func(mockClient *MockClient) {
				scope := values.NewScope()
				scope.DownscaleReplicas = values.AbsoluteReplicas(0)
				_ = scope.ForceDowntime.Set("always")

				mockClient.On("GetScaledObjects", "default", mock.Anything).Return([]scalable.Workload{}, nil)
				mockClient.On("GetNamespaceScope", "default", mock.Anything).Return(scope, nil)
			},
			setupHandler: func(h *WorkloadMutationHandler) { h.includeNamespaces = &[]string{"default"}; h.dryRun = true },
			request: func(t *testing.T) *http.Request {
				t.Helper()
				return newDeploymentRequestWithLabels(t, "default")
			},
			expectedMessage: "would have patched",
			expectedCode:    http.StatusAccepted,
		},
		{
			name: "Workload excluded by annotation",
			setupMocks: func(m *MockClient) {
				m.On("GetScaledObjects", "default", mock.Anything).Return([]scalable.Workload{}, nil)

				scope := values.GetDefaultScope()
				m.On("GetNamespaceScope", "default", mock.Anything).Return(scope, nil)
			},
			setupHandler: func(h *WorkloadMutationHandler) { h.includeNamespaces = &[]string{"default"} },
			request: func(t *testing.T) *http.Request {
				t.Helper()
				return newDeploymentRequestWithExcludeAnnotationTrue(t, "default")
			},
			expectedMessage: "workload is excluded from downscaling",
			expectedCode:    http.StatusAccepted,
		},
		{
			name: "Workload excluded by namespace scope",
			setupMocks: func(m *MockClient) {
				scope := values.GetDefaultScope()
				_ = scope.Exclude.Set("true")

				m.On("GetScaledObjects", "default", mock.Anything).Return([]scalable.Workload{}, nil)
				m.On("GetNamespaceScope", "default", mock.Anything).Return(scope, nil)
			},
			setupHandler: func(h *WorkloadMutationHandler) { h.includeNamespaces = &[]string{"default"} },
			request: func(t *testing.T) *http.Request {
				t.Helper()
				return newDeploymentRequestWithLabels(t, "default")
			},
			expectedMessage: "workload is excluded from downscaling",
			expectedCode:    http.StatusAccepted,
		},
		{
			name: "Workload excluded by exclude list",
			setupMocks: func(m *MockClient) {
				m.On("GetScaledObjects", "default", mock.Anything).Return([]scalable.Workload{}, nil)
			},
			setupHandler: func(h *WorkloadMutationHandler) {
				h.includeNamespaces = &[]string{"default"}
				h.excludeWorkloads = &util.RegexList{regexp.MustCompile("test-deploy")}
			},
			request: func(t *testing.T) *http.Request {
				t.Helper()
				return newDeploymentRequestWithLabels(t, "default")
			},
			expectedMessage: "workload is excluded from downscaling",
			expectedCode:    http.StatusAccepted,
		},
		{
			name: "Workload excluded by external scaling (keda)",
			setupMocks: func(m *MockClient) {
				scaledObject := buildScaledObjectFromBytes(t, "default")
				m.On("GetScaledObjects", "default", mock.Anything).Return([]scalable.Workload{scaledObject}, nil)
			},
			setupHandler: func(h *WorkloadMutationHandler) { h.includeNamespaces = &[]string{"default"} },
			request: func(t *testing.T) *http.Request {
				t.Helper()
				return newDeploymentRequestWithLabels(t, "default")
			},
			expectedMessage: "workload is excluded from downscaling",
			expectedCode:    http.StatusAccepted,
		},
		{
			name:         "Workload ignored because namespace not included",
			setupMocks:   func(m *MockClient) {},
			setupHandler: func(h *WorkloadMutationHandler) { h.includeNamespaces = &[]string{"other-namespace"} },
			request: func(t *testing.T) *http.Request {
				t.Helper()
				return newDeploymentRequestWithLabels(t, "default")
			},
			expectedMessage: "not in the list of included namespaces",
			expectedCode:    http.StatusAccepted,
		},
		{
			name: "Workload excluded because missing labels",
			setupMocks: func(m *MockClient) {
				m.On("GetScaledObjects", "default", mock.Anything).Return([]scalable.Workload{}, nil)

				scope := values.GetDefaultScope()
				m.On("GetNamespaceScope", "default", mock.Anything).Return(scope, nil)
			},
			setupHandler: func(h *WorkloadMutationHandler) { h.includeNamespaces = &[]string{"default"} },
			request: func(t *testing.T) *http.Request {
				t.Helper()
				return newDeploymentRequestWithoutLabels(t, "default")
			},
			expectedMessage: "workload is excluded from downscaling",
			expectedCode:    http.StatusAccepted,
		},
		{
			name: "Workload excluded by exclude list (explicit workload)",
			setupMocks: func(m *MockClient) {
				m.On("GetScaledObjects", "default", mock.Anything).Return([]scalable.Workload{}, nil)
			},
			setupHandler: func(h *WorkloadMutationHandler) {
				h.includeNamespaces = &[]string{"default"}
				excluded := util.RegexList{regexp.MustCompile("test-deploy")}
				h.excludeWorkloads = &excluded
			},
			request: func(t *testing.T) *http.Request {
				t.Helper()
				return newDeploymentRequestWithLabels(t, "default")
			},
			expectedMessage: "workload is excluded from downscaling",
			expectedCode:    http.StatusAccepted,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &MockClient{}
			handler := newHandlerWithMocks(mockClient)

			if testCase.setupMocks != nil {
				tt := testCase
				tt.setupMocks(mockClient)
			}

			if testCase.setupHandler != nil {
				tt := testCase
				tt.setupHandler(handler)
			}

			req := testCase.request(t)
			input, _ := parseAdmissionReviewFromRequest(req)
			workload, _ := scalable.ParseWorkloadFromRawObject("deployment", input.Request.Object.Raw)

			resp, err := handler.evaluateWorkloadMutation(context.Background(), workload, input, false)
			require.NoError(t, err)
			require.Equal(t, testCase.expectedCode, resp.Response.Result.Code)
			require.Contains(t, resp.Response.Result.Message, testCase.expectedMessage)
		})
	}
}
