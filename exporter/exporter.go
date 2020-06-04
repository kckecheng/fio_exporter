package exporter

import (
	"sync"

	"github.com/kckecheng/fio_exporter/fiodriver"
	"github.com/prometheus/client_golang/prometheus"
)

type metric struct {
	data   map[string]float64
	mutext sync.Mutex
}

// MetricCollector fio collector object
type MetricCollector struct {
	desc map[string]*prometheus.Desc
}

var latestMetric metric

func init() {
	latestMetric.data = map[string]float64{}
}

// NewCollector init fio collector
func NewCollector() *MetricCollector {
	desc := make(map[string]*prometheus.Desc)
	for _, field := range fiodriver.Fields {
		desc[field] = prometheus.NewDesc(field, field, nil, nil)
	}
	// log.Printf("Prometheus desc: %+v", desc)

	mc := MetricCollector{desc}
	return &mc
}

// UpdateMetric sync the latest fio stats
func UpdateMetric(data map[string]float64) {
	latestMetric.mutext.Lock()
	for k, v := range data {
		latestMetric.data[k] = v
	}
	latestMetric.mutext.Unlock()
}

// Describe implement required interface
func (mc *MetricCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range mc.desc {
		ch <- desc
	}
}

// Collect collect metrics
func (mc *MetricCollector) Collect(ch chan<- prometheus.Metric) {
	latestMetric.mutext.Lock()
	for k, v := range latestMetric.data {
		ch <- prometheus.MustNewConstMetric(
			mc.desc[k],
			prometheus.GaugeValue,
			v,
		)
	}
	latestMetric.mutext.Unlock()
}
