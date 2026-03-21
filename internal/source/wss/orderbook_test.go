package wss

import (
	"testing"
)

func TestEmptyBook(t *testing.T) {
	ob := NewOrderBook()

	if _, ok := ob.BestBid(); ok {
		t.Fatal("expected no best bid on empty book")
	}
	if _, ok := ob.BestAsk(); ok {
		t.Fatal("expected no best ask on empty book")
	}
	if _, ok := ob.MedianPrice(); ok {
		t.Fatal("expected no median on empty book")
	}
}

func TestApplySnapshot(t *testing.T) {
	ob := NewOrderBook()

	ob.ApplySnapshot(
		[]PriceLevel{{Price: 100, Amount: 5}, {Price: 99, Amount: 3}, {Price: 98, Amount: 10}},
		[]PriceLevel{{Price: 101, Amount: 4}, {Price: 102, Amount: 7}, {Price: 103, Amount: 2}},
	)

	bid, ok := ob.BestBid()
	if !ok || bid != 100 {
		t.Fatalf("expected best bid 100, got %v (ok=%v)", bid, ok)
	}

	ask, ok := ob.BestAsk()
	if !ok || ask != 101 {
		t.Fatalf("expected best ask 101, got %v (ok=%v)", ask, ok)
	}

	median, ok := ob.MedianPrice()
	if !ok || median != 100.5 {
		t.Fatalf("expected median 100.5, got %v (ok=%v)", median, ok)
	}
}

func TestSnapshotReplacesExisting(t *testing.T) {
	ob := NewOrderBook()

	ob.ApplySnapshot(
		[]PriceLevel{{Price: 50, Amount: 1}},
		[]PriceLevel{{Price: 60, Amount: 1}},
	)

	ob.ApplySnapshot(
		[]PriceLevel{{Price: 200, Amount: 1}},
		[]PriceLevel{{Price: 210, Amount: 1}},
	)

	bid, _ := ob.BestBid()
	if bid != 200 {
		t.Fatalf("expected best bid 200 after second snapshot, got %v", bid)
	}
	ask, _ := ob.BestAsk()
	if ask != 210 {
		t.Fatalf("expected best ask 210 after second snapshot, got %v", ask)
	}
}

func TestApplyUpdateInsert(t *testing.T) {
	ob := NewOrderBook()

	ob.ApplySnapshot(
		[]PriceLevel{{Price: 100, Amount: 5}},
		[]PriceLevel{{Price: 101, Amount: 4}},
	)

	ob.ApplyUpdate("bid", 100.5, 2)

	bid, _ := ob.BestBid()
	if bid != 100.5 {
		t.Fatalf("expected best bid 100.5, got %v", bid)
	}
}

func TestApplyUpdateModify(t *testing.T) {
	ob := NewOrderBook()

	ob.ApplySnapshot(
		[]PriceLevel{{Price: 100, Amount: 5}},
		[]PriceLevel{{Price: 101, Amount: 4}},
	)

	ob.ApplyUpdate("bid", 100, 10)

	ob.mu.RLock()
	amount := ob.bids.levels[100]
	ob.mu.RUnlock()

	if amount != 10 {
		t.Fatalf("expected bid amount at 100 to be 10, got %v", amount)
	}

	bid, _ := ob.BestBid()
	if bid != 100 {
		t.Fatalf("best bid should remain 100, got %v", bid)
	}
}

func TestApplyUpdateRemove(t *testing.T) {
	ob := NewOrderBook()

	ob.ApplySnapshot(
		[]PriceLevel{{Price: 99, Amount: 3}, {Price: 100, Amount: 5}},
		[]PriceLevel{{Price: 101, Amount: 4}},
	)

	ob.ApplyUpdate("bid", 100, 0) // remove best bid

	bid, ok := ob.BestBid()
	if !ok || bid != 99 {
		t.Fatalf("expected best bid 99 after removal, got %v (ok=%v)", bid, ok)
	}
}

func TestApplyUpdateRemoveNonexistent(t *testing.T) {
	ob := NewOrderBook()

	ob.ApplySnapshot(
		[]PriceLevel{{Price: 100, Amount: 5}},
		[]PriceLevel{{Price: 101, Amount: 4}},
	)

	ob.ApplyUpdate("bid", 999, 0) // remove price that doesn't exist

	bid, _ := ob.BestBid()
	if bid != 100 {
		t.Fatalf("expected best bid 100 unchanged, got %v", bid)
	}
}

func TestApplyUpdateAskSide(t *testing.T) {
	ob := NewOrderBook()

	ob.ApplySnapshot(
		[]PriceLevel{{Price: 100, Amount: 5}},
		[]PriceLevel{{Price: 102, Amount: 4}},
	)

	ob.ApplyUpdate("ask", 101, 3)

	ask, _ := ob.BestAsk()
	if ask != 101 {
		t.Fatalf("expected best ask 101, got %v", ask)
	}
}

func TestMedianAfterRemovingOneSide(t *testing.T) {
	ob := NewOrderBook()

	ob.ApplySnapshot(
		[]PriceLevel{{Price: 100, Amount: 5}},
		[]PriceLevel{{Price: 101, Amount: 4}},
	)

	ob.ApplyUpdate("bid", 100, 0) // empty bids

	if _, ok := ob.MedianPrice(); ok {
		t.Fatal("expected no median when bids are empty")
	}
}

func TestParseSnapshotMessage(t *testing.T) {
	raw := `{"type":"snapshot","bids":[{"price":100,"amount":5}],"asks":[{"price":101,"amount":4}]}`
	msg, err := ParseMessage([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	snap, ok := msg.(*SnapshotMessage)
	if !ok {
		t.Fatalf("expected *SnapshotMessage, got %T", msg)
	}
	if len(snap.Bids) != 1 || snap.Bids[0].Price != 100 {
		t.Fatalf("unexpected bids: %+v", snap.Bids)
	}
}

func TestParseUpdateMessage(t *testing.T) {
	raw := `{"type":"update","side":"ask","price":101.5,"amount":3}`
	msg, err := ParseMessage([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	upd, ok := msg.(*UpdateMessage)
	if !ok {
		t.Fatalf("expected *UpdateMessage, got %T", msg)
	}
	if upd.Side != "ask" || upd.Price != 101.5 || upd.Amount != 3 {
		t.Fatalf("unexpected update: %+v", upd)
	}
}

func TestParseUnknownType(t *testing.T) {
	raw := `{"type":"unknown"}`
	_, err := ParseMessage([]byte(raw))
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestParseMalformedJSON(t *testing.T) {
	_, err := ParseMessage([]byte(`not json at all`))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestParseEmptyType(t *testing.T) {
	// An empty type string must be rejected the same as any unknown type.
	_, err := ParseMessage([]byte(`{"type":""}`))
	if err == nil {
		t.Fatal("expected error for empty type field")
	}
}

func TestApplySnapshotEmptySlices(t *testing.T) {
	ob := NewOrderBook()
	ob.ApplySnapshot(nil, nil)

	if _, ok := ob.BestBid(); ok {
		t.Fatal("expected no best bid after empty snapshot")
	}
	if _, ok := ob.BestAsk(); ok {
		t.Fatal("expected no best ask after empty snapshot")
	}
}

// TestOrderSideSortInvariant verifies that the internal prices slice stays
// sorted ascending regardless of insertion order.
func TestOrderSideSortInvariant(t *testing.T) {
	ob := NewOrderBook()

	// Insert bids in deliberately descending order.
	ob.ApplySnapshot(
		[]PriceLevel{
			{Price: 105, Amount: 1},
			{Price: 101, Amount: 1},
			{Price: 103, Amount: 1},
			{Price: 102, Amount: 1},
			{Price: 104, Amount: 1},
		},
		[]PriceLevel{{Price: 110, Amount: 1}},
	)

	ob.mu.RLock()
	prices := make([]float64, len(ob.bids.prices))
	copy(prices, ob.bids.prices)
	ob.mu.RUnlock()

	for i := 1; i < len(prices); i++ {
		if prices[i] < prices[i-1] {
			t.Fatalf("prices slice not sorted at index %d: %v >= %v", i, prices[i-1], prices[i])
		}
	}

	// BestBid must be the maximum (last element).
	bid, ok := ob.BestBid()
	if !ok || bid != 105 {
		t.Fatalf("expected best bid 105, got %v (ok=%v)", bid, ok)
	}
}
