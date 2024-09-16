package scalable

import (
	"context"
	"errors"

	keda "github.com/kedacore/keda/v2/pkg/generated/clientset/versioned"
	"k8s.io/client-go/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

var (
	timeout                      int64 = 30
	errNoReplicasSpecified             = errors.New("error: workload has no replicas set")
	errNoMinReplicasSpecified          = errors.New("error: workload has no minimum replicas set")
	errBoundOnScalingTargetValue       = errors.New("error: replicas can only be set to a 32-bit integer >= 0 (or >= 1 for HPAs)")
)

// getResourceFunc is a function that gets a specific resource as a Workload
type getResourceFunc func(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error)

// GetResource maps the resource name to an implementation specific getResourceFunc
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
	// GetLabels gets the labels of the workload
	GetLabels() map[string]string
	// GetCreationTimestamp gets the creation timestamp of the workload
	GetCreationTimestamp() metav1.Time
	// SetAnnotations sets the annotations on the resource. Changes won't be made on kubernetes until update() is called
	SetAnnotations(annotations map[string]string)
	// Update updates the resource with all changes made to it. It should only be called once on a resource
	Update(clientsets *Clientsets, ctx context.Context) error
	// ScaleUp scales up the workload
	ScaleUp() error
	// ScaleDown scales down the workload
	ScaleDown(downscaleReplicas int) error
}

type Clientsets struct {
	Kubernetes *kubernetes.Clientset
	Keda       *keda.Clientset
}
