package metrics

type SavedResources struct {
	totalSavedCPU    float64
	totalSavedMemory float64
}

func NewSavedResources(cpu, memory float64) *SavedResources {
	return &SavedResources{
		totalSavedCPU:    cpu,
		totalSavedMemory: memory,
	}
}

func (sr *SavedResources) TotalCPU() float64 {
	return sr.totalSavedCPU
}

func (sr *SavedResources) TotalMemory() float64 {
	return sr.totalSavedMemory
}
