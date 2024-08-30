package scalable

import (
	"context"
	"errors"

	"k8s.io/client-go/dynamic"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

var (
	timeout                      int64 = 30
	errNoReplicasSpecified             = errors.New("error: workload has no replicas set")
	errNoMinReplicasSpecified          = errors.New("error: workload has no minimum replicas set")
	errBoundOnScalingTargetValue       = errors.New("error: the target values for downscaling must be between 0 (or 1 if hpa is include-resources) and maxInt32")
)

// getResourceFunc is a function that gets a specific resource as a Workload
type getResourceFunc func(namespace string, clientset *kubernetes.Clientset, dynamicClient dynamic.Interface, ctx context.Context) ([]Workload, error)

// GetResource maps the resource name to a implementation specific getResourceFunc
var GetResource = map[string]getResourceFunc{
	"deployments":              getDeployments,
	"statefulsets":             getStatefulSets,
	"cronjobs":                 getCronJobs,
	"jobs":                     getJobs,
	"daemonsets":               getDaemonSets,
	"poddisruptionbudgets":     getPodDisruptionBudgets,
	"horizontalpodautoscalers": getHorizontalPodAutoscalers,
	"scaledobjects":            getScaledObjects,
}

// Workload is an interface for a scalable resource. It holds shared resource specific functions
type Workload interface {
	// GetAnnotations gets the annotations of the resource
	GetAnnotations() map[string]string
	// GetNamespace gets the namespace of the resource
	GetNamespace() string
	// GetName gets the name of the resource
	GetName() string
	// GetUID gets the uid of the workload
	GetUID() types.UID
	// GetObjectKind gets the ObjectKind of the workload
	GetObjectKind() schema.ObjectKind
	// SetAnnotations sets the annotations on the resource. Changes won't be made on kubernetes until update() is called
	SetAnnotations(annotations map[string]string)
	// Update updates the resource with all changes made to it. It should only be called once on a resource
	Update(clientset *kubernetes.Clientset, dynamicClient dynamic.Interface, ctx context.Context) error
	// ScaleUp scales up the workload
	ScaleUp() error
	// ScaleDown scales down the workload
	ScaleDown(downscaleReplicas int) error
}
