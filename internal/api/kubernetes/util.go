package kubernetes

import (
	"fmt"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

// getConfig gets a rest.Config for the specified kubeconfig or if empty from the in-cluster config
func getConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig == "" {
		return rest.InClusterConfig()
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

// GetCurrentNamespaceFromFile retrieves downscaler namespace from its service account file
func GetCurrentNamespaceFromFile() (string, error) {
	namespaceFile := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	namespace, err := os.ReadFile(namespaceFile)
	if err != nil {
		return "", fmt.Errorf("failed to read namespace file: %v", err)
	}
	return string(namespace), nil
}
