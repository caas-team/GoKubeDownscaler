package metrics

// metricName returns the name of the metric based on the base name and whether it's a dry run.
func metricName(base string, dryRun bool) string {
	if dryRun {
		return "kubedownscaler_potential_" + base
	}

	return "kubedownscaler_" + base
}

// helperDescription returns the description of the metric based on the base description and whether it's a dry run.
func helperDescription(base string, dryRun bool) string {
	if dryRun {
		return "Number of potential " + base
	}

	return "Number of " + base
}
