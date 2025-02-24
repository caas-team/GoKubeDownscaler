package util

import (
	"flag"
	"fmt"
	"regexp"
	"time"
)

// KubeDownscalerRuntimeConfiguration represents the runtime configuration for the downscaler.
type KubeDownscalerRuntimeConfiguration struct {
	// DryRun sets if the downscaler should take actions or just print them out.
	DryRun bool
	// Debug sets if debug information should be printed.
	Debug bool
	// Once sets if the scan should only run once.
	Once bool
	// LeaderElection sets if leader election should be performed.
	LeaderElection bool
	// Interval sets how long to wait between scans.
	Interval time.Duration
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
	// MaxRetriesOnError sets the maximum number of retries when encountering Kubernetes HTTP 409 conflict error.
	MaxRetriesOnConflict int
	// Kubeconfig sets an optional kubeconfig to use for testing purposes instead of the in-cluster config.
	Kubeconfig string
}

func GetDefaultConfig() *RuntimeConfiguration {
	return &RuntimeConfiguration{
		DryRun:            false,
		Debug:             false,
		Once:              false,
		Interval:          30 * time.Second,
		IncludeNamespaces: nil,
		IncludeResources:  []string{"deployments"},
		ExcludeNamespaces: RegexList{regexp.MustCompile("kube-system"), regexp.MustCompile("kube-downscaler")},
		ExcludeWorkloads:  nil,
		IncludeLabels:     nil,
		TimeAnnotation:    "",
		Kubeconfig:        "",
	}
}

// ParseConfigFlags sets all cli flags required for the runtime configuration.
func (c *KubeDownscalerRuntimeConfiguration) ParseConfigFlags() {
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
	flag.BoolVar(
		&c.Once,
		"once",
		false,
		"run scan only once (default: false)",
	)
	flag.BoolVar(
		&c.LeaderElection,
		"leader-election",
		false,
		"enables leader election (default: false)",
	)
	flag.Var(
		(*DurationValue)(&c.Interval),
		"interval",
		"time between scans (default: 30s)",
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
		&c.Kubeconfig,
		"k",
		"",
		"kubeconfig to use instead of the in-cluster config (optional)",
	)
	flag.StringVar(
		&c.TimeAnnotation,
		"deployment-time-annotation",
		"",
		"the annotation to use instead of creation time for grace period (optional)",
	)
	flag.IntVar(
		&c.MaxRetriesOnConflict,
		"max-retries-on-conflict",
		0,
		"the annotation to use instead of creation time for grace period (optional)",
	)
}

// ParseConfigEnvVars parses all environment variables for the runtime configuration.
func (c *KubeDownscalerRuntimeConfiguration) ParseConfigEnvVars() error {
	err := GetEnvValue("EXCLUDE_NAMESPACES", &c.ExcludeNamespaces)
	if err != nil {
		return fmt.Errorf("error while getting EXCLUDE_NAMESPACES environment variable: %w", err)
	}

	err = GetEnvValue("EXCLUDE_DEPLOYMENTS", &c.ExcludeWorkloads)
	if err != nil {
		return fmt.Errorf("error while getting EXCLUDE_DEPLOYMENTS environment variable: %w", err)
	}

	return nil
}

// AdmissionControllerRuntimeConfiguration represents the runtime configuration for the admission controller.
type AdmissionControllerRuntimeConfiguration struct {
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
	// Kubeconfig sets an optional kubeconfig to use for testing purposes instead of the in-cluster config.
	Kubeconfig string
}

// ParseConfigFlags sets all cli flags required for the runtime configuration.
func (c *AdmissionControllerRuntimeConfiguration) ParseConfigFlags() {
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
		&c.Kubeconfig,
		"k",
		"",
		"kubeconfig to use instead of the in-cluster config (optional)",
	)
}

// ParseConfigEnvVars parses all environment variables for the runtime configuration.
func (c *AdmissionControllerRuntimeConfiguration) ParseConfigEnvVars() error {
	err := GetEnvValue("EXCLUDE_NAMESPACES", &c.ExcludeNamespaces)
	if err != nil {
		return fmt.Errorf("error while getting EXCLUDE_NAMESPACES environment variable: %w", err)
	}

	err = GetEnvValue("EXCLUDE_DEPLOYMENTS", &c.ExcludeWorkloads)
	if err != nil {
		return fmt.Errorf("error while getting EXCLUDE_DEPLOYMENTS environment variable: %w", err)
	}

	return nil
}
