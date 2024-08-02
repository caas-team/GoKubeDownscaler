package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/caas-team/gokubedownscaler/api/kubernetes"
	"github.com/caas-team/gokubedownscaler/values"
)

var (
	layerCli = values.NewLayer()
	layerEnv = values.NewLayer()

	dryRun            bool              = false                                               // if the downscaler should take actions or just print them out
	debug             bool              = false                                               // if debug information should be printed
	once              bool              = false                                               // if the scan should only run once
	interval          values.Duration   = values.Duration(30 * time.Second)                   // how long to wait between scans
	namespaces        values.StringList = values.StringList{""}                               // list of namespaces to restrict the downscaler to
	resources         values.StringList = values.StringList{"deployments"}                    // list of resources to restrict the downscaler to
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
	flag.Var(&namespaces, "namespace", "restrict the downscaler to the specified namespaces (default: all)")
	flag.Var(&resources, "include-resources", "restricts the downscaler to the specified resource types (default: deployments, incompatible: exclude-resources)")
	flag.Var(&excludeNamespaces, "exclude-namespaces", "exclude namespaces from being scaled (default: kube-system,kube-downscaler)")
	flag.Var(&excludeWorkloads, "exclude-deployments", "exclude deployments from being scaled (optional)")
	flag.StringVar(&kubeconfig, "k", "", "kubeconfig to use instead of the in-cluster config (optional)")

	// env runtime configuration
	err := values.GetEnvValue("EXCLUDE_NAMESPACES", &excludeNamespaces)
	if err != nil {
		slog.Error("error while getting EXCLUDE_NAMESPACES environment variable", "error", err)
	}
	err = values.GetEnvValue("EXCLUDE_DEPLOYMENTS", &excludeWorkloads)
	if err != nil {
		slog.Error("error while getting EXCLUDE_DEPLOYMENTS environment variable", "error", err)
	}

	layerEnv, err = values.GetLayerFromEnv()
	if err != nil {
		slog.Error("failed to get layer from env", "error", err)
	}
}

func main() {
	flag.Parse()
	if debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	ctx := context.Background()

	client, err := kubernetes.NewClient(kubeconfig)
	if err != nil {
		slog.Error("failed to get new client", "error", err)
		os.Exit(1)
	}

	for {
		workloads, err := client.GetWorkloads(namespaces, resources, ctx)
		if err != nil {
			slog.Error("failed to get workloads", "error", err)
			os.Exit(1)
		}

		for _, workload := range workloads {
			namespaceAnnotations, err := client.GetNamespaceAnnotations(workload.GetNamespace(), ctx)
			if err != nil {
				slog.Error("failed to get namespace annotations", "error", err)
				os.Exit(1)
			}
			layerWorkload, err := values.GetLayerFromAnnotations(workload.GetAnnotations())
			if err != nil {
				slog.Error("failed to parse layer from annotations", "error", err)
				os.Exit(1)
			}
			layerNamespace, err := values.GetLayerFromAnnotations(namespaceAnnotations)
			if err != nil {
				slog.Error("failed to parse layer from annotations", "error", err)
				os.Exit(1)
			}

			layers := values.Layers{layerWorkload, layerNamespace, layerCli, layerEnv}

			scaling, err := layers.GetCurrentScaling()
			if err != nil {
				slog.Error("failed to get current scaling for workload", "error", err)
				os.Exit(1)
			}
			if scaling == values.ScalingIncompatible {
				slog.Error("scaling is incompatible, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
				continue
			}
			if scaling == values.ScalingIgnore {
				slog.Debug("scaling is ignored, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
				continue
			}
			if scaling == values.ScalingDown {
				slog.Debug("downscaling workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
				downscaleReplicas, err := layers.GetDownscaleReplicas()
				if err != nil {
					slog.Error("failed to parse layer from annotations", "error", err)
					os.Exit(1)
				}
				err = client.DownscaleWorkload(downscaleReplicas, workload, ctx)
				if err != nil {
					slog.Error("failed to parse layer from annotations", "error", err)
					os.Exit(1)
				}
			}
			if scaling == values.ScalingUp {
				slog.Debug("upscaling workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
				err := client.UpscaleWorkload(workload, ctx)
				if err != nil {
					slog.Error("failed to parse layer from annotations", "error", err)
					os.Exit(1)
				}
			}

		}

		if once {
			break
		}
		time.Sleep(time.Duration(interval))
	}
}
