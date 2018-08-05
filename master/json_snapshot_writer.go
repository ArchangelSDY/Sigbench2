package master

import (
	"encoding/json"
	"os"
	"time"
)

type JsonSnapshotWriter struct {
	outDir string
}

func NewJsonSnapshotWriter(outDir string) *JsonSnapshotWriter {
	return &JsonSnapshotWriter{
		outDir: outDir,
	}
}

type JsonSnapshotCountersRow struct {
	Time     string
	Counters map[string]int64
}

func (w *JsonSnapshotWriter) writeRow(filename string, data []byte) error {
	f, err := os.OpenFile(w.outDir+"/"+filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data = append(data, ',')
	data = append(data, '\n')
	f.Write(data)

	return nil
}

func (w *JsonSnapshotWriter) WriteCounters(now time.Time, counters map[string]int64) error {
	row := &JsonSnapshotCountersRow{
		Time:     time.Now().Format(time.RFC3339),
		Counters: counters,
	}
	data, err := json.Marshal(row)
	if err != nil {
		return err
	}

	return w.writeRow("counters.txt", data)
}

type JsonSnapshotMetricsRow struct {
	Time    string
	Metrics []agentMetrics
}

func (w *JsonSnapshotWriter) WriteMetrics(now time.Time, metrics []agentMetrics) error {
	row := &JsonSnapshotMetricsRow{
		Time:    time.Now().Format(time.RFC3339),
		Metrics: metrics,
	}
	data, err := json.Marshal(row)
	if err != nil {
		return err
	}

	return w.writeRow("metrics.txt", data)
}
