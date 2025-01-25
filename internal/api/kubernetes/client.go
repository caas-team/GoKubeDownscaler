package kubernetes

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	argo "github.com/argoproj/argo-rollouts/pkg/client/clientset/versioned"
	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	keda "github.com/kedacore/keda/v2/pkg/generated/clientset/versioned"
	monitoring "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	zalando "github.com/zalando-incubator/stackset-controller/pkg/clientset"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	componentName = "kubedownscaler"
	timeout       = 30 * time.Second
)

// Client is an interface representing a high-level client to get and modify Kubernetes resources.
type Client interface {
	// GetNamespaceAnnotations gets the annotations of the workload's namespace
	GetNamespaceAnnotations(namespace string, ctx context.Context) (map[string]string, error)
	// GetWorkloads gets all workloads of the specified resources for the specified namespaces
	GetWorkloads(namespaces []string, resourceTypes []string, ctx context.Context) ([]scalable.Workload, error)
	// DownscaleWorkload downscales the workload to the specified replicas
	DownscaleWorkload(replicas int32, workload scalable.Workload, ctx context.Context) error
	// UpscaleWorkload upscales the workload to the original replicas
	UpscaleWorkload(workload scalable.Workload, ctx context.Context) error
	// CreateLease creates a new lease for the downscaler
	CreateLease(leaseName string, leaseNamespace string, ctx context.Context) (*resourcelock.LeaseLock, error)
	// addWorkloadEvent creates a new event on the workload
	addWorkloadEvent(eventType string, reason string, id string, message string, workload scalable.Workload, ctx context.Context) error
	// CreateOrUpdateLease creates or update the downscaler lease
	// CreateOrUpdateLease(ctx context.Context, leaseNamespace string, isLeader *atomic.Bool) error
	// DeleteLease deletes the downscaler lease
	// DeleteLease(ctx context.Context, leaseNamespace string, isLeader *atomic.Bool) error
}

// NewClient makes a new Client.
func NewClient(kubeconfig string, dryRun bool) (client, error) {
	var kubeclient client
	var clientsets scalable.Clientsets

	kubeclient.dryRun = dryRun

	config, err := getConfig(kubeconfig)
	if err != nil {
		return kubeclient, fmt.Errorf("failed to get config for Kubernetes: %w", err)
	}

	// set qps and burst rate limiting options. See https://kubernetes.io/docs/reference/config-api/apiserver-eventratelimit.v1alpha1/
	config.QPS = 500    // available queries per second, when unused will fill the burst buffer
	config.Burst = 1000 // the max size of the buffer of queries

	clientsets.Kubernetes, err = kubernetes.NewForConfig(config)
	if err != nil {
		return kubeclient, fmt.Errorf("failed to get clientset for Kubernetes resources: %w", err)
	}

	clientsets.Keda, err = keda.NewForConfig(config)
	if err != nil {
		return kubeclient, fmt.Errorf("failed to get clientset for keda resources: %w", err)
	}

	clientsets.Argo, err = argo.NewForConfig(config)
	if err != nil {
		return kubeclient, fmt.Errorf("failed to get clientset for argo resources: %w", err)
	}

	clientsets.Zalando, err = zalando.NewForConfig(config)
	if err != nil {
		return kubeclient, fmt.Errorf("failed to get clientset for zalando resources: %w", err)
	}

	clientsets.Monitoring, err = monitoring.NewForConfig(config)
	if err != nil {
		return kubeclient, fmt.Errorf("failed to get clientset for monitoring resources: %w", err)
	}

	kubeclient.clientsets = &clientsets

	return kubeclient, nil
}

// client is a Kubernetes client with downscaling specific functions.
type client struct {
	clientsets *scalable.Clientsets
	dryRun     bool
}

// GetNamespaceAnnotations gets the annotations of the workload's namespace.
func (c client) GetNamespaceAnnotations(namespace string, ctx context.Context) (map[string]string, error) {
	ns, err := c.clientsets.Kubernetes.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}

	return ns.Annotations, nil
}

// GetWorkloads gets all workloads of the specified resources for the specified namespaces.
func (c client) GetWorkloads(namespaces, resourceTypes []string, ctx context.Context) ([]scalable.Workload, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var results []scalable.Workload

	if namespaces == nil {
		namespaces = []string{""}
	}

	for _, namespace := range namespaces {
		for _, resourceType := range resourceTypes {
			slog.Debug("getting workloads from resource type", "resourceType", resourceType)

			workloads, err := scalable.GetWorkloads(strings.ToLower(resourceType), namespace, c.clientsets, ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get workloads: %w", err)
			}

			results = append(results, workloads...)
		}
	}

	return results, nil
}

// DownscaleWorkload downscales the workload to the specified replicas.
func (c client) DownscaleWorkload(replicas int32, workload scalable.Workload, ctx context.Context) error {
	err := workload.ScaleDown(replicas)
	if err != nil {
		return fmt.Errorf("failed to set the workload into a scaled down state: %w", err)
	}

	if c.dryRun {
		slog.Info(
			"running in dry run mode, would have sent update workload request to scale down workload",
			"workload", workload.GetName(),
			"namespace", workload.GetNamespace(),
		)

		return nil
	}

	err = workload.Update(c.clientsets, ctx)
	if err != nil {
		return fmt.Errorf("failed to update the workload: %w", err)
	}

	slog.Debug("successfully scaled down workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())

	return nil
}

// UpscaleWorkload upscales the workload to the original replicas.
func (c client) UpscaleWorkload(workload scalable.Workload, ctx context.Context) error {
	err := workload.ScaleUp()
	if err != nil {
		return fmt.Errorf("failed to set the workload into a scaled up state: %w", err)
	}

	if c.dryRun {
		slog.Info(
			"running in dry run mode, would have sent update workload request to scale up workload",
			"workload", workload.GetName(),
			"namespace", workload.GetNamespace(),
		)

		return nil
	}

	err = workload.Update(c.clientsets, ctx)
	if err != nil {
		return fmt.Errorf("failed to update the workload: %w", err)
	}

	slog.Debug("successfully scaled up workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())

	return nil
}

// addWorkloadEvent creates or updates a new event on the workload.
func (c client) addWorkloadEvent(eventType, reason, identifier, message string, workload scalable.Workload, ctx context.Context) error {
	if c.dryRun {
		slog.Info("running in dry run mode, would have added an event on workload",
			"workload", workload.GetName(),
			"namespace", workload.GetNamespace(),
			"eventType", eventType,
			"reason", reason,
			"id", identifier,
			"message", message,
		)

		return nil
	}

	hash := sha256.Sum256([]byte(fmt.Sprintf("%s.%s", identifier, message)))
	name := fmt.Sprintf("%s.%s.%x", workload.GetName(), reason, hash)
	eventsClient := c.clientsets.Kubernetes.CoreV1().Events(workload.GetNamespace())

	// check if event already exists
	if event, err := eventsClient.Get(ctx, name, metav1.GetOptions{}); err == nil && event != nil {
		// update event
		event.Count++
		event.LastTimestamp = metav1.Now()

		_, err := eventsClient.Update(ctx, event, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update event: %w", err)
		}

		return nil
	}

	// create event
	_, err := c.clientsets.Kubernetes.CoreV1().Events(workload.GetNamespace()).Create(ctx, &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: workload.GetNamespace(),
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:       workload.GroupVersionKind().Kind,
			Namespace:  workload.GetNamespace(),
			Name:       workload.GetName(),
			UID:        workload.GetUID(),
			APIVersion: workload.GroupVersionKind().GroupVersion().String(),
		},
		Reason:         reason,
		Message:        message,
		Type:           eventType,
		Source:         corev1.EventSource{Component: componentName},
		FirstTimestamp: metav1.Now(),
		LastTimestamp:  metav1.Now(),
		Count:          1,
	}, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}

	return nil
}

func (c client) CreateLease(leaseName string, leaseNamespace string, ctx context.Context) (*resourcelock.LeaseLock, error) {
	hostname, err := os.Hostname()
	if err != nil {
		slog.Error("failed to get hostname", "error", err)
		return nil, err
	}

	lease := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      leaseName,
			Namespace: leaseNamespace,
		},
		Client: c.clientsets.Kubernetes.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: hostname,
		},
	}

	return lease, fmt.Errorf("failed to create lease: %w", err)
}

/*
// CreateOrUpdateLease attempts to acquire and maintain a lease for leadership.
func (c client) CreateOrUpdateLease(ctx context.Context, leaseNamespace string, isLeader *atomic.Bool) error {
	// get hostname for holder identity
	holderIdentity, err := os.Hostname()
	if err != nil {
		slog.Error("failed to get hostname", "error", err)
		return err
	}

	leasesClient := c.clientsets.Kubernetes.CoordinationV1().Leases(leaseNamespace)
	leaseDurationSeconds := int32(leaseDuration.Seconds())

	for {
		// lease Object
		lease := &coordv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      leaseName,
				Namespace: leaseNamespace,
			},
			Spec: coordv1.LeaseSpec{
				HolderIdentity:       &holderIdentity,
				LeaseDurationSeconds: &leaseDurationSeconds,
				RenewTime:            &metav1.MicroTime{Time: time.Now()},
			},
		}

		// search for an existing lease inside the namespace
		existingLease, err := leasesClient.Get(ctx, leaseName, metav1.GetOptions{})
		if err != nil {
			// creates new lease if lease doesn't exist, and jump to the next iteration
			slog.Debug("creating new lease", "lease", leaseName, "namespace", leaseNamespace)
			_, err = leasesClient.Create(ctx, lease, metav1.CreateOptions{})
			if err != nil {
				slog.Error("failed to create lease", "error", err)
				time.Sleep(leaseCheckSleepDuration)
				continue
			}
			slog.Debug("acquired lease", "holder", holderIdentity, "namespace", leaseNamespace)
			isLeader.Store(true)
		} else {
			// check if the existing lease has expired or is held by another pod; if it is held by another pod jump to the next iteration
			if existingLease.Spec.RenewTime != nil &&
				time.Since(existingLease.Spec.RenewTime.Time) < leaseDuration {
				if *existingLease.Spec.HolderIdentity != holderIdentity {
					slog.Debug("lease already held by another", "holder", *existingLease.Spec.HolderIdentity)
					isLeader.Store(false)
					time.Sleep(leaseCheckSleepDuration)
					continue
				}
			}

			// update the lease if it is currently held by the current pod
			existingLease.Spec.HolderIdentity = &holderIdentity
			existingLease.Spec.RenewTime = &metav1.MicroTime{Time: time.Now()}
			_, err = leasesClient.Update(ctx, existingLease, metav1.UpdateOptions{})
			if err != nil {
				slog.Error("failed to update lease", "error", err)
				time.Sleep(leaseCheckSleepDuration)
				continue
			}
			slog.Debug("lease renewed", "holder", holderIdentity, "namespace", leaseNamespace)
			isLeader.Store(true)
		}

		// sleep before renewing
		time.Sleep(leaseCheckSleepDuration)
	}
}

func (c client) DeleteLease(ctx context.Context, leaseNamespace string, isLeader *atomic.Bool) error {
	leasesClient := c.clientsets.Kubernetes.CoordinationV1().Leases(leaseNamespace)

	err := leasesClient.Delete(ctx, leaseName, metav1.DeleteOptions{})
	if err != nil {
		slog.Error("failed to delete lease %s in namespace %s", leaseName, leaseNamespace)
		return err
	}

	isLeader.Store(false)
	slog.Debug("deleted lease %s in namespace %s", leaseName, leaseNamespace)
	return nil
}*/
