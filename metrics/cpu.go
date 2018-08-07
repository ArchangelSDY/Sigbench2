package metrics

import (
	"bufio"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
)

func GetCPULoad() (float64, error) {
	data, err := ioutil.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, err
	}

	line := string(data)
	parts := strings.Split(line, " ")
	loadAvg1, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, err
	}

	return loadAvg1, nil
}

type ProcessCPUUsage struct {
	Name       string
	Percentage float64
}

func GetProcessCPUUsages(names []string) ([]*ProcessCPUUsage, error) {
	usages := make([]*ProcessCPUUsage, 0, len(names))
	for _, name := range names {
		usages = append(usages, &ProcessCPUUsage{
			Name: name,
		})
	}

	out, err := exec.Command("ps", "-C", strings.Join(names, ","), "-o", "comm,%%cpu", "--no-headers").Output()
	if err != nil {
		return usages, err
	}

	percentageMap := make(map[string]float64)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		percentageStr := strings.TrimSpace(parts[1])
		percentage, _ := strconv.ParseFloat(percentageStr, 64)
		if v, ok := percentageMap[name]; ok {
			percentageMap[name] = v + percentage
		} else {
			percentageMap[name] = percentage
		}
	}

	for _, usage := range usages {
		if percentage, ok := percentageMap[usage.Name]; ok {
			usage.Percentage = percentage
		}
	}

	return usages, nil
}
