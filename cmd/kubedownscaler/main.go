package main

import (
	"flag"
	"log/slog"
	"time"

	"github.com/caas-team/gokubedownscaler"
	"github.com/caas-team/gokubedownscaler/values"
)

var (
	layerCli values.Layer
	layerEnv values.Layer

	maxGracePeriod    values.Duration                                                         // the maximum grace period allowed to be set in any layer
	dryRun            bool              = false                                               // if the downscaler should take actions or just print them out
	debug             bool              = false                                               // if debug information should be printed
	once              bool              = false                                               // if the scan should only run once
	interval          values.Duration   = values.Duration(30 * time.Second)                   // how long to wait between scans
	namespaces        values.StringList                                                       // list of namespaces to restrict the downscaler to
	resources         values.StringList                                                       // list of resources to restrict the downscaler to
	excludeNamespaces values.StringList = values.StringList{"kube-system", "kube-downscaler"} // list of namespaces to ignore while downscaling
	excludeWorkloads  values.StringList                                                       // list of workload names to ignore while downscaling
	kubeconfig        string                                                                  // optional kubeconfig to use for testing purposes instead of the in-cluster config
)

func init() {
	// cli layer values
	flag.Var(&layerCli.DownscalePeriod, "downscale-period", "")
	flag.Var(&layerCli.DownTime, "default-downtime", "")
	flag.Var(&layerCli.UpscalePeriod, "upscale-period", "")
	flag.Var(&layerCli.UpTime, "default-uptime", "")
	flag.BoolVar(&layerCli.Exclude, "explicit-include", false, "")
	flag.IntVar(&layerCli.DownscaleReplicas, "downtime-replicas", 0, "")
	flag.Var(&layerCli.GracePeriod, "grace-period", "")
	flag.StringVar(&layerCli.TimeAnnotation, "deployment-time-annotation", "", "")

	// cli runtime configuration
	flag.BoolVar(&dryRun, "dry-run", false, "print actions instead of doing them (default: false)")
	flag.BoolVar(&debug, "debug", false, "print more debug information (default: false)")
	flag.BoolVar(&once, "once", false, "run scan only once (default: false)")
	flag.Var(&interval, "interval", "time between scans (default: 30s)")
	flag.Var(&namespaces, "namespace", "restrict the downscaler to the specified namespaces (default: kube-system,kube-downscaler)")
	flag.Var(&resources, "include-resources", "restricts the downscaler to the specified resource types (optional, incompatible: exclude-resources)")
	flag.Var(&excludeNamespaces, "exclude-namespaces", "exclude namespaces from being scaled (optional)")
	flag.Var(&excludeWorkloads, "exclude-deployments", "exclude deployments from being scaled (optional)")
	flag.Var(&maxGracePeriod, "max-grace-period", "a maximum grace period allowed (optional)")
	flag.StringVar(&kubeconfig, "k", "", "kubeconfig to use instead of the in-cluster config (optional)")

	// env layer values
	err := gokubedownscaler.GetEnvValue("UPSCALE_PERIOD", &layerEnv.UpscalePeriod)
	if err != nil {
		slog.Error("error while getting UPSCALE_PERIOD environment variable", "error", err)
	}
	err = gokubedownscaler.GetEnvValue("DEFAULT_UPTIME", &layerEnv.UpTime)
	if err != nil {
		slog.Error("error while getting DEFAULT_UPTIME environment variable", "error", err)
	}
	err = gokubedownscaler.GetEnvValue("DOWNSCALE_PERIOD", &layerEnv.DownscalePeriod)
	if err != nil {
		slog.Error("error while getting DOWNSCALE_PERIOD environment variable", "error", err)
	}
	err = gokubedownscaler.GetEnvValue("DEFAULT_DOWNTIME", &layerEnv.DownTime)
	if err != nil {
		slog.Error("error while getting DEFAULT_DOWNTIME environment variable", "error", err)
	}

	// env runtime configuration
	err = gokubedownscaler.GetEnvValue("EXCLUDE_NAMESPACES", &excludeNamespaces)
	if err != nil {
		slog.Error("error while getting EXCLUDE_NAMESPACES environment variable", "error", err)
	}
	err = gokubedownscaler.GetEnvValue("EXCLUDE_DEPLOYMENTS", &excludeWorkloads)
	if err != nil {
		slog.Error("error while getting EXCLUDE_DEPLOYMENTS environment variable", "error", err)
	}
}

func main() {
	flag.Parse()
}
