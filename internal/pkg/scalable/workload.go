package scalable

import (
	"context"
	"fmt"

	argo "github.com/argoproj/argo-rollouts/pkg/client/clientset/versioned"
	keda "github.com/kedacore/keda/v2/pkg/generated/clientset/versioned"
	monitoring "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	zalando "github.com/zalando-incubator/stackset-controller/pkg/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// getResourceFunc is a function that gets a specific resource as a Workload.
type getResourceFunc func(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error)

// GetWorkloads gets all workloads of the given resource in the cluster.
func GetWorkloads(resource, namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	resourceFuncMap := map[string]getResourceFunc{
		"deployments":              getDeployments,
		"statefulsets":             getStatefulSets,
		"cronjobs":                 getCronJobs,
		"jobs":                     getJobs,
		"daemonsets":               getDaemonSets,
		"poddisruptionbudgets":     getPodDisruptionBudgets,
		"horizontalpodautoscalers": getHorizontalPodAutoscalers,
		"scaledobjects":            getScaledObjects,
		"rollouts":                 getRollouts,
		"stacks":                   getStacks,
		"prometheuses":             getPrometheuses,
	}

	resourceFunc, exists := resourceFuncMap[resource]
	if !exists {
		return nil, newInvalidResourceError(resource)
	}

	workloads, err := resourceFunc(namespace, clientsets, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get workloads of type %q: %w", resource, err)
	}

	return workloads, nil
}

// scalableResource provides all functions needed to scale any type of resource.
type scalableResource interface {
	// GetAnnotations gets the annotations of the resource
	GetAnnotations() map[string]string
	// GetNamespace gets the namespace of the resource
	GetNamespace() string
	// GetName gets the name of the resource
	GetName() string
	// GetUID gets the uid of the workload
	GetUID() types.UID
	// GetLabels gets the labels of the workload
	GetLabels() map[string]string
	// GetCreationTimestamp gets the creation timestamp of the workload
	GetCreationTimestamp() metav1.Time
	// SetAnnotations sets the annotations on the resource. Changes won't be made on Kubernetes until update() is called
	SetAnnotations(annotations map[string]string)
	// GroupVersionKind gets the group version kind of the workload
	GroupVersionKind() schema.GroupVersionKind
	// Reget regets the workload to ensure the latest state
	Reget(clientsets *Clientsets, ctx context.Context) error
}

// Workload provides all functions needed to scale the workload.
type Workload interface {
	scalableResource
	// Update updates the resource with all changes made to it. It should only be called once on a resource
	Update(clientsets *Clientsets, ctx context.Context) error
	// ScaleUp scales up the workload
	ScaleUp() error
	// ScaleDown scales down the workload
	ScaleDown(downscaleReplicas int32) error
}

type Clientsets struct {
	Kubernetes *kubernetes.Clientset
	Keda       *keda.Clientset
	Argo       *argo.Clientset
	Zalando    *zalando.Clientset
	Monitoring *monitoring.Clientset
}
