package bus

import (
	"context"
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"
)

const defaultMaxInFlight = 100

// Subscriber is the interface for consuming messages from a subject.
type Subscriber interface {
	// Subscribe registers handler for messages on subject within the queue group.
	// handler returning non-nil causes the message to be Nak'd (redelivery);
	// handler returning nil causes the message to be Ack'd.
	Subscribe(subject, queue string, handler func(ctx context.Context, data []byte) error) error

	// Close unsubscribes all active subscriptions and drains the connection.
	Close() error
}

// NATSSubscriber implements Subscriber using NATS JetStream durable consumers.
type NATSSubscriber struct {
	nc          *nats.Conn
	js          nats.JetStreamContext
	maxInFlight int
	mu          sync.Mutex
	subs        []*nats.Subscription
}

// NewNATSSubscriber connects to NATS at url, creates a JetStream context, and
// returns a ready Subscriber. Additional NATS options (e.g. TLS) are forwarded.
// Returns an error if NATS is unreachable at startup.
func NewNATSSubscriber(url string, opts ...nats.Option) (*NATSSubscriber, error) {
	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("nats subscriber: connect: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("nats subscriber: jetstream: %w", err)
	}

	return &NATSSubscriber{
		nc:          nc,
		js:          js,
		maxInFlight: defaultMaxInFlight,
	}, nil
}

// Subscribe creates a durable JetStream push consumer on subject with queue
// group load-balancing. Messages are delivered starting from the moment of
// subscription (DeliverNew). Up to maxInFlight messages can be in-flight
// simultaneously before the server stops delivering more.
//
// The consumer is identified by the queue name as its durable name, making it
// survive subscriber restarts.
func (s *NATSSubscriber) Subscribe(
	subject, queue string,
	handler func(ctx context.Context, data []byte) error,
) error {
	sub, err := s.js.QueueSubscribe(
		subject,
		queue,
		func(msg *nats.Msg) {
			if err := handler(context.Background(), msg.Data); err != nil {
				msg.NakWithDelay(0) //nolint:errcheck — immediate redelivery
			} else {
				msg.Ack() //nolint:errcheck
			}
		},
		nats.Durable(queue),
		nats.DeliverNew(),
		nats.MaxAckPending(s.maxInFlight),
		nats.ManualAck(),
	)
	if err != nil {
		return fmt.Errorf("subscribe to %q (queue %q): %w", subject, queue, err)
	}

	s.mu.Lock()
	s.subs = append(s.subs, sub)
	s.mu.Unlock()
	return nil
}

// Close unsubscribes all active subscriptions then drains the NATS connection,
// flushing any pending messages before closing.
func (s *NATSSubscriber) Close() error {
	s.mu.Lock()
	subs := make([]*nats.Subscription, len(s.subs))
	copy(subs, s.subs)
	s.mu.Unlock()

	for _, sub := range subs {
		sub.Unsubscribe() //nolint:errcheck
	}
	return s.nc.Drain()
}
