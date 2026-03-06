package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shazmughal/orderbook/internal/model"
)

const dedupTTL = 5 * time.Minute

// DedupAdapter wraps another adapter and skips records already seen (via Redis).
type DedupAdapter struct {
	inner Adapter
	rdb   *redis.Client
}

func NewDedupAdapter(inner Adapter, rdb *redis.Client) *DedupAdapter {
	return &DedupAdapter{inner: inner, rdb: rdb}
}

func (d *DedupAdapter) Name() string { return d.inner.Name() }

func (d *DedupAdapter) Connect(ctx context.Context) error {
	return d.inner.Connect(ctx)
}

func (d *DedupAdapter) Read(ctx context.Context) (*model.DataRecord, error) {
	for {
		rec, err := d.inner.Read(ctx)
		if err != nil {
			return nil, err
		}

		key := fmt.Sprintf("dedup:%s", rec.RecordID)
		ok, err := d.rdb.SetNX(ctx, key, 1, dedupTTL).Result()
		if err != nil {
			return nil, fmt.Errorf("redis dedup check: %w", err)
		}
		if ok {
			return rec, nil
		}
		// duplicate — skip and read next
	}
}

func (d *DedupAdapter) Close() error {
	return d.inner.Close()
}
