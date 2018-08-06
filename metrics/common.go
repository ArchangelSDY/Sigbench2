package metrics

type AgentMetrics struct {
	MachineMemoryUsage      int64
	MachineMemoryPercentage float64
	MachineCPULoad          float64
	ProcessMemoryUsages     []*ProcessMemoryUsage
	ProcessCPUUsages        []*ProcessCPUUsage
}

func (a *AgentMetrics) GetUntaggedMap() map[string]interface{} {
	return map[string]interface{}{
		"machineMemoryUsage":      a.MachineMemoryUsage,
		"machineMemoryPercentage": a.MachineMemoryPercentage,
		"machineCPULoad":          a.MachineCPULoad,
	}
}

func (a *AgentMetrics) GetTaggedMap() ([]map[string]interface{}, []map[string]string) {
	maps := make([]map[string]interface{}, 0, len(a.ProcessMemoryUsages))
	tags := make([]map[string]string, 0, len(a.ProcessMemoryUsages))

	for i, usage := range a.ProcessMemoryUsages {
		m := make(map[string]interface{})
		m["processMemoryRSS"] = usage.RSS
		m["processCPUPercentage"] = a.ProcessCPUUsages[i].Percentage
		maps = append(maps, m)

		tags = append(tags, map[string]string{
			"process": usage.Name,
		})
	}

	return maps, tags
}
