package kubernetes

import (
	"context"
	"crypto/sha256"
	stdErrors "errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	actionsv1alpha1 "github.com/actions/actions-runner-controller/apis/actions.github.com/v1alpha1"
	argo "github.com/argoproj/argo-rollouts/pkg/client/clientset/versioned"
	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	keda "github.com/kedacore/keda/v2/pkg/generated/clientset/versioned"
	monitoring "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	zalando "github.com/zalando-incubator/stackset-controller/pkg/clientset"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	componentName = "kubedownscaler"
	timeout       = 30 * time.Second
)

var (
	ErrInvalidBurst = stdErrors.New("burst argument must greater than zero")
	ErrInvalidQPS   = stdErrors.New("qps argument can't be zero, it can either be a positive value " +
		"or a negative value to disable rate limiting")
)

// Client is an interface representing a high-level client to get and modify Kubernetes resources.
// nolint: interfacebloat // this interface is meant to represent a high-level client with multiple functions.
type Client interface {
	// GetNamespacesAsSet gets all namespaces or a specific list of namespace
	GetNamespacesAsSet() (map[string]struct{}, error)
	// GetNamespacesScopes gets the namespaces scopes from the namespaces annotations
	GetNamespacesScopes(workloads []scalable.Workload, ctx context.Context) (map[string]*values.Scope, error)
	// GetNamespaceScope gets the namespace scope from its annotations
	GetNamespaceScope(namespace string, ctx context.Context) (*values.Scope, error)
	// GetWorkloads gets all workloads of the specified resources for the specified namespaces
	GetWorkloads(namespaces []string, resourceTypes []string, ctx context.Context) ([]scalable.Workload, error)
	// RegetWorkload gets the workload again to ensure the latest state
	RegetWorkload(workload scalable.Workload, ctx context.Context) error
	// DownscaleWorkload downscales the workload to the specified replicas
	DownscaleWorkload(replicas values.Replicas, workload scalable.Workload, ctx context.Context) (*metrics.SavedResources, error)
	// UpscaleWorkload upscales the workload to the original replicas
	UpscaleWorkload(workload scalable.Workload, ctx context.Context) error
	// ensureSecret ensures that the secret used for storing TLS certificates exists
	ensureSecret(namespace, secretName string, ctx context.Context) (bool, error)
	// GetScaledObjects gets all scaledobjects in the specified namespace
	GetScaledObjects(namespace string, ctx context.Context) ([]scalable.Workload, error)
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
func NewClient(kubeconfig string, dryRun bool, qps float64, burst int) (client, error) {
	var kubeclient client

	var clientsets scalable.Clientsets
	var scheme *runtime.Scheme

	kubeclient.dryRun = dryRun

	config, err := getConfig(kubeconfig)
	if err != nil {
		return kubeclient, fmt.Errorf("failed to get config for Kubernetes: %w", err)
	}

	if burst <= 0 {
		return kubeclient, fmt.Errorf("%w: got %d", ErrInvalidBurst, burst)
	}

	if qps == 0 {
		return kubeclient, fmt.Errorf("%w", ErrInvalidQPS)
	}

	// set qps and burst rate limiting options. See https://kubernetes.io/docs/reference/config-api/apiserver-eventratelimit.v1alpha1/
	config.QPS = float32(qps) // available queries per second, when unused will fill the burst buffer
	config.Burst = burst      // the max size of the buffer of queries

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

	scheme, err = NewScheme()
	if err != nil {
		return kubeclient, fmt.Errorf("failed to build scheme: %w", err)
	}

	clientsets.Client, err = ctrlclient.New(config, ctrlclient.Options{
		Scheme: scheme,
	})
	if err != nil {
		return kubeclient, fmt.Errorf("failed to get controller runtime client: %w", err)
	}

	kubeclient.clientsets = &clientsets

	return kubeclient, nil
}

// NewScheme creates a new runtime.Scheme with all needed APIs registered.
func NewScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()

	// GitHub Actions Runner Controller CRDs
	err := actionsv1alpha1.AddToScheme(scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to add a scheme to generic client: %w", err)
	}

	return scheme, nil
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
//

func (c client) DownscaleWorkload(
	replicas values.Replicas,
	workload scalable.Workload,
	ctx context.Context,
) (*metrics.SavedResources, error) {
	savedResources, err := workload.ScaleDown(replicas)
	if err != nil {
		return metrics.NewSavedResources(0, 0), fmt.Errorf("failed to set the workload into a scaled down state: %w", err)
	}

	if c.dryRun {
		slog.Info(
			"running in dry run mode, would have sent update workload request to scale down workload",
			"workload", workload.GetName(),
			"namespace", workload.GetNamespace(),
		)

		return metrics.NewSavedResources(0, 0), nil
	}

	err = workload.Update(c.clientsets, ctx)
	if err != nil {
		return metrics.NewSavedResources(0, 0), fmt.Errorf("failed to update the workload: %w", err)
	}

	slog.Debug("successfully scaled down workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())

	return savedResources, nil
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

// GetNamespacesAsSet returns all namespaces as a set (map[string]struct{}).
func (c client) GetNamespacesAsSet() (map[string]struct{}, error) {
	namespaceList, err := c.clientsets.Kubernetes.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	namespaceSet := make(map[string]struct{}, len(namespaceList.Items))
	for i := range namespaceList.Items {
		ns := &namespaceList.Items[i]
		namespaceSet[ns.Name] = struct{}{}
	}

	return namespaceSet, nil
}

// GetNamespacesScopes gets the namespaces scopes from the namespaces annotations.
func (c client) GetNamespacesScopes(workloads []scalable.Workload, ctx context.Context) (map[string]*values.Scope, error) {
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
	resultChan := make(chan map[string]*values.Scope, len(namespaceSet))

	for namespace := range namespaceSet {
		waitGroup.Add(1)

		go func(namespace string, ctx context.Context) {
			defer waitGroup.Done()

			slog.Debug("fetching namespace annotations", "namespace", namespace)

			namespaceScope, err := c.GetNamespaceScope(namespace, ctx)
			if err != nil {
				errChan <- fmt.Errorf("failed to get namespace scope for namespace %s: %w", namespace, err)
				return
			}

			resultChan <- map[string]*values.Scope{namespace: namespaceScope}

			slog.Debug("correctly parsed annotations and created namespace scope", "namespace", namespace)
		}(namespace, ctx)
	}

	waitGroup.Wait()
	close(resultChan)
	close(errChan)

	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	for results := range resultChan {
		for namespace, namespaceScope := range results {
			namespaceScopes[namespace] = namespaceScope
		}
	}

	return namespaceScopes, nil
}

func (c client) GetNamespaceScope(namespace string, ctx context.Context) (*values.Scope, error) {
	nsLogger := NewResourceLoggerForNamespace(c, namespace)

	slog.Debug("fetching namespace annotations", "namespace", namespace)

	annotations, err := c.GetNamespaceAnnotations(namespace, ctx)
	if err != nil {
		err = fmt.Errorf("failed to get namespace annotations for namespace %s: %w", namespace, err)
		return nil, err
	}

	namespaceScope := values.NewScope()

	slog.Debug("parsing namespace scope from annotations", "annotations", annotations, "namespace", namespace)

	err = namespaceScope.GetScopeFromAnnotations(annotations, nsLogger, ctx)
	if err != nil {
		err = fmt.Errorf("failed to parse scope from annotations for namespace %s: %w", namespace, err)
		return nil, err
	}

	slog.Debug("correctly parsed namespace annotations", "namespace", namespace, "annotations", annotations)

	return namespaceScope, nil
}

// GetScaledObjects gets all scaledobjects in the specified namespace.
func (c client) GetScaledObjects(namespace string, ctx context.Context) ([]scalable.Workload, error) {
	scaledObjects, err := scalable.GetWorkloads("scaledobject", namespace, c.clientsets, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get scaledobjects: %w", err)
	}

	return scaledObjects, nil
}

// ensureSecret ensures that the secret used for storing TLS certificates exists.
func (c client) ensureSecret(namespace, secretName string, ctx context.Context) (bool, error) {
	isPresent := false

	_, err := c.clientsets.Kubernetes.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			isPresent = false
		} else {
			return isPresent, fmt.Errorf("unable to check secret: %w", err)
		}
	} else {
		isPresent = true
	}

	if !isPresent {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "kube-downscaler",
					"app.kubernetes.io/part-of":    "kube-downscaler",
				},
			},
		}

		_, err = c.clientsets.Kubernetes.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			return isPresent, fmt.Errorf("unable to create certificates secret: %w", err)
		}

		slog.Info(fmt.Sprintf("created the secret %s to store kube downscaler certificates", secretName), "namespace", namespace)
	}

	return isPresent, nil
}
