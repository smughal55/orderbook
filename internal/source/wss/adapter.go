package wss

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shazmughal/orderbook/internal/model"
)

const recordsBufSize = 1024

// backoffDelays is the sequence of base delays (seconds) for successive
// reconnect attempts. All attempts beyond index 5 reuse 30 s.
var backoffDelays = []time.Duration{1, 2, 4, 8, 16, 30}

// defaultBackoff returns the base delay for attempt n (0-indexed) plus ±10% jitter.
func defaultBackoff(attempt int) time.Duration {
	idx := attempt
	if idx >= len(backoffDelays) {
		idx = len(backoffDelays) - 1
	}
	base := backoffDelays[idx] * time.Second
	jitter := time.Duration(float64(base) * 0.1 * (2*rand.Float64() - 1))
	return base + jitter
}

// WSSAdapter connects to a WebSocket feed, maintains an internal OrderBook,
// and emits normalised DataRecords on a buffered channel.
// It automatically reconnects with exponential backoff on unexpected disconnection.
//
// Lifecycle:
//
//	a := NewWSSAdapter("wss://example.com/feed")
//	if err := a.Connect(ctx); err != nil { ... }
//	for r := range a.Records() { ... }
type WSSAdapter struct {
	url  string
	book *OrderBook

	records chan model.DataRecord

	mu   sync.Mutex
	conn *websocket.Conn

	closeOnce sync.Once
	done      chan struct{}

	// backoff controls the delay before each reconnect attempt.
	// Defaults to defaultBackoff; override in tests to speed up reconnect cycles.
	backoff func(attempt int) time.Duration
}

// NewWSSAdapter creates a WSSAdapter for the given WebSocket URL.
// The URL scheme must be ws:// or wss://; validation occurs on Connect.
func NewWSSAdapter(rawURL string) *WSSAdapter {
	return &WSSAdapter{
		url:     rawURL,
		book:    NewOrderBook(),
		records: make(chan model.DataRecord, recordsBufSize),
		done:    make(chan struct{}),
		backoff: defaultBackoff,
	}
}

// Name returns the stable adapter identifier.
func (a *WSSAdapter) Name() string { return "wss" }

// Connect validates the URL scheme, performs the initial dial, and starts
// the reconnect loop in a background goroutine.
//
// Returns immediately with an error if the URL scheme is not ws:// or wss://,
// or if the initial dial fails.
func (a *WSSAdapter) Connect(ctx context.Context) error {
	u, err := url.Parse(a.url)
	if err != nil {
		return fmt.Errorf("wss adapter: invalid URL: %w", err)
	}
	if u.Scheme != "ws" && u.Scheme != "wss" {
		return fmt.Errorf("wss adapter: URL scheme must be ws:// or wss://, got %q", u.Scheme)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, a.url, nil)
	if err != nil {
		return fmt.Errorf("wss adapter: dial: %w", err)
	}

	a.mu.Lock()
	a.conn = conn
	a.mu.Unlock()

	go a.runLoop(ctx, conn)
	return nil
}

// runLoop manages the full connection lifecycle: it reads from the current
// WebSocket connection, and on unexpected disconnection it reconnects with
// exponential backoff. It is the sole owner of a.records and closes it on exit.
func (a *WSSAdapter) runLoop(ctx context.Context, initialConn *websocket.Conn) {
	defer func() {
		// Ensure shutdown state is consistent regardless of who triggered exit.
		a.closeOnce.Do(func() {
			close(a.done)
			a.mu.Lock()
			if a.conn != nil {
				a.conn.Close()
			}
			a.mu.Unlock()
		})
		// records is always closed here — the only place — preventing send-on-closed panics.
		close(a.records)
	}()

	// Bridge ctx cancellation → Close so any blocking ReadMessage unblocks.
	go func() {
		select {
		case <-ctx.Done():
			a.Close()
		case <-a.done:
		}
	}()

	conn := initialConn

	for {
		// Consume messages until the connection drops.
		a.readMessages(conn)

		// If shutdown was requested (Close() or ctx cancel), exit cleanly.
		select {
		case <-a.done:
			return
		default:
		}

		// Genuine server-side disconnect — reconnect with exponential backoff.
		attempt := 0
		for {
			delay := a.backoff(attempt)
			slog.Info("wss adapter: reconnecting",
				"url", a.url,
				"attempt", attempt+1,
				"delay", delay.Round(time.Millisecond))

			select {
			case <-time.After(delay):
			case <-a.done:
				return
			}

			newConn, _, dialErr := websocket.DefaultDialer.DialContext(ctx, a.url, nil)
			if dialErr != nil {
				// Dial failed — could be ctx cancelled or server still down.
				select {
				case <-a.done:
					return
				default:
					attempt++
					continue
				}
			}

			// Reconnected — update state and resume the message loop.
			slog.Info("wss adapter: reconnected", "url", a.url, "after_attempts", attempt+1)
			a.mu.Lock()
			a.conn = newConn
			a.mu.Unlock()
			conn = newConn
			a.book = NewOrderBook() // reset: a fresh snapshot is expected
			break
		}
	}
}

// readMessages reads WebSocket messages from conn, applies them to the internal
// order book, and emits DataRecords. Returns when conn produces an error.
func (a *WSSAdapter) readMessages(conn *websocket.Conn) {
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}

		msg, err := ParseMessage(data)
		if err != nil {
			continue
		}

		switch m := msg.(type) {
		case *SnapshotMessage:
			a.book.ApplySnapshot(m.Bids, m.Asks)
		case *UpdateMessage:
			a.book.ApplyUpdate(m.Side, m.Price, m.Amount)
		}

		a.emitRecord()
	}
}

// emitRecord builds a DataRecord from the current book state and sends it to
// the records channel. If the channel is full, the oldest record is dropped to
// make room for the new one (non-blocking).
//
// Records are only emitted when both sides of the book are populated.
func (a *WSSAdapter) emitRecord() {
	bestBid, okBid := a.book.BestBid()
	bestAsk, okAsk := a.book.BestAsk()
	median, okMedian := a.book.MedianPrice()

	if !okBid || !okAsk || !okMedian {
		return
	}

	record := model.DataRecord{
		Source:    a.Name(),
		Timestamp: time.Now().UTC(),
		Fields: map[string]float64{
			"best_bid":     bestBid,
			"best_ask":     bestAsk,
			"median_price": median,
		},
		Meta: map[string]string{
			"source_url": a.url,
		},
	}

	select {
	case a.records <- record:
	default:
		// Channel full: drop oldest entry to make room.
		select {
		case <-a.records:
		default:
		}
		select {
		case a.records <- record:
		default:
		}
	}
}

// Records returns the channel on which DataRecords are emitted.
// The channel remains open across reconnections and is closed only when the
// adapter fully shuts down. Callers must not close it.
func (a *WSSAdapter) Records() <-chan model.DataRecord {
	return a.records
}

// Close shuts down the adapter: signals the reconnect loop to stop, closes
// the current WebSocket connection, and causes the records channel to be closed.
// Idempotent; subsequent calls are no-ops.
func (a *WSSAdapter) Close() error {
	a.closeOnce.Do(func() {
		close(a.done)
		a.mu.Lock()
		if a.conn != nil {
			a.conn.Close()
		}
		a.mu.Unlock()
	})
	return nil
}
