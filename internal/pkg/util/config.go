package util

import (
	"flag"
	"fmt"
	"regexp"
)

// CommonRuntimeConfiguration contains fields shared among different runtime configurations.
type CommonRuntimeConfiguration struct {
	// DryRun sets if the downscaler should take actions or just print them out.
	DryRun bool
	// Debug sets if debug information should be printed.
	Debug bool
	// IncludeNamespaces sets the list of namespaces to restrict the downscaler to.
	IncludeNamespaces []string
	// IncludeResources sets the list of resources to restrict the downscaler to.
	IncludeResources []string
	// ExcludeNamespaces sets the list of namespaces to ignore while downscaling.
	ExcludeNamespaces RegexList
	// ExcludeWorkloads sets the list of workload names to ignore while downscaling.
	ExcludeWorkloads RegexList
	// IncludeLabels sets the list of labels workloads have to match one of to be scaled.
	IncludeLabels RegexList
	// TimeAnnotation sets the annotation used for grace-period instead of creation time.
	TimeAnnotation string
	// MetricsEnabled sets if Prometheus metrics should be exposed.
	MetricsEnabled bool
	// JsonLogs sets logs in json format
	JsonLogs bool
	// Qps sets the maximum QPS to use while communicating with the Kubernetes API.
	Qps float64
	// Burst sets the maximum burst to use while communicating with the Kubernetes API.
	Burst int
	// Kubeconfig sets an optional kubeconfig to use for testing purposes instead of the in-cluster config.
	Kubeconfig string
}

func GetDefaultConfig() *CommonRuntimeConfiguration {
	return &CommonRuntimeConfiguration{
		DryRun:            false,
		Debug:             false,
		IncludeNamespaces: nil,
		IncludeResources:  []string{"deployments"},
		ExcludeNamespaces: RegexList{regexp.MustCompile("kube-system"), regexp.MustCompile("kube-downscaler")},
		ExcludeWorkloads:  nil,
		IncludeLabels:     nil,
		TimeAnnotation:    "",
		Kubeconfig:        "",
		MetricsEnabled:    false,
		JsonLogs:          false,
	}
}

func (c *CommonRuntimeConfiguration) ParseCommonFlags() {
	flag.BoolVar(
		&c.DryRun,
		"dry-run",
		false,
		"print actions instead of doing them. enables debug logs (default: false)",
	)
	flag.BoolVar(
		&c.Debug,
		"debug",
		false,
		"print more debug information (default: false)",
	)
	flag.Var(
		(*StringListValue)(&c.IncludeNamespaces),
		"namespace",
		"restrict the downscaler to the specified namespaces (default: all)",
	)
	flag.Var(
		(*StringListValue)(&c.IncludeResources),
		"include-resources",
		"restricts the downscaler to the specified resource types (default: deployments)",
	)
	flag.Var(
		&c.ExcludeNamespaces,
		"exclude-namespaces",
		"exclude namespaces from being scaled (default: kube-system,kube-downscaler)",
	)
	flag.Var(
		&c.ExcludeWorkloads,
		"exclude-deployments",
		"exclude deployments from being scaled (optional)",
	)
	flag.Var(
		&c.IncludeLabels,
		"matching-labels",
		"restricts the downscaler to workloads with these labels (default: all)",
	)
	flag.StringVar(
		&c.TimeAnnotation,
		"deployment-time-annotation",
		"",
		"the annotation to use instead of creation time for grace period (optional)",
	)
	flag.BoolVar(
		&c.MetricsEnabled,
		"metrics",
		false,
		"expose Prometheus metrics (default: false)",
	)
	flag.Float64Var(
		&c.Qps,
		"qps",
		500, //nolint:mnd // downscaler default for qps
		"maximum QPS to use while communicating with the Kubernetes API (default: 500)",
	)
	flag.IntVar(
		&c.Burst,
		"burst",
		1000, //nolint:mnd // downscaler default for burst
		"maximum burst to use while communicating with the Kubernetes API (default: 1000)",
	)
	flag.BoolVar(
		&c.JsonLogs,
		"json-logs",
		false,
		"sets logs in json format (default: false)",
	)
	flag.StringVar(
		&c.Kubeconfig,
		"k",
		"",
		"kubeconfig to use instead of the in-cluster config (optional)",
	)
}

func (c *CommonRuntimeConfiguration) ParseConfigEnvVars() error {
	if err := GetEnvValue("EXCLUDE_NAMESPACES", &c.ExcludeNamespaces); err != nil {
		return fmt.Errorf("error while getting EXCLUDE_NAMESPACES environment variable: %w", err)
	}

	if err := GetEnvValue("EXCLUDE_DEPLOYMENTS", &c.ExcludeWorkloads); err != nil {
		return fmt.Errorf("error while getting EXCLUDE_DEPLOYMENTS environment variable: %w", err)
	}

	return nil
}
