package metrics

import (
	"bufio"
	"io/ioutil"
	"os/exec"
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

type ProcessMemoryUsage struct {
	Name string
	RSS  int64
}

func GetProcessMemoryUsages(names []string) ([]*ProcessMemoryUsage, error) {
	usages := make([]*ProcessMemoryUsage, 0, len(names))
	for _, name := range names {
		usages = append(usages, &ProcessMemoryUsage{
			Name: name,
		})
	}

	out, err := exec.Command("ps", "-C", strings.Join(names, ","), "-o", "comm,rss", "--no-headers").Output()
	if err != nil {
		return usages, err
	}

	rssMap := make(map[string]int64)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		rssStr := strings.TrimSpace(parts[1])
		rss, _ := strconv.ParseInt(rssStr, 10, 64)
		if v, ok := rssMap[name]; ok {
			rssMap[name] = v + rss
		} else {
			rssMap[name] = rss
		}
	}

	for _, usage := range usages {
		if rss, ok := rssMap[usage.Name]; ok {
			usage.RSS = rss
		}
	}

	return usages, nil
}
