package kubernetes

import (
	"context"
	"errors"

	"k8s.io/client-go/kubernetes"
)

type getResourceFunc func(namespace string, clientset *kubernetes.Clientset, ctx context.Context) ([]ScalableResource, error)

var scalableResources map[string]getResourceFunc = map[string]getResourceFunc{
	"deployments": getDeployments,
}

var errNoReplicasSpecified = errors.New("error: workload has no replicas set")

type ScalableResource interface {
	// GetAnnotations gets the annotations of the resource
	GetAnnotations() map[string]string
	// GetNamespace gets the namespace of the resource
	GetNamespace() string
	// GetName gets the name of the resource
	GetName() string
	// SetAnnotations sets the annotations on the resource. Changes won't be made on kubernetes until update() is called
	SetAnnotations(annotations map[string]string)
	// setReplicas sets the amount of replicas on the resource. Changes won't be made on kubernetes until update() is called
	setReplicas(replicas int) error
	// getCurrentReplicas gets the current amount of replicas of the resource
	getCurrentReplicas() (int, error)
	// update updates the resource with all changes made to it. It should only be called once on a resource
	update(clientset *kubernetes.Clientset, ctx context.Context) error
}
