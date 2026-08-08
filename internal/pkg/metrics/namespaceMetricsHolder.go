package metrics

import "sync"

// NamespaceMetricsHolder holds the metrics for a specific namespace.
type NamespaceMetricsHolder struct {
	mu sync.RWMutex

	downscaledWorkloads         float64
	upscaledWorkloads           float64
	excludedWorkloads           float64
	invalidScalingValueErrors   float64
	conflictErrors              float64
	genericErrors               float64
	parsingNamespaceScopeErrors float64
	parsingWorkloadScopeErrors  float64
	savedMemoryBytes            float64
	savedCPUcores               float64
}

func NewNamespaceMetricsHolder() *NamespaceMetricsHolder {
	return &NamespaceMetricsHolder{
		downscaledWorkloads:         0,
		upscaledWorkloads:           0,
		excludedWorkloads:           0,
		invalidScalingValueErrors:   0,
		conflictErrors:              0,
		genericErrors:               0,
		parsingNamespaceScopeErrors: 0,
		parsingWorkloadScopeErrors:  0,
		savedMemoryBytes:            0,
		savedCPUcores:               0,
	}
}

func (m *NamespaceMetricsHolder) DownscaledWorkloads() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.downscaledWorkloads
}

func (m *NamespaceMetricsHolder) UpscaledWorkloads() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.upscaledWorkloads
}

func (m *NamespaceMetricsHolder) ExcludedWorkloads() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.excludedWorkloads
}

func (m *NamespaceMetricsHolder) InvalidScalingValueErrors() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.invalidScalingValueErrors
}

func (m *NamespaceMetricsHolder) ConflictErrors() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.conflictErrors
}

func (m *NamespaceMetricsHolder) GenericErrors() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.genericErrors
}

func (m *NamespaceMetricsHolder) ParsingWorkloadScopeErrors() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.parsingWorkloadScopeErrors
}

func (m *NamespaceMetricsHolder) ParsingNamespaceScopeErrors() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.parsingNamespaceScopeErrors
}

func (m *NamespaceMetricsHolder) SavedMemoryBytes() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.savedMemoryBytes
}

func (m *NamespaceMetricsHolder) SavedCPUCores() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.savedCPUcores
}

func (m *NamespaceMetricsHolder) IncrementDownscaledWorkloadsCount() {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.downscaledWorkloads++
}

func (m *NamespaceMetricsHolder) IncrementUpscaledWorkloadsCount() {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.upscaledWorkloads++
}

func (m *NamespaceMetricsHolder) IncrementExcludedWorkloadsCount() {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.excludedWorkloads++
}

func (m *NamespaceMetricsHolder) IncrementInvalidScalingValueErrorsCount() {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.invalidScalingValueErrors++
}

func (m *NamespaceMetricsHolder) IncrementConflictErrorsCount() {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.conflictErrors++
}

func (m *NamespaceMetricsHolder) IncrementGenericErrorsCount() {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.genericErrors++
}

func (m *NamespaceMetricsHolder) MarkParsingNamespaceScopeError() {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.parsingNamespaceScopeErrors = 1
}

func (m *NamespaceMetricsHolder) IncrementParsingWorkloadScopeErrorsCount() {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.parsingWorkloadScopeErrors++
}

func (m *NamespaceMetricsHolder) IncrementSavedResources(savedResources *SavedResources) {
	if m == nil {
		return
	}

	memoryBytes := savedResources.TotalMemory()
	cpuCores := savedResources.TotalCPU()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.savedMemoryBytes += memoryBytes
	m.savedCPUcores += cpuCores
}
