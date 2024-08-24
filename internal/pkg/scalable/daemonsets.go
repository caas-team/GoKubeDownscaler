package scalable

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// getDaemonSets is the getResourceFunc for DaemonSets
func getDaemonSets(namespace string, clientset *kubernetes.Clientset, ctx context.Context) ([]Workload, error) {
	var results []Workload
	daemonsets, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get daemonsets: %w", err)
	}
	for _, item := range daemonsets.Items {
		results = append(results, DaemonSet{&item})
	}
	return results, nil
}

// DaemonSet is a wrapper for batch/v1.CronJob to implement the scalableResource interface
type DaemonSet struct {
	*appsv1.DaemonSet
}

// SetNodeSelector applies a particular NodeSelector to the workload
func (d DaemonSet) SetNodeSelector(key string, value string) {
	nodeSelector := d.Spec.Template.Spec.NodeSelector
	nodeSelector[key] = value
}

// GetNodeSelector get a particular NodeSelector to the workload
func (d DaemonSet) NodeSelectorExists(key string, value string) (bool, error) {
	nodeSelector := d.Spec.Template.Spec.NodeSelector
	if v, ok := nodeSelector[key]; ok {
		if v == value {
			return true, nil
		} else {
			return false, fmt.Errorf("node selector key %q found but the searched value %q is different than the one searched for (%q)", key, value, v)
		}
	}
	return false, nil
}

// GetNodeSelector get a particular NodeSelector to the workload
func (d DaemonSet) RemoveNodeSelector(key string) error {
	nodeSelector := d.Spec.Template.Spec.NodeSelector
	if _, ok := nodeSelector[key]; ok {
		delete(nodeSelector, key)
		return nil
	}
	return fmt.Errorf("node selector key %q not found inside the resource", key)
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (d DaemonSet) Update(clientset *kubernetes.Clientset, ctx context.Context) error {
	_, err := clientset.AppsV1().DaemonSets(d.Namespace).Update(ctx, d.DaemonSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update daemonset: %w", err)
	}
	return nil
}
