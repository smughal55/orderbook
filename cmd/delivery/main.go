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

	"github.com/IBM/sarama"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shazmughal/orderbook/internal/delivery"
	"github.com/shazmughal/orderbook/internal/model"
)

func main() {
	kafkaBrokers := flag.String("kafka", "localhost:9092", "Kafka broker addresses (comma-separated)")
	inputTopic := flag.String("topic", "alerts-triggered", "Kafka topic to consume")
	deadLetterTopic := flag.String("deadletter", "alerts-deadletter", "Kafka dead-letter topic")
	metricsAddr := flag.String("metrics", ":2114", "Prometheus metrics address")
	consumerGroup := flag.String("group", "delivery-group", "Kafka consumer group")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Print("delivery: shutting down...")
		cancel()
	}()

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		log.Printf("delivery: metrics on %s", *metricsAddr)
		if err := http.ListenAndServe(*metricsAddr, mux); err != nil {
			log.Printf("delivery: metrics server error: %v", err)
		}
	}()

	router := delivery.NewRouter(
		delivery.NewEmailHandler(),
		delivery.NewSMSHandler(),
		delivery.NewWebhookHandler(),
	)

	brokers := strings.Split(*kafkaBrokers, ",")

	// Producer for dead-letter queue
	prodCfg := sarama.NewConfig()
	prodCfg.Producer.Return.Successes = true
	producer, err := sarama.NewSyncProducer(brokers, prodCfg)
	if err != nil {
		log.Fatalf("delivery: kafka producer: %v", err)
	}
	defer producer.Close()

	consCfg := sarama.NewConfig()
	consCfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	consCfg.Consumer.Offsets.Initial = sarama.OffsetNewest

	group, err := sarama.NewConsumerGroup(brokers, *consumerGroup, consCfg)
	if err != nil {
		log.Fatalf("delivery: kafka consumer group: %v", err)
	}
	defer group.Close()

	handler := &deliveryHandler{
		ctx:             ctx,
		router:          router,
		producer:        producer,
		deadLetterTopic: *deadLetterTopic,
	}

	log.Printf("delivery: consuming from %s", *inputTopic)
	for {
		if err := group.Consume(ctx, []string{*inputTopic}, handler); err != nil {
			log.Printf("delivery: consumer error: %v", err)
		}
		if ctx.Err() != nil {
			return
		}
	}
}

type deliveryHandler struct {
	ctx             context.Context
	router          *delivery.Router
	producer        sarama.SyncProducer
	deadLetterTopic string
}

func (h *deliveryHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *deliveryHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *deliveryHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		alert, err := model.UnmarshalTriggeredAlert(msg.Value)
		if err != nil {
			log.Printf("delivery: unmarshal error: %v", err)
			session.MarkMessage(msg, "")
			continue
		}

		h.router.Deliver(h.ctx, alert)

		session.MarkMessage(msg, "")
	}
	return nil
}
