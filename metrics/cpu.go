package metrics

import (
	"io/ioutil"
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
