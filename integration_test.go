package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIntegrationClientWithMockServer(t *testing.T) {
	// Spin up the mock server's handler on an httptest server.
	ms := NewMockServer(":0") // address unused; httptest picks its own
	ts := httptest.NewServer(ms.srv.Handler)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"

	book := NewOrderBook()
	client := NewClient(wsURL, book)

	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
	defer cancel()

	go client.Run(ctx)

	// Wait enough time for the snapshot + several updates to arrive.
	time.Sleep(300 * time.Millisecond)

	bid, okBid := book.BestBid()
	ask, okAsk := book.BestAsk()
	median, okMed := book.MedianPrice()

	if !okBid {
		t.Fatal("expected a best bid after receiving snapshot and updates")
	}
	if !okAsk {
		t.Fatal("expected a best ask after receiving snapshot and updates")
	}
	if !okMed {
		t.Fatal("expected a median price")
	}

	if bid <= 0 || ask <= 0 {
		t.Fatalf("bid and ask should be positive, got bid=%v ask=%v", bid, ask)
	}
	if bid >= ask {
		t.Fatalf("best bid should be less than best ask, got bid=%v ask=%v", bid, ask)
	}
	if median != (bid+ask)/2 {
		t.Fatalf("median should be (bid+ask)/2 = %v, got %v", (bid+ask)/2, median)
	}

	t.Logf("bid=%.4f ask=%.4f median=%.4f", bid, ask, median)
}

func TestIntegrationSnapshotThenUpdates(t *testing.T) {
	// A deterministic integration test: hand-craft messages via a custom handler.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()

		messages := []string{
			`{"type":"snapshot","bids":[{"price":100,"amount":5},{"price":99,"amount":3}],"asks":[{"price":101,"amount":4},{"price":102,"amount":7}]}`,
			`{"type":"update","side":"bid","price":100.5,"amount":2}`,
			`{"type":"update","side":"ask","price":100.8,"amount":1}`,
			`{"type":"update","side":"bid","price":100.5,"amount":0}`, // remove 100.5 bid
		}

		for _, msg := range messages {
			if err := conn.WriteMessage(1, []byte(msg)); err != nil {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}

		// Keep connection open until client disconnects.
		<-r.Context().Done()
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	book := NewOrderBook()
	client := NewClient(wsURL, book)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go client.Run(ctx)

	// Wait for all messages to be processed.
	time.Sleep(200 * time.Millisecond)

	bid, _ := book.BestBid()
	ask, _ := book.BestAsk()
	median, ok := book.MedianPrice()

	// After all updates: best bid=100 (100.5 was removed), best ask=100.8
	if bid != 100 {
		t.Fatalf("expected best bid 100, got %v", bid)
	}
	if ask != 100.8 {
		t.Fatalf("expected best ask 100.8, got %v", ask)
	}
	expectedMedian := (100 + 100.8) / 2
	if !ok || median != expectedMedian {
		t.Fatalf("expected median %v, got %v (ok=%v)", expectedMedian, median, ok)
	}

	t.Logf("bid=%.2f ask=%.2f median=%.4f", bid, ask, median)
}
