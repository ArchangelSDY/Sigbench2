package metrics

type AgentMetrics struct {
	MachineMemoryUsage      int64
	MachineMemoryPercentage float64
	MachineCPULoad          float64
	ProcessResourceUsages   []*ProcessResourceUsage
}

type ProcessResourceUsage struct {
	Name          string
	CPUPercentage float64
	MemoryRSS     int64
}
