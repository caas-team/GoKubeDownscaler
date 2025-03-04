package scalable

import "fmt"

type Error struct {
	Message string
	Value   any
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %v", e.Message, e.Value)
}

type RessourceNotSupportedError struct {
	ErrorType string
	Message   string
}

func (i *RessourceNotSupportedError) Error() string {
	return fmt.Sprintf("%s: %s", i.ErrorType, i.Message)
}

// NewRessourceNotSupportedError returns an error when a ressource is not supported.
func NewRessourceNotSupportedError(errorType string, msg string) error {
	return &RessourceNotSupportedError{errorType, msg}
}

type NoReplicasSpecified struct {
	ErrorType string
	Message   string
}

func (i *NoReplicasSpecified) Error() string {
	return fmt.Sprintf("%s: %s", i.ErrorType, i.Message)
}

// NewNoReplicasSpecified returns an error when no replicas are specified.
func NewNoReplicasSpecified(errorType string, msg string) error {
	return &NoReplicasSpecified{errorType, msg}
}

type MinReplicasBoundsExceeded struct {
	ErrorType string
	Message   string
}

func (i *MinReplicasBoundsExceeded) Error() string {
	return fmt.Sprintf("%s: %s", i.ErrorType, i.Message)
}

// NewMinReplicasBoundsExceeded returns an error when the min replicas bounds are exceeded.
func NewMinReplicasBoundsExceeded(errorType string, msg string) error {
	return &MinReplicasBoundsExceeded{errorType, msg}
}

type CronJobError struct {
	ErrorType string
	Message   string
}

func (i *CronJobError) Error() string {
	return fmt.Sprintf("%s: %s", i.ErrorType, i.Message)
}

// NewCronJobError returns an error when a cron job fails.
func NewCronJobError(errorType string, msg string) error {
	return &CronJobError{errorType, msg}
}

type DaemonSetError struct {
	ErrorType string
	Message   string
}

func (i *DaemonSetError) Error() string {
	return fmt.Sprintf("%s: %s", i.ErrorType, i.Message)
}

// NewDaemonSetError returns an error when a daemon set fails.
func NewDaemonSetError(errorType string, msg string) error {
	return &DaemonSetError{errorType, msg}
}

type DeploymentError struct {
	ErrorType string
	Message   string
}

func (i *DeploymentError) Error() string {
	return fmt.Sprintf("%s: %s", i.ErrorType, i.Message)
}

// NewDeploymentError returns an error when a deployment fails.
func NewDeploymentError(errorType string, msg string) error {
	return &DeploymentError{errorType, msg}
}

type HorizontalPodAutscalerError struct {
	ErrorType string
	Message   string
}

func (i *HorizontalPodAutscalerError) Error() string {
	return fmt.Sprintf("%s: %s", i.ErrorType, i.Message)
}

// NewHorizontalPodAutscalerError returns an error when a horizontal pod autscaler fails.
func NewHorizontalPodAutscalerError(errorType string, msg string) error {
	return &HorizontalPodAutscalerError{errorType, msg}
}

type FailedToGetJobsError struct {
	ErrorType string
	Message   string
}

func (i *FailedToGetJobsError) Error() string {
	return fmt.Sprintf("%s: %s", i.ErrorType, i.Message)
}

// NewFailedToGetJobsError returns an error when getting jobs fails.
func NewFailedToGetJobsError(errorType string, msg string) error {
	return &FailedToGetJobsError{errorType, msg}
}

type PodDistributionError struct {
	ErrorType string
	Message   string
}

func (i *PodDistributionError) Error() string {
	return fmt.Sprintf("%s: %s", i.ErrorType, i.Message)
}

// NewPodDistributionError returns an error when pod distribution fails.
func NewPodDistributionError(errorType string, msg string) error {
	return &PodDistributionError{errorType, msg}
}

type PrometheusError struct {
	ErrorType string
	Message   string
}

func (i *PrometheusError) Error() string {
	return fmt.Sprintf("%s: %s", i.ErrorType, i.Message)
}

// NewPrometheusError returns an error when a prometheus query fails.
func NewPrometheusError(errorType string, msg string) error {
	return &PrometheusError{errorType, msg}
}

type RolloutError struct {
	ErrorType string
	Message   string
}

func (i *RolloutError) Error() string {
	return fmt.Sprintf("%s: %s", i.ErrorType, i.Message)
}

// NewRolloutError returns an error when a rollout fails.
func NewRolloutError(errorType string, msg string) error {
	return &RolloutError{errorType, msg}
}

type ScaledObjectError struct {
	ErrorType string
	Message   string
}

func (i *ScaledObjectError) Error() string {
	return fmt.Sprintf("%s: %s", i.ErrorType, i.Message)
}

// NewScaledObjectError returns an error when a scaled object fails.
func NewScaledObjectError(errorType string, msg string) error {
	return &ScaledObjectError{errorType, msg}
}

type StacksError struct {
	ErrorType string
	Message   string
}

func (i *StacksError) Error() string {
	return fmt.Sprintf("%s: %s", i.ErrorType, i.Message)
}

// NewStacksError returns an error when a stack fails.
func NewStacksError(errorType string, msg string) error {
	return &StacksError{errorType, msg}
}

type StatefulSetError struct {
	ErrorType string
	Message   string
}

func (i *StatefulSetError) Error() string {
	return fmt.Sprintf("%s: %s", i.ErrorType, i.Message)
}

// NewStatefulSetError returns an error when a stateful set fails.
func NewStatefulSetError(errorType string, msg string) error {
	return &StatefulSetError{errorType, msg}
}

type WorkloadError struct {
	ErrorType string
	Message   string
}

func (i *WorkloadError) Error() string {
	return fmt.Sprintf("%s: %s", i.ErrorType, i.Message)
}

// NewWorkloadError returns an error when a workload fails.
func NewWorkloadError(errorType string, msg string) error {
	return &WorkloadError{errorType, msg}
}
