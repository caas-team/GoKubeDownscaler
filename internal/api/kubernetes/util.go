package kubernetes

import (
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
