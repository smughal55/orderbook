package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/shazmughal/orderbook/internal/evaluator"
	"github.com/shazmughal/orderbook/internal/metrics"
	"github.com/shazmughal/orderbook/internal/model"
	"github.com/shazmughal/orderbook/internal/ruleengine"
)

func main() {
	redisAddr := flag.String("redis", "localhost:6379", "Redis address")
	kafkaBrokers := flag.String("kafka", "localhost:9092", "Kafka broker addresses (comma-separated)")
	inputTopic := flag.String("input-topic", "raw-data", "Kafka topic to consume")
	outputTopic := flag.String("output-topic", "alerts-triggered", "Kafka topic for triggered alerts")
	metricsAddr := flag.String("metrics", ":2113", "Prometheus metrics address")
	consumerGroup := flag.String("group", "processor-group", "Kafka consumer group")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Print("processor: shutting down...")
		cancel()
	}()

	// Metrics server
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		log.Printf("processor: metrics on %s", *metricsAddr)
		if err := http.ListenAndServe(*metricsAddr, mux); err != nil {
			log.Printf("processor: metrics server error: %v", err)
		}
	}()

	rdb := redis.NewClient(&redis.Options{Addr: *redisAddr})
	defer rdb.Close()

	singleEval := evaluator.NewSingleRecordEvaluator()
	trendEval := evaluator.NewTrendEvaluator(rdb)
	engine := ruleengine.New(rdb)

	// TODO: load rules from Convex via HTTP API; for now use sample rules
	sampleRules := []model.AlertRule{
		{
			ID:            "rule-1",
			Name:          "High Price Alert",
			SourceFilter:  "*",
			EvalType:      model.EvalTypeSingle,
			Condition:     `payload.price > 150`,
			DeliveryChannels: []model.DeliveryChannel{
				{Type: "webhook", Target: "http://localhost:8888/alerts"},
			},
			CooldownSeconds: 60,
			Enabled:         true,
		},
		{
			ID:              "rule-2",
			Name:            "Price Avg Spike",
			SourceFilter:    "*",
			EvalType:        model.EvalTypeTrend,
			Condition:       `value > 120`,
			WindowSeconds:   300,
			Aggregation:     model.AggregationAvg,
			Field:           "price",
			DeliveryChannels: []model.DeliveryChannel{
				{Type: "email", Target: "ops@example.com"},
			},
			CooldownSeconds: 300,
			Enabled:         true,
		},
	}

	if err := singleEval.LoadRules(sampleRules); err != nil {
		log.Fatalf("processor: load single rules: %v", err)
	}
	if err := trendEval.LoadRules(sampleRules); err != nil {
		log.Fatalf("processor: load trend rules: %v", err)
	}

	// Kafka producer for triggered alerts
	brokers := strings.Split(*kafkaBrokers, ",")
	prodCfg := sarama.NewConfig()
	prodCfg.Producer.Return.Successes = true
	producer, err := sarama.NewSyncProducer(brokers, prodCfg)
	if err != nil {
		log.Fatalf("processor: kafka producer: %v", err)
	}
	defer producer.Close()

	// Kafka consumer
	consCfg := sarama.NewConfig()
	consCfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	consCfg.Consumer.Offsets.Initial = sarama.OffsetNewest

	group, err := sarama.NewConsumerGroup(brokers, *consumerGroup, consCfg)
	if err != nil {
		log.Fatalf("processor: kafka consumer group: %v", err)
	}
	defer group.Close()

	handler := &consumerHandler{
		ctx:        ctx,
		singleEval: singleEval,
		trendEval:  trendEval,
		engine:     engine,
		producer:   producer,
		outputTopic: *outputTopic,
	}

	log.Printf("processor: consuming from %s", *inputTopic)
	for {
		if err := group.Consume(ctx, []string{*inputTopic}, handler); err != nil {
			log.Printf("processor: consumer error: %v", err)
		}
		if ctx.Err() != nil {
			return
		}
	}
}

type consumerHandler struct {
	ctx         context.Context
	singleEval  *evaluator.SingleRecordEvaluator
	trendEval   *evaluator.TrendEvaluator
	engine      *ruleengine.Engine
	producer    sarama.SyncProducer
	outputTopic string
}

func (h *consumerHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *consumerHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *consumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		record, err := model.UnmarshalDataRecord(msg.Value)
		if err != nil {
			log.Printf("processor: unmarshal error: %v", err)
			session.MarkMessage(msg, "")
			continue
		}

		start := time.Now()

		// Single-record evaluation
		singleCandidates := h.singleEval.Evaluate(record)
		metrics.EvaluationLatency.WithLabelValues("single").Observe(time.Since(start).Seconds())

		// Trend evaluation
		trendStart := time.Now()
		trendCandidates := h.trendEval.Evaluate(h.ctx, record)
		metrics.EvaluationLatency.WithLabelValues("trend").Observe(time.Since(trendStart).Seconds())

		allCandidates := append(singleCandidates, trendCandidates...)
		triggered := h.engine.Process(h.ctx, allCandidates)

		for _, alert := range triggered {
			metrics.AlertsTriggeredTotal.WithLabelValues(alert.RuleName, string(alert.Severity)).Inc()

			data, err := alert.Marshal()
			if err != nil {
				log.Printf("processor: marshal alert: %v", err)
				continue
			}

			outMsg := &sarama.ProducerMessage{
				Topic: h.outputTopic,
				Key:   sarama.StringEncoder(alert.AlertID),
				Value: sarama.ByteEncoder(data),
			}
			if _, _, err := h.producer.SendMessage(outMsg); err != nil {
				log.Printf("processor: send alert: %v", err)
			}
		}

		session.MarkMessage(msg, "")
	}
	return nil
}
