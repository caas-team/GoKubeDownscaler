package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/caas-team/gokubedownscaler/internal/api/kubernetes"
	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
)

var (
	layerCli = values.NewLayer()
	layerEnv = values.NewLayer()

	dryRun            = false                                               // if the downscaler should take actions or just print them out // NOT_IMPLEMENTED
	debug             = false                                               // if debug information should be printed
	once              = false                                               // if the scan should only run once
	interval          = values.Duration(30 * time.Second)                   // how long to wait between scans
	namespaces        = values.StringList{""}                               // list of namespaces to restrict the downscaler to
	resources         = values.StringList{"deployments"}                    // list of resources to restrict the downscaler to
	excludeNamespaces = values.StringList{"kube-system", "kube-downscaler"} // list of namespaces to ignore while downscaling // NOT_IMPLEMENTED
	excludeWorkloads  values.StringList                                     // list of workload names to ignore while downscaling // NOT_IMPLEMENTED
	kubeconfig        string                                                // optional kubeconfig to use for testing purposes instead of the in-cluster config
)

func init() {
	// cli layer values
	flag.Var(&layerCli.DownscalePeriod, "downscale-period", "period to scale down in (default: never, incompatible: UpscaleTime, DownscaleTime)")
	flag.Var(&layerCli.DownTime, "default-downtime", "timespans where workloads will be scaled down, outside of them they will be scaled up (default: never, incompatible: UpscalePeriod, DownscalePeriod)")
	flag.Var(&layerCli.UpscalePeriod, "upscale-period", "periods to scale up in (default: never, incompatible: UpscaleTime, DownscaleTime)")
	flag.Var(&layerCli.UpTime, "default-uptime", "timespans where workloads will be scaled up, outside of them they will be scaled down (default: never, incompatible: UpscalePeriod, DownscalePeriod)")
	flag.Var(&layerCli.Exclude, "explicit-include", "sets exclude on cli layer to true, makes it so namespaces or deployments have to specify downscaler/exclude=false (default: false)")
	flag.IntVar(&layerCli.DownscaleReplicas, "downtime-replicas", 0, "the replicas to scale down to (default: 0)")
	flag.Var(&layerCli.GracePeriod, "grace-period", "the grace period between creation of workload until first downscale (default: 15min)")                       // NOT_IMPLEMENTED: default not implemented
	flag.StringVar(&layerCli.TimeAnnotation, "deployment-time-annotation", "", "the annotation to use instead of creation time for grace period (default: none)") // NOT_IMPLEMENTED: not implemented to ignore ""

	// cli runtime configuration
	flag.BoolVar(&dryRun, "dry-run", false, "print actions instead of doing them (default: false)")
	flag.BoolVar(&debug, "debug", false, "print more debug information (default: false)")
	flag.BoolVar(&once, "once", false, "run scan only once (default: false)")
	flag.Var(&interval, "interval", "time between scans (default: 30s)")
	flag.Var(&namespaces, "namespace", "restrict the downscaler to the specified namespaces (default: all, incompatible: exclude-namespaces)")
	flag.Var(&resources, "include-resources", "restricts the downscaler to the specified resource types (default: deployments)")
	flag.Var(&excludeNamespaces, "exclude-namespaces", "exclude namespaces from being scaled (default: kube-system,kube-downscaler, incompatible: namespaces)")
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
		slog.Error("failed to create new kubernetes client", "error", err)
		os.Exit(1)
	}

	for {
		slog.Debug("scanning workloads")

		workloads, err := client.GetWorkloads(namespaces, resources, ctx)
		if err != nil {
			slog.Error("failed to get workloads", "error", err)
			os.Exit(1)
		}

		var wg sync.WaitGroup
		for _, workload := range workloads {
			wg.Add(1)
			go func() {
				slog.Debug("scanning workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
				defer wg.Done()

				ok := scanWorkload(workload, client, ctx, layerCli, layerEnv)
				if !ok {
					slog.Error("failed to scan workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
					return
				}

				slog.Debug("successfully scanned workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
			}()
		}
		wg.Wait()

		if once {
			slog.Debug("once is set to true, exiting")
			break
		}
		slog.Debug("waiting until next scan", "interval", interval.String())
		time.Sleep(time.Duration(interval))
	}
}

// scanWorkload runs a scan on the worklod, determining the scaling and scaling the workload
func scanWorkload(workload scalable.Workload, client kubernetes.Client, ctx context.Context, layerCli, layerEnv values.Layer) bool {
	resourceLogger := kubernetes.NewResourceLogger(client, workload)

	namespaceAnnotations, err := client.GetNamespaceAnnotations(workload.GetNamespace(), ctx)
	if err != nil {
		slog.Error("failed to get namespace annotations", "error", err, "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return false
	}
	layerWorkload, err := values.GetLayerFromAnnotations(workload.GetAnnotations(), resourceLogger, ctx)
	if err != nil {
		slog.Error("failed to parse workload layer from annotations", "error", err, "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return false
	}
	layerNamespace, err := values.GetLayerFromAnnotations(namespaceAnnotations, resourceLogger, ctx)
	if err != nil {
		slog.Error("failed to parse namespace layer from annotations", "error", err, "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return false
	}

	layers := values.Layers{layerWorkload, layerNamespace, layerCli, layerEnv}

	if layers.GetExcluded() {
		slog.Debug("workload is excluded, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return true
	}

	scaling, err := layers.GetCurrentScaling()
	if err != nil {
		slog.Error("failed to get current scaling for workload", "error", err, "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return false
	}
	if scaling == values.ScalingIncompatible {
		slog.Error("scaling is incompatible, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return false
	}
	if scaling == values.ScalingIgnore {
		slog.Debug("scaling is ignored, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return true
	}
	if scaling == values.ScalingDown {
		slog.Debug("downscaling workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		downscaleReplicas, err := layers.GetDownscaleReplicas()
		if err != nil {
			slog.Error("failed to get downscale replicas", "error", err, "workload", workload.GetName(), "namespace", workload.GetNamespace())
			return false
		}
		err = client.DownscaleWorkload(downscaleReplicas, workload, ctx)
		if err != nil {
			slog.Error("failed to downscale workload", "error", err, "workload", workload.GetName(), "namespace", workload.GetNamespace())
			return false
		}
	}
	if scaling == values.ScalingUp {
		slog.Debug("upscaling workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		err := client.UpscaleWorkload(workload, ctx)
		if err != nil {
			slog.Error("failed to upscale workload", "error", err, "workload", workload.GetName(), "namespace", workload.GetNamespace())
			return false
		}
	}
	return true
}
