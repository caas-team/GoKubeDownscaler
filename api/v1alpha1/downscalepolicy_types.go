package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// DownscalePolicyKindName is the kind of the DownscalePolicy resource.
const DownscalePolicyKindName = "DownscalePolicy"

// MaxListedWorkloads is the maximum amount of workloads listed in the status of a DownscalePolicy.
const MaxListedWorkloads = 50

// Condition types and reasons used in the DownscalePolicy status.
const (
	// ConditionTypeValid indicates whether the scaling values of the policy could be parsed.
	ConditionTypeValid = "Valid"
	// ConditionTypeAccepted indicates whether the policy is applied to all workloads it matches.
	ConditionTypeAccepted = "Accepted"

	// ReasonParsed is used when all scaling values of the policy were parsed successfully.
	ReasonParsed = "Parsed"
	// ReasonInvalidValue is used when at least one scaling value of the policy is invalid.
	ReasonInvalidValue = "InvalidValue"
	// ReasonAccepted is used when the policy is applied to all workloads it matches.
	ReasonAccepted = "Accepted"
	// ReasonConflicted is used when another policy takes precedence on some of the matched workloads.
	ReasonConflicted = "Conflicted"
	// ReasonInvalid is used when the policy is not applied because it is not valid.
	ReasonInvalid = "Invalid"
)

// DownscalePolicyKind identifies a kind of workload by its API group and kind.
type DownscalePolicyKind struct {
	// Group is the API group of the workload kind. Use an empty string for the core API group.
	// +optional
	Group gatewayv1.Group `json:"group"`

	// Kind is the kind of the workload, e.g. Deployment.
	// +required
	Kind gatewayv1.Kind `json:"kind"`
}

// DownscalePolicySelector selects current and future workloads by kind and labels.
// At least one of kinds, matchLabels or matchExpressions has to be set.
// To match all workloads in the namespace, omit both the selector and targetRefs.
//
// +kubebuilder:validation:XValidation:rule="(has(self.kinds) && size(self.kinds) > 0) || (has(self.matchLabels) && size(self.matchLabels) > 0) || (has(self.matchExpressions) && size(self.matchExpressions) > 0)",message="selector must set at least one of kinds, matchLabels or matchExpressions; omit the selector and targetRefs to match all workloads in the namespace"
//
//nolint:lll // kubebuilder markers can't be split
type DownscalePolicySelector struct {
	// Kinds restricts the selector to the given workload kinds.
	// When omitted, all kinds handled by the downscaler are selected.
	// +optional
	// +listType=atomic
	Kinds []DownscalePolicyKind `json:"kinds,omitempty"`

	// LabelSelector selects workloads by their labels using matchLabels and/or matchExpressions.
	// When empty, all workloads of the selected kinds are matched.
	metav1.LabelSelector `json:",inline"`
}

// DownscalePolicyValues holds the scaling values of a policy.
// The names and syntax are the same as the corresponding downscaler annotations.
type DownscalePolicyValues struct {
	// DownscalePeriod is the period to scale down in. Incompatible with uptime and downtime.
	// +optional
	DownscalePeriod string `json:"downscalePeriod,omitempty"`

	// Downtime is the timespan in which workloads are scaled down, outside of it they are scaled up.
	// Incompatible with uptime, downscalePeriod and upscalePeriod.
	// +optional
	Downtime string `json:"downtime,omitempty"`

	// UpscalePeriod is the period to scale up in. Incompatible with uptime and downtime.
	// +optional
	UpscalePeriod string `json:"upscalePeriod,omitempty"`

	// Uptime is the timespan in which workloads are scaled up, outside of it they are scaled down.
	// Incompatible with downtime, downscalePeriod and upscalePeriod.
	// +optional
	Uptime string `json:"uptime,omitempty"`

	// Exclude excludes the workloads from scaling, either as a boolean or as timespans.
	// +optional
	Exclude string `json:"exclude,omitempty"`

	// ExcludeUntil excludes the workloads from scaling until the given RFC3339 timestamp.
	// +optional
	ExcludeUntil string `json:"excludeUntil,omitempty"`

	// ForceUptime forces the workloads into an uptime state, either as a boolean or as timespans.
	// +optional
	ForceUptime string `json:"forceUptime,omitempty"`

	// ForceDowntime forces the workloads into a downtime state, either as a boolean or as timespans.
	// +optional
	ForceDowntime string `json:"forceDowntime,omitempty"`

	// DownscaleReplicas is the amount (or percentage) of replicas to scale down to.
	// +optional
	DownscaleReplicas string `json:"downscaleReplicas,omitempty"`

	// GracePeriod is the duration after the creation of a workload until it is scaled down for the first time.
	// +optional
	GracePeriod string `json:"gracePeriod,omitempty"`

	// ScaleChildren makes the scaling of a workload immediately scale its children, when applicable.
	// +optional
	ScaleChildren string `json:"scaleChildren,omitempty"`

	// UpscaleExcluded makes excluded workloads be scaled up.
	// +optional
	UpscaleExcluded string `json:"upscaleExcluded,omitempty"`
}

// DownscalePolicySpec defines which workloads a policy applies to and which scaling values it sets.
// A workload is matched if it is listed in targetRefs or matches the selector.
// If both targetRefs and selector are omitted, the policy applies to all workloads in its namespace.
type DownscalePolicySpec struct {
	// TargetRefs explicitly lists workloads in the namespace of the policy.
	// +optional
	// +listType=atomic
	// +kubebuilder:validation:MaxItems=64
	TargetRefs []gatewayv1.LocalPolicyTargetReference `json:"targetRefs,omitempty"`

	// Selector matches current and future workloads in the namespace of the policy by kind and labels.
	// +optional
	Selector *DownscalePolicySelector `json:"selector,omitempty"`

	DownscalePolicyValues `json:",inline"`
}

// DownscalePolicyWorkload identifies a workload in the namespace of the policy.
type DownscalePolicyWorkload struct {
	// Group is the API group of the workload. It is empty for the core API group.
	// +optional
	Group gatewayv1.Group `json:"group"`

	// Kind is the kind of the workload.
	// +required
	Kind gatewayv1.Kind `json:"kind"`

	// Name is the name of the workload.
	// +required
	Name gatewayv1.ObjectName `json:"name"`
}

// DownscalePolicyStatus reports how the policy was evaluated by the downscaler.
type DownscalePolicyStatus struct {
	// Conditions describe the current state of the policy.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// MatchedWorkloads is the amount of workloads the policy is applied to.
	// +optional
	MatchedWorkloads int32 `json:"matchedWorkloads"`

	// Workloads lists the workloads the policy is applied to, sorted by group, kind and name.
	// At most 50 workloads are listed; matchedWorkloads always holds the total amount.
	// +optional
	// +listType=atomic
	// +kubebuilder:validation:MaxItems=50
	Workloads []DownscalePolicyWorkload `json:"workloads,omitempty"`

	// ObservedGeneration is the generation of the policy that was last evaluated.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// DownscalePolicy defines uptime/downtime schedules for workloads in its namespace.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=dsp,categories=kube-downscaler
// +kubebuilder:printcolumn:name="Valid",type=string,JSONPath=`.status.conditions[?(@.type=="Valid")].status`
// +kubebuilder:printcolumn:name="Accepted",type=string,JSONPath=`.status.conditions[?(@.type=="Accepted")].status`
// +kubebuilder:printcolumn:name="Matched",type=integer,JSONPath=`.status.matchedWorkloads`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type DownscalePolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DownscalePolicySpec   `json:"spec,omitempty"`
	Status DownscalePolicyStatus `json:"status,omitempty"`
}

// DownscalePolicyList contains a list of DownscalePolicy.
//
// +kubebuilder:object:root=true
type DownscalePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []DownscalePolicy `json:"items"`
}
