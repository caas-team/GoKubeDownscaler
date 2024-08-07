package scalable

import (
	"context"
	"errors"

	"k8s.io/client-go/kubernetes"
)

var (
	timeout                int64 = 30
	errNoReplicasSpecified       = errors.New("error: workload has no replicas set")
)

// getResourceFunc is a function that gets a specific resource as a scalableResource
type getResourceFunc func(namespace string, clientset *kubernetes.Clientset, ctx context.Context) ([]Workload, error)

// GetResource maps the resource name to a implementation specific getResourceFunc
var GetResource = map[string]getResourceFunc{
	"deployments": getDeployments,
}

// Workload is a interface for a scalable resource. It holds all resource specific functions
type Workload interface {
	// GetAnnotations gets the annotations of the resource
	GetAnnotations() map[string]string
	// GetNamespace gets the namespace of the resource
	GetNamespace() string
	// GetName gets the name of the resource
	GetName() string
	// SetAnnotations sets the annotations on the resource. Changes won't be made on kubernetes until update() is called
	SetAnnotations(annotations map[string]string)
	// SetReplicas sets the amount of replicas on the resource. Changes won't be made on kubernetes until update() is called
	SetReplicas(replicas int)
	// GetCurrentReplicas gets the current amount of replicas of the resource
	GetCurrentReplicas() (int, error)
	// Update updates the resource with all changes made to it. It should only be called once on a resource
	Update(clientset *kubernetes.Clientset, ctx context.Context) error
}
