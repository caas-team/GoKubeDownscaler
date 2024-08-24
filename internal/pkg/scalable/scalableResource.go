package scalable

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

var (
	timeout                int64 = 30
	errNoReplicasSpecified       = errors.New("error: workload has no replicas set")
	errNoSuspendSpecified        = errors.New("error: workload has no suspend specified")
)

// getResourceFunc is a function that gets a specific resource as a scalableResource
type getResourceFunc func(namespace string, clientset *kubernetes.Clientset, ctx context.Context) ([]Workload, error)

// GetResource maps the resource name to a implementation specific getResourceFunc
var GetResource = map[string]getResourceFunc{
	"deployments":  getDeployments,
	"statefulsets": getStatefulSets,
	"cronJobs":     getCronJobs,
	"jobs":         getJobs,
	"daemonsets":   getDaemonSets,
}

// Workload is a interface for a scalable resource. It holds all resource specific functions
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
	Update(clientset *kubernetes.Clientset, ctx context.Context) error
}

// AppWorkload is a child interface for Workload. It holds all resource specific functions for apps/v1 workloads such as deployments and statefulsets
type AppWorkload interface {
	Workload
	// SetReplicas sets the amount of replicas on the resource. Changes won't be made on kubernetes until update() is called
	SetReplicas(replicas int)
	// GetCurrentReplicas gets the current amount of replicas of the resource
	GetCurrentReplicas() (int, error)
}

// BatchWorkload is a child interface for Workload. It holds all resource specific functions for batch/v1 workloads suchs as jobs and cronjobs
type BatchWorkload interface {
	Workload
	// SetSuspend sets the amount of replicas on the resource. Changes won't be made on kubernetes until update() is called
	SetSuspend(suspend bool)
	// GetSuspend gets the current status of spec.Suspend
	GetSuspend() (bool, error)
}

// DaemonWorkload is a child interface for Workload. It holds all resource specific functions for daemonsets apps/v1 workloads suchs
type DaemonWorkload interface {
	Workload
	// SetNodeSelector sets a particular nodeSelector on the resource. Changes won't be made on kubernetes until update() is called
	SetNodeSelector(key string, value string)
	// RemoveNodeSelector removes a particular nodeSelector from the resource. Changes won't be made on kubernetes until update() is called
	RemoveNodeSelector(key string) error
	// NodeSelectorExists checks if a particular nodeSelector exists
	NodeSelectorExists(key string, value string) (bool, error)
}
