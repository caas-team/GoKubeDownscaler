package values

import "github.com/caas-team/gokubedownscaler/api/v1alpha1"

// PolicyValuesToAnnotations maps the values of a DownscalePolicy to the corresponding downscaler annotations,
// so they can be parsed with GetScopeFromAnnotations. Empty values are omitted.
func PolicyValuesToAnnotations(policyValues *v1alpha1.DownscalePolicyValues) map[string]string {
	mapping := []struct {
		annotation string
		value      string
	}{
		{annotationDownscalePeriod, policyValues.DownscalePeriod},
		{annotationDowntime, policyValues.Downtime},
		{annotationUpscalePeriod, policyValues.UpscalePeriod},
		{annotationUptime, policyValues.Uptime},
		{annotationExclude, policyValues.Exclude},
		{annotationExcludeUntil, policyValues.ExcludeUntil},
		{annotationForceUptime, policyValues.ForceUptime},
		{annotationForceDowntime, policyValues.ForceDowntime},
		{annotationDownscaleReplicas, policyValues.DownscaleReplicas},
		{annotationGracePeriod, policyValues.GracePeriod},
		{annotationScaleChildren, policyValues.ScaleChildren},
		{annotationExclusionUpscale, policyValues.UpscaleExcluded},
	}

	annotations := make(map[string]string, len(mapping))

	for _, entry := range mapping {
		if entry.value == "" {
			continue
		}

		annotations[entry.annotation] = entry.value
	}

	return annotations
}
