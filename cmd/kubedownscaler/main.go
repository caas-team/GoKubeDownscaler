package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/caas-team/gokubedownscaler/internal/api/kubernetes"
	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/client-go/tools/leaderelection"
)

const (
	leaseName = "downscaler-lease"
)

func main() {
	config, scopeDefault, scopeCli, scopeEnv := initComponent()

	slog.Debug("getting client for kubernetes")

	client, err := kubernetes.NewClient(config.Kubeconfig, config.DryRun)
	if err != nil {
		slog.Error("failed to create new Kubernetes client", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	if !config.LeaderElection {
		runWithoutLeaderElection(client, ctx, scopeDefault, scopeCli, scopeEnv, config)
		return
	}

	runWithLeaderElection(client, cancel, ctx, scopeDefault, scopeCli, scopeEnv, config)
}

// runWithLeaderElection runs the downscaler with leader election enabled.
func runWithLeaderElection(
	client kubernetes.Client,
	cancel context.CancelFunc,
	ctx context.Context,
	scopeDefault, scopeCli, scopeEnv *values.Scope,
	config *runtimeConfiguration,
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
				err = startScanning(client, ctx, scopeDefault, scopeCli, scopeEnv, config)
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

// runWithoutLeaderElection runs the downscaler without leader election enabled.
func runWithoutLeaderElection(
	client kubernetes.Client,
	ctx context.Context,
	scopeDefault, scopeCli, scopeEnv *values.Scope,
	config *runtimeConfiguration,
) {
	slog.Warn("proceeding without leader election; this could cause errors when running with multiple replicas")

	err := startScanning(client, ctx, scopeDefault, scopeCli, scopeEnv, config)
	if err != nil {
		slog.Error("an error occurred while scanning workloads, exiting", "error", err)
		os.Exit(1)
	}
}

// startScanning periodically triggers a scan on all workloads.
func startScanning(
	client kubernetes.Client,
	ctx context.Context,
	scopeDefault, scopeCli, scopeEnv *values.Scope,
	config *runtimeConfiguration,
) error {
	slog.Info("started downscaler")

	for {
		slog.Info("scanning workloads")

		workloads, err := client.GetWorkloads(config.IncludeNamespaces, config.IncludeResources, ctx)
		if err != nil {
			return fmt.Errorf("failed to get workloads: %w", err)
		}

		workloads = scalable.FilterExcluded(
			workloads,
			config.IncludeLabels,
			config.ExcludeNamespaces,
			config.ExcludeWorkloads,
		)
		slog.Info("scanning over workloads matching filters", "amount", len(workloads))

		namespaceScopes, err := client.GetNamespacesScopes(workloads, ctx)
		if err != nil {
			return fmt.Errorf("failed to get namespace annotations: %w", err)
		}

		var waitGroup sync.WaitGroup
		for _, workload := range workloads {
			waitGroup.Add(1)

			go func(workload scalable.Workload) {
				slog.Debug("scanning workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())

				defer waitGroup.Done()

				err := scanWorkload(workload, client, ctx, scopeDefault, scopeCli, scopeEnv, namespaceScopes, config)
				if err != nil {
					slog.Error("failed to scan workload", "error", err, "workload", workload.GetName(), "namespace", workload.GetNamespace())
					return
				}

				slog.Debug("successfully scanned workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
			}(workload)
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

// attemptScaling handles retries for scaling a workload in case of conflicts.
func attemptScaling(
	client kubernetes.Client,
	ctx context.Context,
	scaling values.Scaling,
	workload scalable.Workload,
	scopes values.Scopes,
	config *runtimeConfiguration,
) error {
	for retry := range config.MaxRetriesOnConflict + 1 {
		err := scaleWorkload(scaling, workload, scopes, client, ctx)
		if err != nil {
			if !strings.Contains(err.Error(), registry.OptimisticLockErrorMsg) {
				return fmt.Errorf("failed to scale workload: %w", err)
			}

			slog.Warn("workload modified, retrying", "attempt", retry+1, "workload", workload.GetName(), "namespace", workload.GetNamespace())

			err = client.RegetWorkload(workload, ctx)
			if err != nil {
				return fmt.Errorf("failed to fetch updated workload: %w", err)
			}

			continue
		}

		slog.Debug("successfully scaled workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())

		return nil
	}

	slog.Error("failed to scale workload", "attempts", config.MaxRetriesOnConflict+1)

	return newMaxRetriesExceeded(config.MaxRetriesOnConflict)
}

// scanWorkload runs a scan on the workload, determining the scaling and scaling the workload.
//

func scanWorkload(
	workload scalable.Workload,
	client kubernetes.Client,
	ctx context.Context,
	scopeDefault, scopeCli, scopeEnv *values.Scope,
	namespaceScopes map[string]*values.Scope,
	config *runtimeConfiguration,
) error {
	resourceLogger := kubernetes.NewResourceLoggerForWorkload(client, workload)

	var err error
	slog.Debug(
		"parsing workload scope from annotations",
		"annotations", workload.GetAnnotations(),
		"name", workload.GetName(),
		"namespace", workload.GetNamespace(),
	)

	scopeWorkload := values.NewScope()
	if err = scopeWorkload.GetScopeFromAnnotations(workload.GetAnnotations(), resourceLogger, ctx); err != nil {
		return fmt.Errorf("failed to parse workload scope from annotations: %w", err)
	}

	scopeNamespace, exists := namespaceScopes[workload.GetNamespace()]
	if !exists {
		return newNamespaceScopeRetrieveError(workload.GetNamespace())
	}

	scopes := values.Scopes{scopeWorkload, scopeNamespace, scopeCli, scopeEnv, scopeDefault}

	slog.Debug("finished parsing all scopes", "scopes", scopes, "workload", workload.GetName(), "namespace", workload.GetNamespace())

	isInGracePeriod, err := scopes.IsInGracePeriod(
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

	if scopes.GetExcluded() {
		slog.Debug("workload is excluded, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return nil
	}

	scaling := scopes.GetCurrentScaling()

	err = attemptScaling(client, ctx, scaling, workload, scopes, config)
	if err != nil {
		return fmt.Errorf("failed to scale workload: %w", err)
	}

	if scopes.GetScaleChildren() {
		childrenWorkloads, err := client.GetChildrenWorkloads(workload, ctx)
		if err != nil {
			return fmt.Errorf("failed to get children workloads: %w", err)
		}

		scaleWorkloads(scaling, childrenWorkloads, scopes, client, ctx, config)
	}

	return nil
}

// scaleWorkloads scales the given workloads to the specified scaling asynchronously.
func scaleWorkloads(
	scaling values.Scaling,
	workloads []scalable.Workload,
	scopes values.Scopes,
	client kubernetes.Client,
	ctx context.Context,
	config *runtimeConfiguration,
) {
	for _, workload := range workloads {
		go func(workload scalable.Workload) {
			err := attemptScaling(client, ctx, scaling, workload, scopes, config)
			if err != nil {
				slog.Error("failed to scale workload", "error", err, "workload", workload.GetName(), "namespace", workload.GetNamespace())
			}
		}(workload)
	}
}

// scaleWorkload scales the given workload according to the given wanted scaling state.
func scaleWorkload(
	scaling values.Scaling,
	workload scalable.Workload,
	scopes values.Scopes,
	client kubernetes.Client,
	ctx context.Context,
) error {
	if scaling == values.ScalingNone {
		slog.Debug("scaling is not set by any scope, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return nil
	}

	if scaling == values.ScalingIgnore {
		slog.Debug("scaling is ignored, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return nil
	}

	if scaling == values.ScalingMultiple {
		return newScalingInvalidError(
			`scaling values matched to multiple states.
this is the result of a faulty configuration where on a scope there is multiple values with the same priority
setting different scaling states at the same time (e.g. downtime-period and uptime-period or force-downtime and force-uptime)`,
		)
	}

	if scaling == values.ScalingDown {
		slog.Debug("downscaling workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())

		downscaleReplicas, err := scopes.GetDownscaleReplicas()
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
