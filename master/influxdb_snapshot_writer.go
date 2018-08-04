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

func (w *InfluxDBSnapshotWriter) WriteMetrics(now time.Time, metrics map[string]metrics.AgentMetrics) error {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  w.db,
		Precision: "s",
	})
	if err != nil {
		return err
	}

	for agent, data := range metrics {
		tags := map[string]string{
			"agent": agent,
		}
		pt, err := client.NewPoint("metrics", tags, data.ToMap(), now)
		if err != nil {
			return err
		}
		bp.AddPoint(pt)
	}

	if err = w.client.Write(bp); err != nil {
		return err
	}

	return nil
}
