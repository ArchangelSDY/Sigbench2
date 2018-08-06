package metrics

type AgentMetrics struct {
	MachineMemoryUsage      int64
	MachineMemoryPercentage float64
	MachineCPULoad          float64
	ProcessMemoryUsages     []*ProcessMemoryUsage
	ProcessCPUUsages        []*ProcessCPUUsage
}

func (m *AgentMetrics) ToMap() map[string]interface{} {
	ret := map[string]interface{}{
		"machineMemoryUsage":      m.MachineMemoryUsage,
		"machineMemoryPercentage": m.MachineMemoryPercentage,
		"machineCPULoad":          m.MachineCPULoad,
	}

	for _, usage := range m.ProcessMemoryUsages {
		ret["processMemoryRSS:"+usage.Name] = usage.RSS
	}

	for _, usage := range m.ProcessCPUUsages {
		ret["processCPUPercentage:"+usage.Name] = usage.Percentage
	}

	return ret
}
