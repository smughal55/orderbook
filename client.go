package main

import (
	"context"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// Client connects to a WebSocket feed, maintains an OrderBook,
// and prints the median price every 200ms.
type Client struct {
	url  string
	book *OrderBook
}

func NewClient(url string, book *OrderBook) *Client {
	return &Client{url: url, book: book}
}

// Run connects to the WebSocket and processes messages until ctx is cancelled.
// It also starts a 200ms ticker that logs the median price.
func (c *Client) Run(ctx context.Context) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, c.url, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Close the connection when context is cancelled so ReadMessage unblocks.
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	done := make(chan struct{})

	go c.medianTicker(ctx, done)

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			select {
			case <-ctx.Done():
				close(done)
				return nil
			default:
				close(done)
				return err
			}
		}

		msg, err := ParseMessage(data)
		if err != nil {
			log.Printf("parse error: %v", err)
			continue
		}

		switch m := msg.(type) {
		case *SnapshotMessage:
			c.book.ApplySnapshot(m.Bids, m.Asks)
			log.Printf("snapshot applied: %d bids, %d asks", len(m.Bids), len(m.Asks))
		case *UpdateMessage:
			c.book.ApplyUpdate(m.Side, m.Price, m.Amount)
		}
	}
}

func (c *Client) medianTicker(ctx context.Context, done <-chan struct{}) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			if median, ok := c.book.MedianPrice(); ok {
				bid, _ := c.book.BestBid()
				ask, _ := c.book.BestAsk()
				log.Printf("median=%.4f  best_bid=%.2f  best_ask=%.2f", median, bid, ask)
			} else {
				log.Print("median: N/A (book not ready)")
			}
		}
	}
}
