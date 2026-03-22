package bus

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// Publisher is the interface for sending messages to a named subject.
type Publisher interface {
	// Publish serialises data and delivers it to the given subject.
	// The call blocks until a JetStream server acknowledgement is received or
	// the context / a 5-second internal timeout expires.
	Publish(ctx context.Context, subject string, data []byte) error

	// Close drains any in-flight publishes and closes the NATS connection.
	Close() error
}

// NATSPublisher implements Publisher using NATS JetStream async publish.
type NATSPublisher struct {
	nc      *nats.Conn
	js      nats.JetStreamContext
	mu      sync.Mutex
	streams map[string]struct{} // stream names already ensured to exist
}

// NewNATSPublisher connects to NATS at url, creates a JetStream context, and
// returns a ready Publisher. Additional NATS options (e.g. TLS) are forwarded.
// Returns an error if NATS is unreachable at startup.
func NewNATSPublisher(url string, opts ...nats.Option) (*NATSPublisher, error) {
	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("nats publisher: connect: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("nats publisher: jetstream: %w", err)
	}

	return &NATSPublisher{
		nc:      nc,
		js:      js,
		streams: make(map[string]struct{}),
	}, nil
}

// Publish delivers data to subject using JetStream async publish and waits up
// to 5 seconds for the server acknowledgement. The required JetStream stream is
// auto-created on first use if it does not already exist.
func (p *NATSPublisher) Publish(ctx context.Context, subject string, data []byte) error {
	if err := p.ensureStream(subject); err != nil {
		return err
	}

	future, err := p.js.PublishAsync(subject, data)
	if err != nil {
		return fmt.Errorf("publish to %q: %w", subject, err)
	}

	select {
	case <-future.Ok():
		return nil
	case pubErr := <-future.Err():
		return fmt.Errorf("publish ack for %q: %w", subject, pubErr)
	case <-time.After(5 * time.Second):
		return fmt.Errorf("publish ack timeout for subject %q", subject)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ensureStream guarantees the JetStream stream for subject's family exists.
// The stream name is the first dotted segment of the subject ("records" from
// "records.source1"). Results are cached so only the first call hits NATS.
func (p *NATSPublisher) ensureStream(subject string) error {
	streamName := streamNameFromSubject(subject)

	p.mu.Lock()
	_, seen := p.streams[streamName]
	p.mu.Unlock()

	if seen {
		return nil
	}

	_, err := p.js.StreamInfo(streamName)
	if err != nil {
		if !errors.Is(err, nats.ErrStreamNotFound) {
			return fmt.Errorf("bus: check stream %q: %w", streamName, err)
		}
		// Stream does not exist — create it.
		_, err = p.js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{streamName + ".>"},
		})
		if err != nil {
			return fmt.Errorf("bus: create stream %q: %w", streamName, err)
		}
	}

	p.mu.Lock()
	p.streams[streamName] = struct{}{}
	p.mu.Unlock()
	return nil
}

// streamNameFromSubject derives a NATS stream name from the leading subject
// segment, e.g. "records" from "records.source1".
func streamNameFromSubject(subject string) string {
	if i := strings.IndexByte(subject, '.'); i > 0 {
		return subject[:i]
	}
	return subject
}

// Close drains any pending publishes and closes the NATS connection.
func (p *NATSPublisher) Close() error {
	return p.nc.Drain()
}
