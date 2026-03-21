package wss

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shazmughal/orderbook/internal/model"
)

var testUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// wsTestServer starts an httptest HTTP server that upgrades connections to
// WebSocket and calls handler for each connection.
// Returns the server and its ws:// URL. The server is closed via t.Cleanup.
func wsTestServer(t *testing.T, handler func(*websocket.Conn)) (srv *httptest.Server, wsURL string) {
	t.Helper()
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		defer conn.Close()
		handler(conn)
	}))
	t.Cleanup(srv.Close)
	wsURL = "ws" + strings.TrimPrefix(srv.URL, "http")
	return
}

func sendJSON(t *testing.T, conn *websocket.Conn, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Logf("write: %v", err) // may fail if client disconnected; not fatal
	}
}

func waitForClosed(t *testing.T, a *WSSAdapter, timeout time.Duration) {
	t.Helper()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case _, open := <-a.Records():
			if !open {
				return
			}
		case <-timer.C:
			t.Fatal("timed out waiting for records channel to close")
		}
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestConnectValidURL(t *testing.T) {
	_, wsURL := wsTestServer(t, func(conn *websocket.Conn) {
		// Server does nothing; just stays open long enough for the dial.
		time.Sleep(100 * time.Millisecond)
	})

	a := NewWSSAdapter(wsURL)
	defer a.Close()

	if err := a.Connect(context.Background()); err != nil {
		t.Fatalf("expected Connect to succeed, got: %v", err)
	}
}

func TestConnectInvalidScheme_HTTP(t *testing.T) {
	a := NewWSSAdapter("http://localhost:9999/ws")
	err := a.Connect(context.Background())
	if err == nil {
		t.Fatal("expected error for http:// scheme")
	}
	if !strings.Contains(err.Error(), "scheme") {
		t.Fatalf("expected scheme error, got: %v", err)
	}
}

func TestConnectInvalidScheme_HTTPS(t *testing.T) {
	a := NewWSSAdapter("https://localhost:9999/ws")
	err := a.Connect(context.Background())
	if err == nil {
		t.Fatal("expected error for https:// scheme")
	}
	if !strings.Contains(err.Error(), "scheme") {
		t.Fatalf("expected scheme error, got: %v", err)
	}
}

func TestRecordsAfterSnapshotAndUpdate(t *testing.T) {
	snap := SnapshotMessage{
		Type: "snapshot",
		Bids: []PriceLevel{{Price: 100, Amount: 5}},
		Asks: []PriceLevel{{Price: 101, Amount: 4}},
	}
	upd := UpdateMessage{
		Type:   "update",
		Side:   "bid",
		Price:  100.5,
		Amount: 2,
	}

	_, wsURL := wsTestServer(t, func(conn *websocket.Conn) {
		sendJSON(t, conn, snap)
		sendJSON(t, conn, upd)
		// Hold the connection open so the adapter can read.
		time.Sleep(2 * time.Second)
	})

	a := NewWSSAdapter(wsURL)
	defer a.Close()

	if err := a.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	select {
	case r, open := <-a.Records():
		if !open {
			t.Fatal("records channel closed prematurely")
		}
		if r.Source != "wss" {
			t.Errorf("expected source=wss, got %q", r.Source)
		}
		if r.Meta["source_url"] != wsURL {
			t.Errorf("expected source_url=%q, got %q", wsURL, r.Meta["source_url"])
		}
		if r.Fields["best_bid"] == 0 {
			t.Errorf("expected best_bid to be set, got 0")
		}
		if r.Fields["best_ask"] == 0 {
			t.Errorf("expected best_ask to be set, got 0")
		}
		if r.Fields["median_price"] == 0 {
			t.Errorf("expected median_price to be set, got 0")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a DataRecord")
	}
}

func TestSnapshotFieldValues(t *testing.T) {
	snap := SnapshotMessage{
		Type: "snapshot",
		Bids: []PriceLevel{{Price: 200, Amount: 10}},
		Asks: []PriceLevel{{Price: 202, Amount: 8}},
	}

	_, wsURL := wsTestServer(t, func(conn *websocket.Conn) {
		sendJSON(t, conn, snap)
		time.Sleep(2 * time.Second)
	})

	a := NewWSSAdapter(wsURL)
	defer a.Close()

	if err := a.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	select {
	case r := <-a.Records():
		if r.Fields["best_bid"] != 200 {
			t.Errorf("best_bid: want 200, got %v", r.Fields["best_bid"])
		}
		if r.Fields["best_ask"] != 202 {
			t.Errorf("best_ask: want 202, got %v", r.Fields["best_ask"])
		}
		if r.Fields["median_price"] != 201 {
			t.Errorf("median_price: want 201, got %v", r.Fields["median_price"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for DataRecord")
	}
}

func TestNoRecordBeforeSnapshot(t *testing.T) {
	upd := UpdateMessage{
		Type:   "update",
		Side:   "bid",
		Price:  50,
		Amount: 1,
	}

	_, wsURL := wsTestServer(t, func(conn *websocket.Conn) {
		// Send an update only — no prior snapshot, so asks side is empty.
		sendJSON(t, conn, upd)
		time.Sleep(300 * time.Millisecond)
	})

	a := NewWSSAdapter(wsURL)
	defer a.Close()

	if err := a.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	select {
	case r, open := <-a.Records():
		if open {
			t.Fatalf("expected no record before both sides populated, got: %+v", r)
		}
		// Channel closed after server dropped connection — acceptable.
	case <-time.After(500 * time.Millisecond):
		// No record received: correct behaviour.
	}
}

func TestRecordsChannelClosedAfterClose(t *testing.T) {
	_, wsURL := wsTestServer(t, func(conn *websocket.Conn) {
		snap := SnapshotMessage{
			Type: "snapshot",
			Bids: []PriceLevel{{Price: 100, Amount: 5}},
			Asks: []PriceLevel{{Price: 101, Amount: 4}},
		}
		sendJSON(t, conn, snap)
		time.Sleep(5 * time.Second) // keep open until adapter closes
	})

	a := NewWSSAdapter(wsURL)
	if err := a.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Consume one record to confirm the adapter is live.
	select {
	case <-a.Records():
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first record")
	}

	a.Close()
	waitForClosed(t, a, 2*time.Second)
}

func TestContextCancellationStopsAdapter(t *testing.T) {
	_, wsURL := wsTestServer(t, func(conn *websocket.Conn) {
		snap := SnapshotMessage{
			Type: "snapshot",
			Bids: []PriceLevel{{Price: 100, Amount: 5}},
			Asks: []PriceLevel{{Price: 101, Amount: 4}},
		}
		sendJSON(t, conn, snap)
		time.Sleep(5 * time.Second)
	})

	ctx, cancel := context.WithCancel(context.Background())
	a := NewWSSAdapter(wsURL)
	if err := a.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Wait for at least one record to confirm the adapter is live.
	select {
	case <-a.Records():
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first record")
	}

	cancel()
	waitForClosed(t, a, 2*time.Second)
}

func TestCloseIsIdempotent(t *testing.T) {
	_, wsURL := wsTestServer(t, func(conn *websocket.Conn) {
		time.Sleep(2 * time.Second)
	})

	a := NewWSSAdapter(wsURL)
	if err := a.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Multiple Close calls must not panic or return an error.
	for i := range 5 {
		if err := a.Close(); err != nil {
			t.Errorf("Close() call %d returned error: %v", i+1, err)
		}
	}
}

// ---------------------------------------------------------------------------
// Additional coverage for Step 7 logic
// ---------------------------------------------------------------------------

// TestAdapterName verifies the stable identifier used in DataRecord.Source
// and NATS subject names.
func TestAdapterName(t *testing.T) {
	a := NewWSSAdapter("ws://localhost:9999")
	if got := a.Name(); got != "wss" {
		t.Errorf("Name(): want %q, got %q", "wss", got)
	}
}

// TestConnectDialFailure verifies that a connection refused error is surfaced
// by Connect rather than silently swallowed.
func TestConnectDialFailure(t *testing.T) {
	// Start a server, close it immediately, then attempt to connect to its
	// (now-closed) address — guarantees a connection-refused condition.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	srv.Close()

	a := NewWSSAdapter(closedURL)
	err := a.Connect(context.Background())
	if err == nil {
		t.Fatal("expected dial error for closed server, got nil")
	}
	if !strings.Contains(err.Error(), "dial") {
		t.Errorf("expected error to mention dial, got: %v", err)
	}
}

// TestRecordsBufferCapacity verifies the channel is created with the
// documented 1024-record buffer so slow consumers do not block the read loop.
func TestRecordsBufferCapacity(t *testing.T) {
	a := NewWSSAdapter("ws://localhost:9999")
	if got := cap(a.Records()); got != recordsBufSize {
		t.Errorf("Records() buffer: want %d, got %d", recordsBufSize, got)
	}
}

// TestInvalidMessageSkipped verifies that a malformed WebSocket message does
// not kill the read loop — subsequent valid messages must still be processed.
func TestInvalidMessageSkipped(t *testing.T) {
	snap := SnapshotMessage{
		Type: "snapshot",
		Bids: []PriceLevel{{Price: 300, Amount: 5}},
		Asks: []PriceLevel{{Price: 301, Amount: 4}},
	}

	_, wsURL := wsTestServer(t, func(conn *websocket.Conn) {
		// Send garbage first.
		if err := conn.WriteMessage(websocket.TextMessage, []byte("not valid json {{{")); err != nil {
			t.Logf("write garbage: %v", err)
			return
		}
		// Then send a valid snapshot — adapter must still process it.
		sendJSON(t, conn, snap)
		time.Sleep(2 * time.Second)
	})

	a := NewWSSAdapter(wsURL)
	defer a.Close()

	if err := a.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	select {
	case r, open := <-a.Records():
		if !open {
			t.Fatal("records channel closed prematurely after invalid message")
		}
		if r.Fields["best_bid"] != 300 {
			t.Errorf("best_bid: want 300, got %v", r.Fields["best_bid"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out — adapter may have stopped after invalid message")
	}
}

// ---------------------------------------------------------------------------
// Reconnection tests (Step 8)
// ---------------------------------------------------------------------------

// TestDefaultBackoffSequence verifies the full delay table defined by Step 8:
// 1 s, 2 s, 4 s, 8 s, 16 s, 30 s (capped). Each delay includes ±10% jitter,
// so every returned value must fall in [base*0.9, base*1.1).
// Running 100 iterations per step gives strong statistical confidence.
func TestDefaultBackoffSequence(t *testing.T) {
	cases := []struct {
		attempt int
		base    time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
		{5, 30 * time.Second},
		{6, 30 * time.Second},   // first attempt beyond the table — must use 30 s cap
		{100, 30 * time.Second}, // far beyond the table — still capped at 30 s
	}

	for _, tc := range cases {
		low := time.Duration(float64(tc.base) * 0.9)
		high := time.Duration(float64(tc.base) * 1.1)
		for i := 0; i < 100; i++ {
			d := defaultBackoff(tc.attempt)
			if d < low || d >= high {
				t.Errorf("attempt=%d run=%d: want delay in [%v, %v), got %v",
					tc.attempt, i+1, low, high, d)
				break // one failure per attempt case is enough
			}
		}
	}
}

// TestDefaultBackoffMonotonicity verifies that each successive step in the
// delay table produces a larger base delay than the previous one.
func TestDefaultBackoffMonotonicity(t *testing.T) {
	var prev time.Duration
	for attempt := 0; attempt < len(backoffDelays); attempt++ {
		// Use the raw table entry to compare bases without jitter noise.
		idx := attempt
		if idx >= len(backoffDelays) {
			idx = len(backoffDelays) - 1
		}
		base := backoffDelays[idx] * time.Second
		if attempt > 0 && base <= prev {
			t.Errorf("delay at attempt %d (%v) is not greater than attempt %d (%v)",
				attempt, base, attempt-1, prev)
		}
		prev = base
	}
}

// TestNewWSSAdapterDefaultBackoff verifies that NewWSSAdapter sets the
// backoff field to a non-nil function that returns a positive duration.
// A nil backoff would panic the first time runLoop tries to reconnect.
func TestNewWSSAdapterDefaultBackoff(t *testing.T) {
	a := NewWSSAdapter("ws://localhost:9999")
	if a.backoff == nil {
		t.Fatal("backoff field must not be nil after NewWSSAdapter")
	}
	d := a.backoff(0)
	if d <= 0 {
		t.Errorf("default backoff for attempt 0 must be positive, got %v", d)
	}
}

// TestReconnectAfterServerClose verifies that the adapter reconnects and
// continues emitting records after the server drops the connection.
// The first connection sends a snapshot with bid=100 then closes.
// The second connection sends a snapshot with bid=200 and stays open.
func TestReconnectAfterServerClose(t *testing.T) {
	var connCount atomic.Int32

	_, wsURL := wsTestServer(t, func(conn *websocket.Conn) {
		n := connCount.Add(1)
		snap := SnapshotMessage{
			Type: "snapshot",
			Bids: []PriceLevel{{Price: float64(n * 100), Amount: 1}},
			Asks: []PriceLevel{{Price: float64(n*100 + 1), Amount: 1}},
		}
		sendJSON(t, conn, snap)
		if n == 1 {
			return // drop first connection immediately after snapshot
		}
		time.Sleep(5 * time.Second) // keep second connection open
	})

	a := NewWSSAdapter(wsURL)
	a.backoff = func(int) time.Duration { return 50 * time.Millisecond }
	defer a.Close()

	if err := a.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Collect records until we see one from the second connection (bid=200).
	deadline := time.After(5 * time.Second)
	var sawSecond bool
	for !sawSecond {
		select {
		case r, open := <-a.Records():
			if !open {
				t.Fatal("records channel closed before second connection record arrived")
			}
			if r.Fields["best_bid"] == 200 {
				sawSecond = true
			}
		case <-deadline:
			t.Fatal("timed out waiting for record from reconnected connection")
		}
	}
}

// TestContextCancelDuringBackoff verifies that cancelling the context while
// the adapter is sleeping in a backoff delay causes it to stop promptly
// without hanging until the delay elapses.
func TestContextCancelDuringBackoff(t *testing.T) {
	// Server closes every connection immediately so the adapter always enters backoff.
	_, wsURL := wsTestServer(t, func(conn *websocket.Conn) {
		conn.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	a := NewWSSAdapter(wsURL)
	// Use a long backoff so the test would hang if cancel is not respected.
	a.backoff = func(int) time.Duration { return 2 * time.Second }
	if err := a.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Give the adapter time to enter the backoff sleep.
	time.Sleep(100 * time.Millisecond)
	cancel()

	// The adapter must stop well before the 2 s backoff would expire.
	waitForClosed(t, a, time.Second)
}

// TestReconnectResetsOrderbook verifies that the internal order book is reset
// on reconnection so that stale state from a previous connection cannot bleed
// into records emitted after reconnection.
//
// Sequence:
//  1. First connection: snapshot bid=100, ask=101  → record emitted
//  2. Connection dropped → adapter reconnects
//  3. Second connection: update bid=999 only (no snapshot)  → no record (asks empty)
//  4. Second connection: snapshot bid=500, ask=501  → record with bid=500 (not 100 or 999)
func TestReconnectResetsOrderbook(t *testing.T) {
	var connCount atomic.Int32

	_, wsURL := wsTestServer(t, func(conn *websocket.Conn) {
		n := connCount.Add(1)
		switch n {
		case 1:
			sendJSON(t, conn, SnapshotMessage{
				Type: "snapshot",
				Bids: []PriceLevel{{Price: 100, Amount: 1}},
				Asks: []PriceLevel{{Price: 101, Amount: 1}},
			})
			time.Sleep(100 * time.Millisecond) // let adapter read, then drop
		case 2:
			// Update only — no snapshot. Book is reset so asks side is empty;
			// this must NOT produce a record.
			sendJSON(t, conn, UpdateMessage{
				Type:   "update",
				Side:   "bid",
				Price:  999,
				Amount: 5,
			})
			// Now a full snapshot with fresh prices.
			sendJSON(t, conn, SnapshotMessage{
				Type: "snapshot",
				Bids: []PriceLevel{{Price: 500, Amount: 1}},
				Asks: []PriceLevel{{Price: 501, Amount: 1}},
			})
			time.Sleep(5 * time.Second)
		}
	})

	a := NewWSSAdapter(wsURL)
	a.backoff = func(int) time.Duration { return 50 * time.Millisecond }
	defer a.Close()

	if err := a.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Consume first record from the first connection.
	var firstRecord model.DataRecord
	select {
	case r, open := <-a.Records():
		if !open {
			t.Fatal("channel closed before first record")
		}
		firstRecord = r
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first record")
	}
	if firstRecord.Fields["best_bid"] != 100 {
		t.Fatalf("first record best_bid: want 100, got %v", firstRecord.Fields["best_bid"])
	}

	// Wait for a record from the second connection. It must come from the
	// fresh snapshot (bid=500), not from the stale update (bid=999).
	deadline := time.After(5 * time.Second)
	for {
		select {
		case r, open := <-a.Records():
			if !open {
				t.Fatal("channel closed before second-connection record")
			}
			if r.Fields["best_bid"] == 500 {
				// Correct — new snapshot propagated after book reset.
				return
			}
			if r.Fields["best_bid"] == 999 || r.Fields["best_bid"] == 100 {
				t.Fatalf("stale book state not reset on reconnect: best_bid=%v", r.Fields["best_bid"])
			}
		case <-deadline:
			t.Fatal("timed out waiting for post-reconnect record")
		}
	}
}
