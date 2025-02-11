package kubernetes

import (
	"fmt"
	"os"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// getConfig gets a rest.Config for the specified kubeconfig or if empty from the in-cluster config.
func getConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig == "" {
		return rest.InClusterConfig() //nolint: wrapcheck // error gets wrapped in the calling function, so its fine
	}

	return clientcmd.BuildConfigFromFlags("", kubeconfig) //nolint: wrapcheck // error gets wrapped in the calling function, so its fine
}

// GetCurrentNamespace retrieves downscaler namespace from its service account file.
func getCurrentNamespace() (string, error) {
	const namespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

	namespace, err := os.ReadFile(namespaceFile)
	if err != nil {
		return "", fmt.Errorf("failed to read namespace file: %w", err)
	}

	return string(namespace), nil
}
