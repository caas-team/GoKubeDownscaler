// Package v1alpha1 contains the v1alpha1 API of the kube-downscaler.io group.
//
// +kubebuilder:object:generate=true
// +groupName=kube-downscaler.io
package v1alpha1

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.23.0 object paths=./...
//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.23.0 crd paths=./... output:crd:artifacts:config=../../deployments/chart/crds

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

//nolint:gochecknoglobals // the scheme builder pattern requires package level variables
var (
	// GroupVersion is the group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: "kube-downscaler.io", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// addKnownTypes adds the types of this group-version to the given scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion, &DownscalePolicy{}, &DownscalePolicyList{})
	metav1.AddToGroupVersion(scheme, GroupVersion)

	return nil
}
