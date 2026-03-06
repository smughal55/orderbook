package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
	"github.com/shazmughal/orderbook/internal/adapter"
)

func main() {
	redisAddr := flag.String("redis", "localhost:6379", "Redis address")
	kafkaBrokers := flag.String("kafka", "localhost:9092", "Kafka broker addresses (comma-separated)")
	kafkaTopic := flag.String("topic", "raw-data", "Kafka topic for normalized records")
	sources := flag.String("sources", "", "Comma-separated WebSocket source specs: name=url,name=url")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Print("ingestion: shutting down...")
		cancel()
	}()

	rdb := redis.NewClient(&redis.Options{Addr: *redisAddr})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}
	log.Print("ingestion: connected to redis")

	brokers := strings.Split(*kafkaBrokers, ",")
	kafkaCfg := sarama.NewConfig()
	kafkaCfg.Producer.Return.Successes = true
	kafkaCfg.Producer.RequiredAcks = sarama.WaitForAll
	producer, err := sarama.NewSyncProducer(brokers, kafkaCfg)
	if err != nil {
		log.Fatalf("kafka producer failed: %v", err)
	}
	defer producer.Close()
	log.Print("ingestion: connected to kafka")

	adapters := buildAdapters(*sources, rdb)
	if len(adapters) == 0 {
		log.Fatal("ingestion: no sources configured. Use -sources name=ws://host:port/path")
	}

	var wg sync.WaitGroup
	for _, a := range adapters {
		wg.Add(1)
		go func(a adapter.Adapter) {
			defer wg.Done()
			runAdapter(ctx, a, producer, *kafkaTopic)
		}(a)
	}

	wg.Wait()
	log.Print("ingestion: all adapters stopped")
}

func buildAdapters(sourcesFlag string, rdb *redis.Client) []adapter.Adapter {
	if sourcesFlag == "" {
		return nil
	}

	var adapters []adapter.Adapter
	for _, spec := range strings.Split(sourcesFlag, ",") {
		parts := strings.SplitN(spec, "=", 2)
		if len(parts) != 2 {
			log.Printf("ingestion: invalid source spec %q, expected name=url", spec)
			continue
		}
		name, url := parts[0], parts[1]
		ws := adapter.NewWebSocketAdapter(name, url)
		dedup := adapter.NewDedupAdapter(ws, rdb)
		adapters = append(adapters, dedup)
		log.Printf("ingestion: registered source %s -> %s", name, url)
	}
	return adapters
}

func runAdapter(ctx context.Context, a adapter.Adapter, producer sarama.SyncProducer, topic string) {
	log.Printf("ingestion: starting adapter %s", a.Name())

	for {
		if err := a.Connect(ctx); err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("ingestion: %s connect error: %v, retrying in 5s", a.Name(), err)
				time.Sleep(5 * time.Second)
				continue
			}
		}
		break
	}
	defer a.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		rec, err := a.Read(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("ingestion: %s read error: %v", a.Name(), err)
				continue
			}
		}

		data, err := rec.Marshal()
		if err != nil {
			log.Printf("ingestion: %s marshal error: %v", a.Name(), err)
			continue
		}

		msg := &sarama.ProducerMessage{
			Topic: topic,
			Key:   sarama.StringEncoder(rec.SourceID),
			Value: sarama.ByteEncoder(data),
		}

		if _, _, err := producer.SendMessage(msg); err != nil {
			log.Printf("ingestion: %s kafka send error: %v", a.Name(), err)
		}
	}
}
