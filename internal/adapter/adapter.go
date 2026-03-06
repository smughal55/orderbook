package adapter

import (
	"context"

	"github.com/shazmughal/orderbook/internal/model"
)

// Adapter is the interface every data source adapter must implement.
type Adapter interface {
	// Name returns a unique identifier for this source.
	Name() string
	// Connect establishes the connection to the data source.
	Connect(ctx context.Context) error
	// Read blocks until the next record is available or ctx is cancelled.
	Read(ctx context.Context) (*model.DataRecord, error)
	// Close tears down the connection.
	Close() error
}
