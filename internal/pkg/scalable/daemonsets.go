package scalable

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	labelMatchNone = "downscaler/match-none"
)

// getDaemonSets is the getResourceFunc for DaemonSets.
func getDaemonSets(name, namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	if name != "" {
		results := make([]Workload, 0, 1)

		daemonset, err := clientsets.Kubernetes.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get daemonset: %w", err)
		}

		results = append(results, &daemonSet{daemonset})

		return results, nil
	}

	daemonsets, err := clientsets.Kubernetes.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get daemonsets: %w", err)
	}

	results := make([]Workload, 0, len(daemonsets.Items))
	for i := range daemonsets.Items {
		results = append(results, &daemonSet{&daemonsets.Items[i]})
	}

	return results, nil
}

// daemonSet is a wrapper for apps/v1.DeamonSet to implement the Workload interface.
type daemonSet struct {
	*appsv1.DaemonSet
}

// ScaleUp scales the resource up.
func (d *daemonSet) ScaleUp() error {
	delete(d.Spec.Template.Spec.NodeSelector, labelMatchNone)
	return nil
}

// ScaleDown scales the resource down.
func (d *daemonSet) ScaleDown(_ int32) error {
	if d.Spec.Template.Spec.NodeSelector == nil {
		d.Spec.Template.Spec.NodeSelector = map[string]string{}
	}

	d.Spec.Template.Spec.NodeSelector[labelMatchNone] = "true"

	return nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (d *daemonSet) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.AppsV1().DaemonSets(d.Namespace).Update(ctx, d.DaemonSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update daemonset: %w", err)
	}

	return nil
}
