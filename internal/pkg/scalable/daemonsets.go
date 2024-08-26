package scalable

import (
	"context"
	"fmt"
	"log/slog"

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

// setNodeSelector applies a particular NodeSelector to the workload
func (d daemonSet) setNodeSelector(key string, value string) {
	if d.Spec.Template.Spec.NodeSelector == nil {
		d.Spec.Template.Spec.NodeSelector = map[string]string{}
		d.Spec.Template.Spec.NodeSelector[key] = value
	}
	d.Spec.Template.Spec.NodeSelector[key] = value
}

// nodeSelectorExists check if a particular NodeSelector exists inside a workload
func (d daemonSet) nodeSelectorExists(key string, value string) (bool, error) {
	nodeSelector := d.Spec.Template.Spec.NodeSelector
	if v, ok := nodeSelector[key]; ok {
		if v == value {
			return true, nil
		}

		return false, fmt.Errorf("node selector key %q found but the searched value %q is different than the one searched for (%q)", key, value, v)

	}
	return false, nil
}

// removeNodeSelector remove a particular node selector from the workload
func (d daemonSet) removeNodeSelector(key string) error {
	nodeSelector := d.Spec.Template.Spec.NodeSelector
	if _, ok := nodeSelector[key]; ok {
		delete(nodeSelector, key)
		return nil
	}
	return fmt.Errorf("node selector key %q not found inside the resource", key)
}

// ScaleUp upscale the resource when the downscale period ends
func (d daemonSet) ScaleUp() error {
	nodeSelectorExists, err := d.nodeSelectorExists("kube-downscaler-non-existent", "true")
	if err != nil {
		return fmt.Errorf("failed to upscale the workload: %w. Make sure you didn't specify a kubedownscaler reserved node selector", err)
	}
	if !nodeSelectorExists {
		slog.Debug("workload is already upscaled, skipping", "workload", d.GetName(), "namespace", d.GetNamespace())
		return nil
	}

	err = d.removeNodeSelector("kube-downscaler-non-existent")
	if err != nil {
		return fmt.Errorf("failed to remove node selector from the workload: %w", err)
	}
	return nil
}

// ScaleDown downscale the resource when the downscale period starts
func (d daemonSet) ScaleDown(_ int) error {
	nodeSelectorExists, err := d.nodeSelectorExists("kube-downscaler-non-existent", "true")
	if err != nil {
		return fmt.Errorf("failed to downscale the workload: %w. Make sure you didn't specify a kubedownscaler reserved node selector", err)
	}
	if nodeSelectorExists {
		slog.Debug("workload is already downscaled, skipping", "workload", d.GetName(), "namespace", d.GetNamespace())
		return nil
	}

	d.setNodeSelector("kube-downscaler-non-existent", "true")
	return nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (d daemonSet) Update(clientset *kubernetes.Clientset, _ dynamic.Interface, ctx context.Context) error {
	_, err := clientset.AppsV1().DaemonSets(d.Namespace).Update(ctx, d.DaemonSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update daemonset: %w", err)
	}
	return nil
}
