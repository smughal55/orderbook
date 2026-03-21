package source

import (
	"context"

	"github.com/shazmughal/orderbook/internal/model"
)

// SourceAdapter is the interface that all data source adapters must implement.
// An adapter is responsible for connecting to an external data feed, consuming
// its messages, and emitting normalised DataRecords for downstream processing.
//
// Lifecycle:
//
//	adapter := NewXXXAdapter(...)
//	if err := adapter.Connect(ctx); err != nil { ... }
//	for record := range adapter.Records() { ... }
//	adapter.Close()
type SourceAdapter interface {
	// Name returns a stable, unique identifier for this adapter (e.g. "wss-orderbook").
	// The name is used as the source field on emitted DataRecords and as part of
	// NATS subject names (records.<name>), so it must not contain spaces or dots.
	Name() string

	// Connect establishes the connection to the external data source and starts
	// the internal read loop that populates the Records channel.
	//
	// Connect must be called exactly once before Records() is consumed.
	// Calling Connect a second time on an already-connected adapter returns an
	// error without establishing a second connection.
	//
	// Connect returns an error if the initial connection attempt fails.
	// Implementations that support reconnection (e.g. WSSAdapter) manage
	// subsequent reconnection internally without returning from Connect.
	//
	// Connect returns when the adapter is shut down (ctx cancelled or Close called).
	Connect(ctx context.Context) error

	// Records returns the channel on which the adapter emits DataRecords.
	// The channel is created at construction time and remains valid for the
	// lifetime of the adapter.
	//
	// After Close() is called and the internal read loop has stopped, the channel
	// will be closed by the implementation, causing any range loop to exit cleanly.
	//
	// Callers must not close this channel themselves.
	Records() <-chan model.DataRecord

	// Close shuts down the adapter, terminates the connection to the data source,
	// and closes the Records channel. It is safe to call Close more than once;
	// subsequent calls are no-ops and return nil.
	Close() error
}
