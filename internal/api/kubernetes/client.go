package kubernetes

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	argo "github.com/argoproj/argo-rollouts/pkg/client/clientset/versioned"
	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
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
	// GetNamespaceScopes gets the namespace scope from the namespace annotations
	GetNamespaceScopes(workloads []scalable.Workload, ctx context.Context) (map[string]*values.Scope, error)
	// GetWorkloads gets all workloads of the specified resources for the specified namespaces
	GetWorkloads(namespaces []string, resourceTypes []string, ctx context.Context) ([]scalable.Workload, error)
	// RegetWorkload gets the workload again to ensure the latest state
	RegetWorkload(workload scalable.Workload, ctx context.Context) error
	// DownscaleWorkload downscales the workload to the specified replicas
	DownscaleWorkload(replicas values.Replicas, workload scalable.Workload, ctx context.Context) error
	// UpscaleWorkload upscales the workload to the original replicas
	UpscaleWorkload(workload scalable.Workload, ctx context.Context) error
	// CreateLease creates a new lease for the downscaler
	CreateLease(leaseName string) (*resourcelock.LeaseLock, error)
	// GetNamespaceAnnotations gets the annotations of the workload's namespace
	GetNamespaceAnnotations(namespace string, ctx context.Context) (map[string]string, error)
	// addEvent creates a new event on either a workload or a namespace
	addEvent(eventType, reason, identifier, message string, object *corev1.ObjectReference, ctx context.Context) error
	// GetChildrenWorkloads gets the children workloads of the specified workload
	GetChildrenWorkloads(workload scalable.Workload, ctx context.Context) ([]scalable.Workload, error)
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

// getNamespaceAnnotations gets the annotations of the workload's namespace.
func (c client) GetNamespaceAnnotations(namespace string, ctx context.Context) (map[string]string, error) {
	ns, err := c.clientsets.Kubernetes.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}

	return ns.Annotations, nil
}

// GetWorkloads gets all workloads of the specified resources for the specified namespaces.
func (c client) GetWorkloads(
	namespaces,
	resourceTypes []string,
	ctx context.Context,
) ([]scalable.Workload, error) {
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

// GetChildrenWorkloads gets the children workloads of the specified workload.
func (c client) GetChildrenWorkloads(workload scalable.Workload, ctx context.Context) ([]scalable.Workload, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if parent, ok := workload.(scalable.ParentWorkload); ok {
		slog.Debug(
			"getting children workloads for workload",
			"workload", workload.GetName(),
			"namespace", workload.GetNamespace(),
			"resourceType", workload.GroupVersionKind().Kind,
		)

		children, err := parent.GetChildren(ctx, c.clientsets)
		if err != nil {
			return nil, fmt.Errorf("failed to get children workloads: %w", err)
		}

		return children, nil
	}

	return nil, nil
}

// RegetWorkload gets the workload again to ensure the latest state.
func (c client) RegetWorkload(workload scalable.Workload, ctx context.Context) error {
	err := workload.Reget(c.clientsets, ctx)
	if err != nil {
		return fmt.Errorf("failed to get workload: %w", err)
	}

	return nil
}

// DownscaleWorkload downscales the workload to the specified replicas.
func (c client) DownscaleWorkload(replicas values.Replicas, workload scalable.Workload, ctx context.Context) error {
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

// addEvent creates or updates a new event on either a workload or a namespace.
func (c client) addEvent(
	eventType, reason, identifier, message string,
	object *corev1.ObjectReference, // ObjectReference passed directly
	ctx context.Context,
) error {
	if c.dryRun {
		// Dry run mode
		slog.Info("running in dry run mode, would have added an event",
			"objectKind", object.Kind,
			"namespace", object.Namespace,
			"name", object.Name,
			"eventType", eventType,
			"reason", reason,
			"id", identifier,
			"message", message,
		)

		return nil
	}

	hash := sha256.Sum256([]byte(fmt.Sprintf("%s.%s", identifier, message)))
	name := fmt.Sprintf("%s.%s.%x", object.Name, reason, hash)

	eventsClient := c.clientsets.Kubernetes.CoreV1().Events(object.Namespace)

	if event, err := eventsClient.Get(ctx, name, metav1.GetOptions{}); err == nil && event != nil {
		event.Count++
		event.LastTimestamp = metav1.Now()

		_, err := eventsClient.Update(ctx, event, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update event: %w", err)
		}

		return nil
	}

	_, err := eventsClient.Create(ctx, &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: object.Namespace,
		},
		InvolvedObject: *object,
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

func (c client) CreateLease(leaseName string) (*resourcelock.LeaseLock, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}

	leaseNamespace, err := getCurrentNamespace()
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace or running outside of cluster: %w", err)
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

	return lease, nil
}

// GetNamespaceScopes gets the namespace scopes from the namespace annotations.
func (c client) GetNamespaceScopes(workloads []scalable.Workload, ctx context.Context) (map[string]*values.Scope, error) {
	var waitGroup sync.WaitGroup

	namespaceSet := make(map[string]struct{})

	for _, workload := range workloads {
		if _, exists := namespaceSet[workload.GetNamespace()]; !exists {
			namespaceSet[workload.GetNamespace()] = struct{}{}

			slog.Debug("visited namespace", "namespace", workload.GetNamespace())
		}
	}

	namespaceScopes := make(map[string]*values.Scope, len(namespaceSet))
	errChan := make(chan error, len(namespaceSet))

	for namespace := range namespaceSet {
		namespaceScopes[namespace] = values.NewScope()
	}

	for namespace := range namespaceSet {
		waitGroup.Add(1)

		go func(namespace string, ctx context.Context) {
			defer waitGroup.Done()

			nsLogger := NewResourceLoggerForNamespace(c, namespace)

			slog.Debug("fetching namespace annotations", "namespace", namespace)

			annotations, err := c.GetNamespaceAnnotations(namespace, ctx)
			if err != nil {
				errChan <- fmt.Errorf("failed to get namespace annotations for namespace %s: %w", namespace, err)
				return
			}

			slog.Debug("parsing workload scope from annotations", "annotations", annotations, "namespace", namespace)

			err = namespaceScopes[namespace].GetScopeFromAnnotations(annotations, nsLogger, ctx)
			if err != nil {
				errChan <- fmt.Errorf("failed to parse scope from annotations for namespace %s: %w", namespace, err)
				return
			}

			slog.Debug("correctly parsed namespace annotations", "namespace", namespace, "annotations", annotations)
		}(namespace, ctx)
	}

	waitGroup.Wait()

	close(errChan)

	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	return namespaceScopes, nil
}
