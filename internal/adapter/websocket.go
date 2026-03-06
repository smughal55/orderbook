package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shazmughal/orderbook/internal/model"
)

// WebSocketAdapter connects to a WebSocket feed and emits DataRecords.
type WebSocketAdapter struct {
	name string
	url  string
	conn *websocket.Conn
	seq  int64
}

func NewWebSocketAdapter(name, url string) *WebSocketAdapter {
	return &WebSocketAdapter{name: name, url: url}
}

func (a *WebSocketAdapter) Name() string { return a.name }

func (a *WebSocketAdapter) Connect(ctx context.Context) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, a.url, nil)
	if err != nil {
		return fmt.Errorf("websocket connect %s: %w", a.url, err)
	}
	a.conn = conn

	go func() {
		<-ctx.Done()
		a.conn.Close()
	}()

	return nil
}

func (a *WebSocketAdapter) Read(ctx context.Context) (*model.DataRecord, error) {
	_, data, err := a.conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("websocket read: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("json parse: %w", err)
	}

	a.seq++
	recordID := fmt.Sprintf("%s-%d-%d", a.name, time.Now().UnixNano(), a.seq)

	record := model.NewDataRecord(a.name, recordID, raw)
	return &record, nil
}

func (a *WebSocketAdapter) Close() error {
	if a.conn != nil {
		return a.conn.Close()
	}
	return nil
}
