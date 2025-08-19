package metrics

// NamespaceMetricsHolder holds the metrics for a specific namespace.
type NamespaceMetricsHolder struct {
	downscaledWorkloads       float64
	upscaledWorkloads         float64
	excludedWorkloads         float64
	invalidScalingValueErrors float64
	conflictErrors            float64
	genericErrors             float64
	savedMemoryBytes          float64
	savedCPUcores             float64
}

func NewNamespaceMetricsHolder() *NamespaceMetricsHolder {
	return &NamespaceMetricsHolder{
		downscaledWorkloads:       0,
		upscaledWorkloads:         0,
		excludedWorkloads:         0,
		invalidScalingValueErrors: 0,
		conflictErrors:            0,
		genericErrors:             0,
		savedMemoryBytes:          0,
		savedCPUcores:             0,
	}
}

func (m *NamespaceMetricsHolder) DownscaledWorkloads() float64 {
	return m.downscaledWorkloads
}

func (m *NamespaceMetricsHolder) UpscaledWorkloads() float64 {
	return m.upscaledWorkloads
}

func (m *NamespaceMetricsHolder) ExcludedWorkloads() float64 {
	return m.excludedWorkloads
}

func (m *NamespaceMetricsHolder) InvalidScalingValueErrors() float64 {
	return m.invalidScalingValueErrors
}

func (m *NamespaceMetricsHolder) ConflictErrors() float64 {
	return m.conflictErrors
}

func (m *NamespaceMetricsHolder) GenericErrors() float64 {
	return m.genericErrors
}

func (m *NamespaceMetricsHolder) SavedMemoryBytes() float64 {
	return m.savedMemoryBytes
}

func (m *NamespaceMetricsHolder) SavedCPUCores() float64 {
	return m.savedCPUcores
}

func (m *NamespaceMetricsHolder) IncrementDownscaledWorkloadsCount() {
	m.downscaledWorkloads++
}

func (m *NamespaceMetricsHolder) IncrementUpscaledWorkloadsCount() {
	m.upscaledWorkloads++
}

func (m *NamespaceMetricsHolder) IncrementExcludedWorkloadsCount() {
	m.excludedWorkloads++
}

func (m *NamespaceMetricsHolder) IncrementInvalidScalingValueErrorsCount() {
	m.invalidScalingValueErrors++
}

func (m *NamespaceMetricsHolder) IncrementConflictErrorsCount() {
	m.conflictErrors++
}

func (m *NamespaceMetricsHolder) IncrementGenericErrorsCount() {
	m.genericErrors++
}

func (m *NamespaceMetricsHolder) IncrementSavedMemoryBytesCount(value float64) {
	m.savedMemoryBytes += value
}

func (m *NamespaceMetricsHolder) IncrementSavedCPUCoresCount(value float64) {
	m.savedCPUcores += value
}
