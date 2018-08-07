package master

import (
	"log"
	"time"

	"aspnet.com/metrics"

	"github.com/influxdata/influxdb/client/v2"
)

type InfluxDBSnapshotWriter struct {
	db     string
	client client.Client
}

func NewInfluxDBSnapshotWriter(addr, db string) *InfluxDBSnapshotWriter {
	c, err := client.NewHTTPClient(client.HTTPConfig{Addr: addr})
	if err != nil {
		log.Fatal(err)
	}

	return &InfluxDBSnapshotWriter{
		db:     db,
		client: c,
	}
}

func (w *InfluxDBSnapshotWriter) WriteCounters(now time.Time, counters map[string]int64) error {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  w.db,
		Precision: "s",
	})
	if err != nil {
		return err
	}

	tags := map[string]string{}
	fields := make(map[string]interface{})
	for k, v := range counters {
		fields[k] = v
	}
	pt, err := client.NewPoint("counters", tags, fields, now)
	if err != nil {
		return err
	}
	bp.AddPoint(pt)

	if err = w.client.Write(bp); err != nil {
		return err
	}

	return nil
}

func addAgentMetrics(bp client.BatchPoints, data *metrics.AgentMetrics, now time.Time, commonTags map[string]string) error {
	fields := map[string]interface{}{
		"machineMemoryUsage":      data.MachineMemoryUsage,
		"machineMemoryPercentage": data.MachineMemoryPercentage,
		"machineCPULoad":          data.MachineCPULoad,
	}
	pt, err := client.NewPoint("metrics", commonTags, fields, now)
	if err != nil {
		return err
	}
	bp.AddPoint(pt)
	return nil
}

func addProcessMetrics(bp client.BatchPoints, data *metrics.ProcessResourceUsage, now time.Time, commonTags map[string]string) error {
	fields := map[string]interface{}{
		"processMemoryRSS":     data.MemoryRSS,
		"processCPUPercentage": data.CPUPercentage,
	}
	tags := map[string]string{
		"process": data.Name,
	}
	for k, v := range commonTags {
		tags[k] = v
	}
	pt, err := client.NewPoint("metrics", tags, fields, now)
	if err != nil {
		return err
	}
	bp.AddPoint(pt)
	return nil
}

func (w *InfluxDBSnapshotWriter) WriteMetrics(now time.Time, metrics []agentMetrics) error {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  w.db,
		Precision: "s",
	})
	if err != nil {
		return err
	}

	for _, m := range metrics {
		commonTags := map[string]string{
			"agent":     m.Agent,
			"agentRole": m.AgentRole,
		}

		addAgentMetrics(bp, &m.Metrics, now, commonTags)

		for _, usage := range m.Metrics.ProcessResourceUsages {
			addProcessMetrics(bp, usage, now, commonTags)
		}
	}

	if err = w.client.Write(bp); err != nil {
		return err
	}

	return nil
}
