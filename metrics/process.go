package metrics

import (
	"bufio"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var procUsageRegex = regexp.MustCompile("\\s+")

func GetProcessResourceUsages(names []string) ([]*ProcessResourceUsage, error) {
	usages := make([]*ProcessResourceUsage, 0, len(names))
	for _, name := range names {
		usages = append(usages, &ProcessResourceUsage{
			Name: name,
		})
	}

	out, err := exec.Command("ps", "-C", strings.Join(names, ","), "-o", "comm,rss,%cpu", "--no-headers").Output()
	if err != nil {
		return usages, err
	}

	rssMap := make(map[string]int64)
	cpuMap := make(map[string]float64)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := procUsageRegex.Split(line, 3)
		if len(parts) < 3 {
			continue
		}

		name := parts[0]
		rssStr := parts[1]
		cpuStr := parts[2]

		rss, _ := strconv.ParseInt(rssStr, 10, 64)
		if v, ok := rssMap[name]; ok {
			rssMap[name] = v + rss
		} else {
			rssMap[name] = rss
		}

		cpu, _ := strconv.ParseFloat(cpuStr, 64)
		if v, ok := cpuMap[name]; ok {
			cpuMap[name] = v + cpu
		} else {
			cpuMap[name] = cpu
		}
	}

	for _, usage := range usages {
		if rss, ok := rssMap[usage.Name]; ok {
			usage.MemoryRSS = rss
		}
		if cpu, ok := cpuMap[usage.Name]; ok {
			usage.CPUPercentage = cpu
		}
	}

	return usages, nil
}
