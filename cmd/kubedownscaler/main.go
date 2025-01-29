package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/caas-team/gokubedownscaler/internal/api/kubernetes"
	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"k8s.io/client-go/tools/leaderelection"
)

const (
	// value defaults.
	defaultGracePeriod       = 15 * time.Minute
	defaultDownscaleReplicas = 0

	leaseName = "downscaler-lease"

	// runtime config defaults.
	defaultInterval = 30 * time.Second
)

func main() {
	// set defaults for runtime configuration
	config := &util.RuntimeConfiguration{
		DryRun:            false,
		Debug:             false,
		Once:              false,
		Interval:          defaultInterval,
		IncludeNamespaces: nil,
		IncludeResources:  []string{"deployments"},
		ExcludeNamespaces: util.RegexList{regexp.MustCompile("kube-system"), regexp.MustCompile("kube-downscaler")},
		ExcludeWorkloads:  nil,
		IncludeLabels:     nil,
		TimeAnnotation:    "",
		Kubeconfig:        "",
	}

	layerCli := values.NewLayer()
	layerEnv := values.NewLayer()

	// set defaults for layers
	layerCli.GracePeriod = defaultGracePeriod
	layerCli.DownscaleReplicas = defaultDownscaleReplicas

	config.ParseConfigFlags()

	layerCli.ParseLayerFlags()

	flag.Parse()

	err := layerEnv.GetLayerFromEnv()
	if err != nil {
		slog.Error("failed to get layer from env", "error", err)
		os.Exit(1)
	}

	if config.Debug || config.DryRun {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	if err = layerCli.CheckForIncompatibleFields(); err != nil {
		slog.Error("found incompatible fields", "error", err)
		os.Exit(1)
	}

	slog.Debug("getting client for kubernetes")

	client, err := kubernetes.NewClient(config.Kubeconfig, config.DryRun)
	if err != nil {
		slog.Error("failed to create new Kubernetes client", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	downscalerNamespace, err := kubernetes.GetCurrentNamespace()
	if err != nil {
		slog.Warn("couldn't get namespace or running outside of cluster; skipping leader election", "error", err)
		slog.Warn("proceeding without leader election, this may cause multiple instances to conflict when modifying the same resources")
		startScanning(client, ctx, &layerCli, &layerEnv, config)

		return
	}

	lease, err := client.CreateLease(leaseName, downscalerNamespace)
	if err != nil {
		slog.Warn("failed to create lease", "error", err)
		os.Exit(1)
	}

	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigs
		cancel()
	}()

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lease,
		ReleaseOnCancel: true,
		LeaseDuration:   30 * time.Second,
		RenewDeadline:   20 * time.Second,
		RetryPeriod:     5 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				slog.Info("started leading")
				err := startScanning(client, ctx, &layerCli, &layerEnv, config)
				if err != nil {
					return
				}
			},
			OnStoppedLeading: func() {
				slog.Info("stopped leading")
			},
			OnNewLeader: func(identity string) {
				slog.Info("new leader elected", "identity", identity)
			},
		},
	})
}

func startScanning(client kubernetes.Client, ctx context.Context, layerCli, layerEnv *values.Layer, config *util.RuntimeConfiguration) error {
	slog.Info("started downscaler")

	err := scanWorkloads(client, ctx, layerCli, layerEnv, config)
	if err != nil {
		slog.Error("failed to scan over workloads",
			"error", err,
			"config", config,
			"CliLayer", *layerCli,
			"EnvLayer", *layerEnv,
		)

		return err
	}

	return nil
}

// scanWorkloads scans over all workloads every scan.
func scanWorkloads(
	client kubernetes.Client,
	ctx context.Context,
	layerCli, layerEnv *values.Layer,
	config *util.RuntimeConfiguration,
) error {
	for {
		slog.Info("scanning workloads")

		workloads, err := client.GetWorkloads(config.IncludeNamespaces, config.IncludeResources, ctx)
		if err != nil {
			return fmt.Errorf("failed to get workloads: %w", err)
		}

		workloads = scalable.FilterExcluded(workloads, config.IncludeLabels, config.ExcludeNamespaces, config.ExcludeWorkloads)
		slog.Info("scanning over workloads matching filters", "amount", len(workloads))

		var waitGroup sync.WaitGroup
		for _, workload := range workloads {
			waitGroup.Add(1)

			go func() {
				slog.Debug("scanning workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())

				defer waitGroup.Done()

				err := scanWorkload(workload, client, ctx, layerCli, layerEnv, config)
				if err != nil {
					slog.Error("failed to scan workload", "error", err, "workload", workload.GetName(), "namespace", workload.GetNamespace())
					return
				}

				slog.Debug("successfully scanned workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
			}()
		}

		waitGroup.Wait()
		slog.Info("successfully scanned all workloads")

		if config.Once {
			slog.Debug("once is set to true, exiting")
			break
		}

		slog.Debug("waiting until next scan", "interval", config.Interval.String())
		time.Sleep(config.Interval)
	}

	return nil
}

// scanWorkload runs a scan on the worklod, determining the scaling and scaling the workload.
func scanWorkload(
	workload scalable.Workload,
	client kubernetes.Client,
	ctx context.Context,
	layerCli, layerEnv *values.Layer,
	config *util.RuntimeConfiguration,
) error {
	resourceLogger := kubernetes.NewResourceLogger(client, workload)

	namespaceAnnotations, err := client.GetNamespaceAnnotations(workload.GetNamespace(), ctx)
	if err != nil {
		return fmt.Errorf("failed to get namespace annotations: %w", err)
	}

	slog.Debug(
		"parsing workload layer from annotations",
		"annotations", workload.GetAnnotations(),
		"name", workload.GetName(),
		"namespace", workload.GetNamespace(),
	)

	layerWorkload := values.NewLayer()
	if err = layerWorkload.GetLayerFromAnnotations(workload.GetAnnotations(), resourceLogger, ctx); err != nil {
		return fmt.Errorf("failed to parse workload layer from annotations: %w", err)
	}

	slog.Debug(
		"parsing namespace layer from annotations",
		"annotations", namespaceAnnotations,
		"name", workload.GetName(),
		"namespace", workload.GetNamespace(),
	)

	layerNamespace := values.NewLayer()
	if err = layerNamespace.GetLayerFromAnnotations(namespaceAnnotations, resourceLogger, ctx); err != nil {
		return fmt.Errorf("failed to parse namespace layer from annotations: %w", err)
	}

	layers := values.Layers{&layerWorkload, &layerNamespace, layerCli, layerEnv}

	slog.Debug("finished parsing all layers", "layers", layers, "workload", workload.GetName(), "namespace", workload.GetNamespace())

	isInGracePeriod, err := layers.IsInGracePeriod(
		config.TimeAnnotation,
		workload.GetAnnotations(),
		workload.GetCreationTimestamp().Time,
		resourceLogger,
		ctx,
	)
	if err != nil {
		return fmt.Errorf("failed to get if workload is on grace period: %w", err)
	}

	if isInGracePeriod {
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

	err = scaleWorkload(scaling, workload, layers, client, ctx)
	if err != nil {
		return fmt.Errorf("failed to scale workload: %w", err)
	}

	return nil
}

// scaleWorkload scales the given workload according to the given wanted scaling state.
func scaleWorkload(
	scaling values.Scaling,
	workload scalable.Workload,
	layers values.Layers,
	client kubernetes.Client,
	ctx context.Context,
) error {
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
