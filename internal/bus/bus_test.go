package bus

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

// Compile-time interface compliance assertions.
// These cause a build failure if NATSPublisher or NATSSubscriber stops
// satisfying the declared interface, providing a safety net against drift.
var (
	_ Publisher  = (*NATSPublisher)(nil)
	_ Subscriber = (*NATSSubscriber)(nil)
)

// ---------------------------------------------------------------------------
// streamNameFromSubject — pure function
// ---------------------------------------------------------------------------

func TestStreamNameFromSubject(t *testing.T) {
	cases := []struct {
		subject string
		want    string
	}{
		// Normal multi-segment subjects: only the leading segment is returned.
		{"records.source1", "records"},
		{"alerts.rule42.fired", "alerts"},
		{"a.b", "a"},
		// No dot present: the whole subject is the stream name.
		{"noprefix", "noprefix"},
		// Dot at position 0 (i > 0 guard): whole string returned unchanged.
		{".leadingdot", ".leadingdot"},
		// Empty string: returns empty string without panic.
		{"", ""},
	}

	for _, tc := range cases {
		got := streamNameFromSubject(tc.subject)
		if got != tc.want {
			t.Errorf("streamNameFromSubject(%q): want %q, got %q", tc.subject, tc.want, got)
		}
	}
}

// ---------------------------------------------------------------------------
// defaultMaxInFlight constant
// ---------------------------------------------------------------------------

func TestDefaultMaxInFlightValue(t *testing.T) {
	const want = 100
	if defaultMaxInFlight != want {
		t.Errorf("defaultMaxInFlight: want %d, got %d", want, defaultMaxInFlight)
	}
}

// ---------------------------------------------------------------------------
// Constructor failure — unreachable NATS server
// ---------------------------------------------------------------------------

// unreachableNATSURL returns a NATS URL that will always refuse the TCP
// connection. Port 1 (the TCP Port Service Multiplexer) is practically never
// open, giving an immediate ECONNREFUSED.
const unreachableNATSURL = "nats://127.0.0.1:1"

// fastFailOpts returns NATS options that disable reconnect retries and cap
// the dial timeout to 200 ms so the tests finish quickly.
func fastFailOpts() []nats.Option {
	return []nats.Option{
		nats.NoReconnect(),
		nats.Timeout(200 * time.Millisecond),
	}
}

func TestNewNATSPublisherUnreachable(t *testing.T) {
	pub, err := NewNATSPublisher(unreachableNATSURL, fastFailOpts()...)
	if err == nil {
		pub.Close()
		t.Fatal("expected error connecting to unreachable NATS server, got nil")
	}
	if !strings.Contains(err.Error(), "connect") {
		t.Errorf("expected error to mention 'connect', got: %v", err)
	}
}

func TestNewNATSSubscriberUnreachable(t *testing.T) {
	sub, err := NewNATSSubscriber(unreachableNATSURL, fastFailOpts()...)
	if err == nil {
		sub.Close()
		t.Fatal("expected error connecting to unreachable NATS server, got nil")
	}
	if !strings.Contains(err.Error(), "connect") {
		t.Errorf("expected error to mention 'connect', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Integration tests — require embedded NATS JetStream
// ---------------------------------------------------------------------------

// startTestNATS launches an embedded NATS server with JetStream enabled on a
// random port. The server is shut down via t.Cleanup. Returns the client URL.
func startTestNATS(t *testing.T) string {
	t.Helper()
	opts := &server.Options{
		Port:      -1, // OS-assigned random port
		JetStream: true,
		StoreDir:  t.TempDir(),
		NoLog:     true,
		NoSigs:    true,
	}
	s, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("create embedded NATS server: %v", err)
	}
	s.Start()
	if !s.ReadyForConnections(5 * time.Second) {
		t.Fatal("embedded NATS server not ready within 5 s")
	}
	t.Cleanup(func() {
		s.Shutdown()
		s.WaitForShutdown()
	})
	return s.ClientURL()
}

// pubsub is a convenience helper that creates a publisher and subscriber
// connected to url, ensures the JetStream stream exists for subject (by
// publishing one setup message), and returns both ready to use.
// Both are closed via t.Cleanup.
func pubsub(t *testing.T, url, subject string) (*NATSPublisher, *NATSSubscriber) {
	t.Helper()

	pub, err := NewNATSPublisher(url)
	if err != nil {
		t.Fatalf("NewNATSPublisher: %v", err)
	}
	t.Cleanup(func() { pub.Close() })

	// Publish a setup message so the stream exists before any Subscribe call.
	// DeliverNew consumers created afterwards will not receive this message.
	if err := pub.Publish(context.Background(), subject, []byte("__setup__")); err != nil {
		t.Fatalf("setup publish: %v", err)
	}

	sub, err := NewNATSSubscriber(url)
	if err != nil {
		t.Fatalf("NewNATSSubscriber: %v", err)
	}
	t.Cleanup(func() { sub.Close() })

	return pub, sub
}

// collectN reads exactly n items from ch, failing the test with t.Fatalf if
// they do not all arrive within timeout.
func collectN(t *testing.T, ch <-chan string, n int, timeout time.Duration) []string {
	t.Helper()
	out := make([]string, 0, n)
	deadline := time.After(timeout)
	for len(out) < n {
		select {
		case m := <-ch:
			out = append(out, m)
		case <-deadline:
			t.Fatalf("timed out: received %d/%d messages", len(out), n)
		}
	}
	return out
}

// TestIntegration_PublishSubscribeSingle publishes 100 messages to a single
// consumer and verifies all 100 arrive in order.
func TestIntegration_PublishSubscribeSingle(t *testing.T) {
	url := startTestNATS(t)
	pub, sub := pubsub(t, url, "single.events")

	received := make(chan string, 200)
	if err := sub.Subscribe("single.events", "grp-single",
		func(_ context.Context, data []byte) error {
			received <- string(data)
			return nil
		},
	); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	ctx := context.Background()
	for i := range 100 {
		payload := fmt.Sprintf("msg-%d", i)
		if err := pub.Publish(ctx, "single.events", []byte(payload)); err != nil {
			t.Fatalf("Publish %d: %v", i, err)
		}
	}

	msgs := collectN(t, received, 100, 5*time.Second)

	for i, got := range msgs {
		want := fmt.Sprintf("msg-%d", i)
		if got != want {
			t.Errorf("message[%d]: want %q, got %q", i, want, got)
		}
	}
}

// TestIntegration_QueueGroupExclusiveDelivery publishes 100 messages to a
// subject with two consumers in the same queue group and verifies each
// message is delivered to exactly one consumer (total == 100, not 200).
func TestIntegration_QueueGroupExclusiveDelivery(t *testing.T) {
	url := startTestNATS(t)
	pub, _ := pubsub(t, url, "work.events")

	// Second subscriber needs its own connection but shares the queue group.
	sub2, err := NewNATSSubscriber(url)
	if err != nil {
		t.Fatalf("NewNATSSubscriber (sub2): %v", err)
	}
	t.Cleanup(func() { sub2.Close() })

	// Use the subscriber created by pubsub as sub1.
	sub1, err := NewNATSSubscriber(url)
	if err != nil {
		t.Fatalf("NewNATSSubscriber (sub1): %v", err)
	}
	t.Cleanup(func() { sub1.Close() })

	var mu sync.Mutex
	counts := [2]int{}

	makeHandler := func(idx int) func(context.Context, []byte) error {
		return func(_ context.Context, _ []byte) error {
			mu.Lock()
			counts[idx]++
			mu.Unlock()
			return nil
		}
	}

	if err := sub1.Subscribe("work.events", "workers", makeHandler(0)); err != nil {
		t.Fatalf("sub1.Subscribe: %v", err)
	}
	if err := sub2.Subscribe("work.events", "workers", makeHandler(1)); err != nil {
		t.Fatalf("sub2.Subscribe: %v", err)
	}

	ctx := context.Background()
	for i := range 100 {
		if err := pub.Publish(ctx, "work.events", []byte(fmt.Sprintf("msg-%d", i))); err != nil {
			t.Fatalf("Publish %d: %v", i, err)
		}
	}

	// Poll until 100 messages arrive or timeout.
	deadline := time.After(5 * time.Second)
	for {
		mu.Lock()
		total := counts[0] + counts[1]
		mu.Unlock()
		if total == 100 {
			break
		}
		select {
		case <-deadline:
			mu.Lock()
			t.Fatalf("timed out: received %d/100 (sub1=%d sub2=%d)",
				counts[0]+counts[1], counts[0], counts[1])
			mu.Unlock()
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	mu.Lock()
	total := counts[0] + counts[1]
	mu.Unlock()

	if total != 100 {
		t.Errorf("total deliveries: want 100, got %d", total)
	}
	// Verify load was distributed: with 100 messages and 2 members, each must
	// have received at least one. An all-or-nothing split would indicate the
	// queue group is not functioning as expected.
	mu.Lock()
	if counts[0] == 0 || counts[1] == 0 {
		t.Errorf("load not distributed across both queue members: sub1=%d sub2=%d",
			counts[0], counts[1])
	}
	mu.Unlock()
	t.Logf("queue distribution: sub1=%d sub2=%d", counts[0], counts[1])
}

// TestIntegration_HandlerErrorCausesRedelivery verifies that when a handler
// returns a non-nil error the message is Nak'd and redelivered by JetStream.
// The handler fails on the first call and succeeds on the second, which means
// the subscriber must receive the same message at least twice.
func TestIntegration_HandlerErrorCausesRedelivery(t *testing.T) {
	url := startTestNATS(t)
	pub, sub := pubsub(t, url, "retry.events")

	var attempts atomic.Int32
	var once sync.Once
	done := make(chan struct{})
	var redeliveredPayload atomic.Value // stores string payload on second delivery

	if err := sub.Subscribe("retry.events", "retry-workers",
		func(_ context.Context, data []byte) error {
			n := attempts.Add(1)
			if n == 1 {
				// First delivery: return error to force Nak + redelivery.
				return errors.New("transient error — force redelivery")
			}
			// Second delivery: capture payload, then Ack and signal the test.
			redeliveredPayload.Store(string(data))
			once.Do(func() { close(done) })
			return nil
		},
	); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	if err := pub.Publish(context.Background(), "retry.events", []byte("must-retry")); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	select {
	case <-done:
		// Message was successfully processed after at least one redelivery.
	case <-time.After(5 * time.Second):
		t.Fatalf("redelivery did not occur within 5 s (attempts so far: %d)", attempts.Load())
	}

	if got := int(attempts.Load()); got < 2 {
		t.Errorf("expected at least 2 delivery attempts, got %d", got)
	}

	// Verify the redelivered message carried the same payload.
	if got, _ := redeliveredPayload.Load().(string); got != "must-retry" {
		t.Errorf("redelivered payload: want %q, got %q", "must-retry", got)
	}

	// After a successful Ack the server must not redeliver the message.
	// A brief sleep gives NATS time to process any spurious delivery if present.
	time.Sleep(200 * time.Millisecond)
	if got := int(attempts.Load()); got != 2 {
		t.Errorf("after Ack, expected exactly 2 total deliveries, got %d", got)
	}
}
