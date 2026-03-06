package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/shazmughal/orderbook/internal/model"
)

// RESTAdapter polls a REST endpoint at a fixed interval and emits DataRecords.
type RESTAdapter struct {
	name     string
	url      string
	interval time.Duration
	client   *http.Client
	seq      int64
	ticker   *time.Ticker
}

func NewRESTAdapter(name, url string, interval time.Duration) *RESTAdapter {
	return &RESTAdapter{
		name:     name,
		url:      url,
		interval: interval,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *RESTAdapter) Name() string { return a.name }

func (a *RESTAdapter) Connect(_ context.Context) error {
	a.ticker = time.NewTicker(a.interval)
	return nil
}

func (a *RESTAdapter) Read(ctx context.Context) (*model.DataRecord, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-a.ticker.C:
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.url, nil)
	if err != nil {
		return nil, fmt.Errorf("rest request build: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rest fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("rest read body: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("rest json parse: %w", err)
	}

	a.seq++
	recordID := fmt.Sprintf("%s-%d-%d", a.name, time.Now().UnixNano(), a.seq)
	record := model.NewDataRecord(a.name, recordID, raw)
	return &record, nil
}

func (a *RESTAdapter) Close() error {
	if a.ticker != nil {
		a.ticker.Stop()
	}
	return nil
}
