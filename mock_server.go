package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// MockServer simulates a WebSocket feed that sends an initial snapshot
// followed by order updates every 20ms.
type MockServer struct {
	srv *http.Server
}

// NewMockServer creates a mock server on the given address (e.g. ":8080").
func NewMockServer(addr string) *MockServer {
	mux := http.NewServeMux()
	ms := &MockServer{
		srv: &http.Server{Addr: addr, Handler: mux},
	}
	mux.HandleFunc("/ws", ms.handleWS)
	return ms
}

// Start begins listening. Blocks until the server shuts down.
func (ms *MockServer) Start() error {
	log.Printf("mock server listening on %s", ms.srv.Addr)
	return ms.srv.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (ms *MockServer) Shutdown(ctx context.Context) error {
	return ms.srv.Shutdown(ctx)
}

func (ms *MockServer) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}
	defer conn.Close()

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	midPrice := 100.0

	snapshot := buildSnapshot(midPrice, rng)
	data, _ := json.Marshal(snapshot)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return
	}

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			update := generateUpdate(midPrice, rng)
			data, _ := json.Marshal(update)
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		}
	}
}

func buildSnapshot(mid float64, rng *rand.Rand) SnapshotMessage {
	bids := make([]PriceLevel, 10)
	asks := make([]PriceLevel, 10)

	for i := range 10 {
		spread := float64(i+1) * 0.10
		bids[i] = PriceLevel{
			Price:  round2(mid - spread),
			Amount: round2(1 + rng.Float64()*20),
		}
		asks[i] = PriceLevel{
			Price:  round2(mid + spread),
			Amount: round2(1 + rng.Float64()*20),
		}
	}

	return SnapshotMessage{Type: "snapshot", Bids: bids, Asks: asks}
}

func generateUpdate(mid float64, rng *rand.Rand) UpdateMessage {
	side := "bid"
	price := mid - rng.Float64()*1.0
	if rng.Intn(2) == 0 {
		side = "ask"
		price = mid + rng.Float64()*1.0
	}

	amount := 0.0
	if rng.Intn(5) > 0 { // 80% insert/update, 20% remove
		amount = round2(1 + rng.Float64()*15)
	}

	return UpdateMessage{
		Type:   "update",
		Side:   side,
		Price:  round2(price),
		Amount: amount,
	}
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
