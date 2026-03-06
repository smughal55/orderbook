package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	IngestionRecordsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ingestion_records_total",
		Help: "Total number of records ingested, by source.",
	}, []string{"source"})

	IngestionDuplicatesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ingestion_duplicates_total",
		Help: "Total number of duplicate records skipped, by source.",
	}, []string{"source"})

	EvaluationLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "evaluation_latency_seconds",
		Help:    "Latency of alert evaluation in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"eval_type"})

	AlertsTriggeredTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "alerts_triggered_total",
		Help: "Total number of alerts triggered, by rule and severity.",
	}, []string{"rule", "severity"})

	DeliverySuccessTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "delivery_success_total",
		Help: "Total successful alert deliveries, by channel.",
	}, []string{"channel"})

	DeliveryFailureTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "delivery_failure_total",
		Help: "Total failed alert deliveries, by channel.",
	}, []string{"channel"})

	KafkaConsumerLag = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kafka_consumer_lag",
		Help: "Kafka consumer lag by topic and partition.",
	}, []string{"topic", "partition"})
)
