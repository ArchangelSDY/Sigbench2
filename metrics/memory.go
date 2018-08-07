package metrics

import (
	"io/ioutil"
	"strconv"
	"strings"
)

type MachineMemoryUsage struct {
	Total     int64
	Free      int64
	Available int64
}

func GetMachineMemoryUsage() (*MachineMemoryUsage, error) {
	var usage MachineMemoryUsage

	data, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) != 2 {
			continue
		}
		key := strings.TrimSpace(fields[0])
		value := strings.TrimSpace(fields[1])
		value = strings.Replace(value, " kB", "", -1)

		t, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, err
		}

		switch key {
		case "MemTotal":
			usage.Total = t * 1024
		case "MemFree":
			usage.Free = t * 1024
		case "MemAvailable":
			usage.Available = t * 1024
		}
	}

	return &usage, nil
}
