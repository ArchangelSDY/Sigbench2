package metrics

type AgentMetrics struct {
	MachineMemoryUsage      int64
	MachineMemoryPercentage float64
	MachineCPULoad          float64
}

func (m *AgentMetrics) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"machineMemoryUsage":      m.MachineMemoryUsage,
		"machineMemoryPercentage": m.MachineMemoryPercentage,
		"machineCPULoad":          m.MachineCPULoad,
	}
}
