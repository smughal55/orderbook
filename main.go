package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shazmughal/orderbook/internal/source/wss"
)

func main() {
	wsURL := flag.String("url", "ws://localhost:8080/ws", "WebSocket server URL")
	mock := flag.Bool("mock", false, "start a built-in mock WebSocket server")
	mockAddr := flag.String("mock-addr", ":8080", "address for the mock server")
	useAdapter := flag.Bool("adapter", false, "use the SourceAdapter path instead of the legacy client")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		fmt.Println()
		log.Print("shutting down...")
		cancel()
	}()

	if *mock {
		ms := NewMockServer(*mockAddr)
		go func() {
			if err := ms.Start(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("mock server error: %v", err)
			}
		}()
		defer func() {
			shutCtx, shutCancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer shutCancel()
			ms.Shutdown(shutCtx)
		}()

		time.Sleep(50 * time.Millisecond) // let the server start
		log.Printf("mock server started on %s", *mockAddr)
	}

	if *useAdapter {
		adapter := wss.NewWSSAdapter(*wsURL)
		defer adapter.Close() //nolint:errcheck

		log.Printf("adapter: connecting to %s", *wsURL)
		if err := adapter.Connect(ctx); err != nil {
			log.Fatalf("adapter: connect error: %v", err)
		}

		for record := range adapter.Records() {
			log.Printf("adapter: source=%s ts=%s median_price=%.4f",
				record.Source,
				record.Timestamp.Format(time.RFC3339),
				record.Fields["median_price"],
			)
		}
		return
	}

	book := NewOrderBook()
	client := NewClient(*wsURL, book)

	log.Printf("connecting to %s", *wsURL)
	if err := client.Run(ctx); err != nil {
		log.Fatalf("client error: %v", err)
	}
}
