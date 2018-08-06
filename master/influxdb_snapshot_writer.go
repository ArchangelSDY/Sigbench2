package master

import (
	"log"
	"time"

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
		pt, err := client.NewPoint("metrics", commonTags, m.Metrics.GetUntaggedMap(), now)
		if err != nil {
			return err
		}
		bp.AddPoint(pt)

		taggedMaps, tags := m.Metrics.GetTaggedMap()
		for i, m := range taggedMaps {
			t := tags[i]
			for k, v := range commonTags {
				t[k] = v
			}

			pt, err := client.NewPoint("metrics", t, m, now)
			if err != nil {
				return err
			}
			bp.AddPoint(pt)
		}
	}

	if err = w.client.Write(bp); err != nil {
		return err
	}

	return nil
}
