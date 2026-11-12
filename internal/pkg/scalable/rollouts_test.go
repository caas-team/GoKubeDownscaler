package scalable

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	argov1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	argo "github.com/argoproj/argo-rollouts/pkg/client/clientset/versioned"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func TestRollout_WorkloadRefDownscaleUpdateSucceeds(t *testing.T) {
	t.Parallel()

	requestBody := make(chan []byte, 1)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("failed to read update request: %v", err)
			http.Error(writer, "failed to read request", http.StatusInternalServerError)

			return
		}

		requestBody <- body

		var updated argov1alpha1.Rollout
		if err := json.Unmarshal(body, &updated); err != nil {
			t.Errorf("failed to decode update request: %v", err)
			http.Error(writer, "failed to decode request", http.StatusBadRequest)

			return
		}

		var rawUpdate struct {
			Spec struct {
				Template *struct {
					Spec struct {
						Containers json.RawMessage `json:"containers"`
					} `json:"spec"`
				} `json:"template"`
			} `json:"spec"`
		}
		if err := json.Unmarshal(body, &rawUpdate); err != nil {
			t.Errorf("failed to inspect update request: %v", err)
			http.Error(writer, "failed to inspect request", http.StatusBadRequest)

			return
		}

		if rawUpdate.Spec.Template != nil && string(rawUpdate.Spec.Template.Spec.Containers) == "null" {
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(writer).Encode(metav1.Status{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"},
				Status:   metav1.StatusFailure,
				Message:  `Rollout.argoproj.io "app-rollout" is invalid: spec.template.spec.containers: Required value`,
				Reason:   metav1.StatusReasonInvalid,
				Details: &metav1.StatusDetails{
					Name:  "app-rollout",
					Group: "argoproj.io",
					Kind:  "Rollout",
					Causes: []metav1.StatusCause{{
						Type:    metav1.CauseTypeFieldValueRequired,
						Message: "Required value",
						Field:   "spec.template.spec.containers",
					}},
				},
				Code: http.StatusUnprocessableEntity,
			})

			return
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(argov1alpha1.Rollout{
			TypeMeta: metav1.TypeMeta{
				APIVersion: argov1alpha1.SchemeGroupVersion.String(),
				Kind:       rolloutKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "app-rollout",
				Namespace: "ns1",
			},
			Spec: updated.Spec,
		})
	}))
	defer server.Close()

	argoClient, err := argo.NewForConfig(&rest.Config{Host: server.URL})
	require.NoError(t, err)

	replicas := int32(3)
	workload := &replicaScaledWorkload{&rollout{&argov1alpha1.Rollout{
		TypeMeta: metav1.TypeMeta{
			APIVersion: argov1alpha1.SchemeGroupVersion.String(),
			Kind:       rolloutKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-rollout",
			Namespace: "ns1",
		},
		Spec: argov1alpha1.RolloutSpec{
			Replicas: &replicas,
			WorkloadRef: &argov1alpha1.ObjectRef{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "app-deployment",
			},
		},
	}}}

	_, err = workload.ScaleDown(values.AbsoluteReplicas(0), nil)
	require.NoError(t, err)

	err = workload.Update(&Clientsets{Argo: argoClient}, context.Background())
	require.NoError(t, err)

	var updateRequest struct {
		Spec struct {
			Replicas    json.RawMessage         `json:"replicas"`
			WorkloadRef *argov1alpha1.ObjectRef `json:"workloadRef"`
			Template    *struct {
				Spec struct {
					Containers json.RawMessage `json:"containers"`
				} `json:"spec"`
			} `json:"template"`
		} `json:"spec"`
	}
	require.NoError(t, json.Unmarshal(<-requestBody, &updateRequest))
	assert.Equal(t, "0", string(updateRequest.Spec.Replicas))
	require.NotNil(t, updateRequest.Spec.WorkloadRef)
	assert.Equal(t, "Deployment", updateRequest.Spec.WorkloadRef.Kind)
	assert.Nil(t, updateRequest.Spec.Template)
}
