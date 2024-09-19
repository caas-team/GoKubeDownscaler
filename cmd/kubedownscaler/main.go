package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/caas-team/gokubedownscaler/internal/api/kubernetes"
	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
)

var (
	layerCli = values.NewLayer()
	layerEnv = values.NewLayer()

	// if the downscaler should take actions or just print them out
	dryRun = false
	// if debug information should be printed
	debug = false
	// if the scan should only run once
	once = false
	// how long to wait between scans
	interval = values.Duration(30 * time.Second)
	// list of namespaces to restrict the downscaler to
	includeNamespaces values.StringList
	// list of resources to restrict the downscaler to
	includeResources = values.StringList{"deployments"}
	// list of namespaces to ignore while downscaling
	excludeNamespaces = values.RegexList{regexp.MustCompile("kube-system"), regexp.MustCompile("kube-downscaler")}
	// list of workload names to ignore while downscaling
	excludeWorkloads values.RegexList
	// workloads have to match one of these labels to be scaled
	includeLabels values.RegexList
	// annotation to use for grace-period instead of creation time
	timeAnnotation string
	// optional kubeconfig to use for testing purposes instead of the in-cluster config
	kubeconfig string
)

func init() {
	// set defaults for layers
	layerCli.GracePeriod = values.Duration(15 * time.Minute)
	layerCli.DownscaleReplicas = 0

	// cli layer values
	flag.Var(&layerCli.DownscalePeriod, "downscale-period", "period to scale down in (default: never, incompatible: UpscaleTime, DownscaleTime)")
	flag.Var(&layerCli.DownTime, "default-downtime", "timespans where workloads will be scaled down, outside of them they will be scaled up (default: never, incompatible: UpscalePeriod, DownscalePeriod)")
	flag.Var(&layerCli.UpscalePeriod, "upscale-period", "periods to scale up in (default: never, incompatible: UpscaleTime, DownscaleTime)")
	flag.Var(&layerCli.UpTime, "default-uptime", "timespans where workloads will be scaled up, outside of them they will be scaled down (default: never, incompatible: UpscalePeriod, DownscalePeriod)")
	flag.Var(&layerCli.Exclude, "explicit-include", "sets exclude on cli layer to true, makes it so namespaces or deployments have to specify downscaler/exclude=false (default: false)")
	flag.Var((*values.Int32Value)(&layerCli.DownscaleReplicas), "downtime-replicas", "the replicas to scale down to (default: 0)")
	flag.Var(&layerCli.GracePeriod, "grace-period", "the grace period between creation of workload until first downscale (default: 15min)")

	// cli runtime configuration
	flag.BoolVar(&dryRun, "dry-run", false, "print actions instead of doing them. enables debug logs (default: false)")
	flag.BoolVar(&debug, "debug", false, "print more debug information (default: false)")
	flag.BoolVar(&once, "once", false, "run scan only once (default: false)")
	flag.Var(&interval, "interval", "time between scans (default: 30s)")
	flag.Var(&includeNamespaces, "namespace", "restrict the downscaler to the specified namespaces (default: all)")
	flag.Var(&includeResources, "include-resources", "restricts the downscaler to the specified resource types (default: deployments)")
	flag.Var(&excludeNamespaces, "exclude-namespaces", "exclude namespaces from being scaled (default: kube-system,kube-downscaler)")
	flag.Var(&excludeWorkloads, "exclude-deployments", "exclude deployments from being scaled (optional)")
	flag.Var(&includeLabels, "matching-labels", "restricts the downscaler to workloads with these labels (default: all)")
	flag.StringVar(&kubeconfig, "k", "", "kubeconfig to use instead of the in-cluster config (optional)")
	flag.StringVar(&timeAnnotation, "deployment-time-annotation", "", "the annotation to use instead of creation time for grace period (optional)")

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
	if debug || dryRun {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	if err := layerCli.CheckForIncompatibleFields(); err != nil {
		slog.Error("found incompatible fields", "error", err)
		os.Exit(1)
	}
	ctx := context.Background()

	client, err := kubernetes.NewClient(kubeconfig, dryRun)
	if err != nil {
		slog.Error("failed to create new Kubernetes client", "error", err)
		os.Exit(1)
	}

	for {
		slog.Debug("scanning workloads")

		workloads, err := client.GetWorkloads(includeNamespaces, includeResources, ctx)
		if err != nil {
			slog.Error("failed to get workloads", "error", err)
			os.Exit(1)
		}
		workloads = scalable.FilterExcluded(workloads, includeLabels, excludeNamespaces, excludeWorkloads)

		var wg sync.WaitGroup
		for _, workload := range workloads {
			wg.Add(1)
			go func() {
				slog.Debug("scanning workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
				defer wg.Done()

				err := scanWorkload(workload, client, ctx, layerCli, layerEnv)
				if err != nil {
					slog.Error("failed to scan workload", "error", err, "workload", workload.GetName(), "namespace", workload.GetNamespace())
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
func scanWorkload(workload scalable.Workload, client kubernetes.Client, ctx context.Context, layerCli, layerEnv values.Layer) error {
	resourceLogger := kubernetes.NewResourceLogger(client, workload)

	namespaceAnnotations, err := client.GetNamespaceAnnotations(workload.GetNamespace(), ctx)
	if err != nil {
		return fmt.Errorf("failed to get namespace annotations: %w", err)
	}
	layerWorkload, err := values.GetLayerFromAnnotations(workload.GetAnnotations(), resourceLogger, ctx)
	if err != nil {
		return fmt.Errorf("failed to parse workload layer from annotations: %w", err)
	}
	layerNamespace, err := values.GetLayerFromAnnotations(namespaceAnnotations, resourceLogger, ctx)
	if err != nil {
		return fmt.Errorf("failed to parse namespace layer from annotations: %w", err)
	}

	layers := values.Layers{layerWorkload, layerNamespace, layerCli, layerEnv}

	ok, err := layers.IsInGracePeriod(timeAnnotation, workload.GetAnnotations(), workload.GetCreationTimestamp().Time, resourceLogger, ctx)
	if err != nil {
		return fmt.Errorf("failed to get if workload is on grace period: %w", err)
	}
	if ok {
		slog.Debug("workload is on grace period, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return nil
	}
	if layers.GetExcluded() {
		slog.Debug("workload is excluded, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return nil
	}

	scaling := layers.GetCurrentScaling()
	if scaling == values.ScalingNone {
		slog.Debug("scaling is not set by any layer, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return nil
	}
	if scaling == values.ScalingIgnore {
		slog.Debug("scaling is ignored, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return nil
	}
	if scaling == values.ScalingDown {
		slog.Debug("downscaling workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		downscaleReplicas, err := layers.GetDownscaleReplicas()
		if err != nil {
			return fmt.Errorf("failed to get downscale replicas: %w", err)
		}
		err = client.DownscaleWorkload(downscaleReplicas, workload, ctx)
		if err != nil {
			return fmt.Errorf("failed to downscale workload: %w", err)
		}
	}
	if scaling == values.ScalingUp {
		slog.Debug("upscaling workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		err := client.UpscaleWorkload(workload, ctx)
		if err != nil {
			return fmt.Errorf("failed to upscale workload: %w", err)
		}
	}
	return nil
}
