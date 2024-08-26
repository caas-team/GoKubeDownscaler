package scalable

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// getDaemonSets is the getResourceFunc for DaemonSets
func getDaemonSets(namespace string, clientset *kubernetes.Clientset, _ dynamic.Interface, ctx context.Context) ([]Workload, error) {
	var results []Workload
	daemonsets, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get daemonsets: %w", err)
	}
	for _, item := range daemonsets.Items {
		results = append(results, daemonSet{&item})
	}
	return results, nil
}

// daemonSet is a wrapper for batch/v1.cronJob to implement the scalableResource interface
type daemonSet struct {
	*appsv1.DaemonSet
}

// SetNodeSelector applies a particular NodeSelector to the workload
func (d daemonSet) SetNodeSelector(key string, value string) {
	if d.Spec.Template.Spec.NodeSelector == nil {
		d.Spec.Template.Spec.NodeSelector = map[string]string{}
		d.Spec.Template.Spec.NodeSelector[key] = value
	}
	d.Spec.Template.Spec.NodeSelector[key] = value
}

// NodeSelectorExists check if a particular NodeSelector exists inside a workload
func (d daemonSet) NodeSelectorExists(key string, value string) (bool, error) {
	nodeSelector := d.Spec.Template.Spec.NodeSelector
	if v, ok := nodeSelector[key]; ok {
		if v == value {
			return true, nil
		}

		return false, fmt.Errorf("node selector key %q found but the searched value %q is different than the one searched for (%q)", key, value, v)

	}
	return false, nil
}

// RemoveNodeSelector remove a particular node selector from the workload
func (d daemonSet) RemoveNodeSelector(key string) error {
	nodeSelector := d.Spec.Template.Spec.NodeSelector
	if _, ok := nodeSelector[key]; ok {
		delete(nodeSelector, key)
		return nil
	}
	return fmt.Errorf("node selector key %q not found inside the resource", key)
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (d daemonSet) Update(clientset *kubernetes.Clientset, _ dynamic.Interface, ctx context.Context) error {
	_, err := clientset.AppsV1().DaemonSets(d.Namespace).Update(ctx, d.DaemonSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update daemonset: %w", err)
	}
	return nil
}
