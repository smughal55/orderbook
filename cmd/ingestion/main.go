package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/shazmughal/orderbook/internal/bus"
	"github.com/shazmughal/orderbook/internal/config"
	"github.com/shazmughal/orderbook/internal/source/wss"
)

func main() {
	if err := run(); err != nil {
		slog.Error("ingestion: fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// --- Config ---
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	slog.Info("ingestion: starting",
		"nats_url", cfg.NATSUrl,
		"wss_url", cfg.WSSUrl,
		"log_level", cfg.LogLevel,
	)

	// --- Context wired to OS signals ---
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// --- NATS publisher ---
	pub, err := bus.NewNATSPublisher(cfg.NATSUrl)
	if err != nil {
		return fmt.Errorf("nats publisher: %w", err)
	}
	defer pub.Close() //nolint:errcheck

	// --- WSS adapter ---
	adapter := wss.NewWSSAdapter(cfg.WSSUrl)
	if err := adapter.Connect(ctx); err != nil {
		return fmt.Errorf("wss adapter connect: %w", err)
	}
	defer adapter.Close() //nolint:errcheck

	slog.Info("ingestion: connected, consuming records")

	// --- Read loop ---
	var dropped atomic.Int64
	for record := range adapter.Records() {
		data, marshalErr := json.Marshal(record)
		if marshalErr != nil {
			slog.Error("ingestion: marshal failed, dropping record",
				"error", marshalErr,
				"source", record.Source,
			)
			dropped.Add(1)
			continue
		}

		subject := "records." + record.Source
		if pubErr := pub.Publish(ctx, subject, data); pubErr != nil {
			n := dropped.Add(1)
			slog.Error("ingestion: publish failed, record dropped",
				"error", pubErr,
				"subject", subject,
				"total_dropped", n,
			)
		}
	}

	slog.Info("ingestion: records channel closed, shutting down",
		"total_dropped", dropped.Load(),
	)
	return nil
}
