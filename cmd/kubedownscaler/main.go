package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
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
	leaseName = "downscaler-lease"
)

var ErrNamespaceAnnotationsFailed = errors.New("failed to get namespace annotations")

func main() {
	config := util.GetDefaultConfig()
	config.ParseConfigFlags()

	err := config.ParseConfigEnvVars()
	if err != nil {
		slog.Error("failed to parse env vars for config", "error", err)
		os.Exit(1)
	}

	layerDefault := values.GetDefaultLayer()
	layerCli := values.NewLayer()
	layerEnv := values.NewLayer()

	err = layerEnv.GetLayerFromEnv()
	if err != nil {
		slog.Error("failed to get layer from env", "error", err)
		os.Exit(1)
	}

	layerCli.ParseLayerFlags()

	flag.Parse()

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

	defer cancel()

	if !config.LeaderElection {
		runWithoutLeaderElection(client, ctx, layerDefault, &layerCli, &layerEnv, config)
		return
	}

	runWithLeaderElection(client, cancel, ctx, layerDefault, &layerCli, &layerEnv, config)
}

func runWithLeaderElection(
	client kubernetes.Client,
	cancel context.CancelFunc,
	ctx context.Context,
	layerDefault, layerCli, layerEnv *values.Layer,
	config *util.RuntimeConfiguration,
) {
	lease, err := client.CreateLease(leaseName)
	if err != nil {
		slog.Error("failed to create lease", "error", err)
		os.Exit(1)
	}

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
				err = startScanning(client, ctx, layerDefault, layerCli, layerEnv, config)
				if err != nil {
					slog.Error("an error occurred while scanning workloads", "error", err)
					cancel()
				}
			},
			OnStoppedLeading: func() {
				slog.Info("stopped leading")
				cancel()
			},
			OnNewLeader: func(identity string) {
				slog.Info("new leader elected", "identity", identity)
			},
		},
	})
}

func runWithoutLeaderElection(
	client kubernetes.Client,
	ctx context.Context,
	layerDefault, layerCli, layerEnv *values.Layer,
	config *util.RuntimeConfiguration,
) {
	slog.Warn("proceeding without leader election; this could cause errors when running with multiple replicas")

	err := startScanning(client, ctx, layerDefault, layerCli, layerEnv, config)
	if err != nil {
		slog.Error("an error occurred while scanning workloads, exiting", "error", err)
		os.Exit(1)
	}
}

func startScanning(
	client kubernetes.Client,
	ctx context.Context,
	layerDefault, layerCli, layerEnv *values.Layer,
	config *util.RuntimeConfiguration,
) error {
	slog.Info("started downscaler")

	for {
		slog.Info("scanning workloads")

		workloads, err := client.GetWorkloads(config.IncludeNamespaces, config.IncludeResources, ctx)
		if err != nil {
			return fmt.Errorf("failed to get workloads: %w", err)
		}

		workloads = scalable.FilterExcluded(workloads, config.IncludeLabels, config.ExcludeNamespaces, config.ExcludeWorkloads)
		slog.Info("scanning over workloads matching filters", "amount", len(workloads))

		annotationsToNamespaces := GetAnnotationsFromNamespace(workloads, client, ctx)

		var waitGroup sync.WaitGroup
		for _, workload := range workloads {
			waitGroup.Add(1)

			go func() {
				slog.Debug("scanning workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())

				defer waitGroup.Done()

				err := scanWorkload(workload, client, ctx, layerDefault, layerCli, layerEnv, config, annotationsToNamespaces)
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

func GetAnnotationsFromNamespace(
	workloads []scalable.Workload,
	client kubernetes.Client,
	ctx context.Context,
) map[string]map[string]string {
	var waitGroup sync.WaitGroup
	var mutex sync.Mutex
	namespaceSet := make(map[string]struct{})
	namespaceToAnnotations := make(map[string]map[string]string)

	for _, workload := range workloads {
		waitGroup.Add(1)

		go func(workload scalable.Workload) {
			defer waitGroup.Done()

			namespace := workload.GetNamespace()

			mutex.Lock()
			if _, exists := namespaceSet[namespace]; !exists {
				namespaceSet[namespace] = struct{}{}

				slog.Debug("visited namespace", "namespace", namespace)
			}

			namespaceToAnnotations[namespace] = make(map[string]string)
			mutex.Unlock()
		}(workload)
	}

	waitGroup.Wait()

	for namespace := range namespaceSet {
		waitGroup.Add(1)

		go func(namespace string) {
			defer waitGroup.Done()

			slog.Debug("fetching namespace annotations", "namespace", namespace)

			annotations, err := client.GetNamespaceAnnotations(namespace, ctx)
			if err != nil {
				slog.Error("failed to get namespace annotations", "namespace", namespace, "error", err)
				return
			}

			mutex.Lock()
			namespaceToAnnotations[namespace] = annotations
			mutex.Unlock()

			slog.Debug("correctly retrieved namespace annotations", "namespace", namespace, "annotations", annotations)
		}(namespace)
	}

	waitGroup.Wait()

	return namespaceToAnnotations
}

// scanWorkload runs a scan on the worklod, determining the scaling and scaling the workload.
func scanWorkload(
	workload scalable.Workload,
	client kubernetes.Client,
	ctx context.Context,
	layerDefault, layerCli, layerEnv *values.Layer,
	config *util.RuntimeConfiguration,
	annotationsToNamespaces map[string]map[string]string,
) error {
	resourceLogger := kubernetes.NewResourceLogger(client, workload)

	namespaceAnnotations, exists := annotationsToNamespaces[workload.GetNamespace()]
	if !exists {
		return fmt.Errorf("%w: namespace %q", ErrNamespaceAnnotationsFailed, workload.GetNamespace())
	}

	slog.Debug(
		"parsing workload layer from annotations",
		"annotations", workload.GetAnnotations(),
		"name", workload.GetName(),
		"namespace", workload.GetNamespace(),
	)

	layerWorkload := values.NewLayer()
	if err := layerWorkload.GetLayerFromAnnotations(workload.GetAnnotations(), resourceLogger, ctx); err != nil {
		return fmt.Errorf("failed to parse workload layer from annotations: %w", err)
	}

	slog.Debug(
		"parsing namespace layer from annotations",
		"annotations", namespaceAnnotations,
		"name", workload.GetName(),
		"namespace", workload.GetNamespace(),
	)

	layerNamespace := values.NewLayer()
	if err := layerNamespace.GetLayerFromAnnotations(namespaceAnnotations, resourceLogger, ctx); err != nil {
		return fmt.Errorf("failed to parse namespace layer from annotations: %w", err)
	}

	layers := values.Layers{&layerWorkload, &layerNamespace, layerCli, layerEnv, layerDefault}

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
