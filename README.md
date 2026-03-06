# Order Book + Real-Time Alerting System

A real-time order book engine and alerting system in Go. Ingests high-throughput data from multiple sources, evaluates single-record and trend-based alert conditions, and delivers notifications via email, SMS, and webhooks.

## System Architecture

```
Sources (WS/REST/...)
        │
        ▼
┌───────────────────┐     ┌───────────────┐
│  Ingestion Layer  │────►│     Kafka     │
│ (Normalizer+Dedup)│     │  (raw-data)   │
└───────────────────┘     └───────┬───────┘
        │ Redis dedup             │
        ▼                 ┌───────┴────────┐
┌──────────────┐   ┌──────┴─────┐  ┌───────┴──────┐
│  Convex DB   │   │Single-Eval │  │  Trend-Eval  │
│ (Alert Rules)│◄──┤(expr-lang) │  │(Redis window)│
└──────────────┘   └──────┬─────┘  └───────┬──────┘
        ▲                 └────┬───────────┘
   React UI                    ▼
                      ┌────────────────┐
                      │  Rule Engine   │
                      │(cooldown/dedup)│
                      └────────┬───────┘
                               │ Kafka (alerts-triggered)
                               ▼
                      ┌────────────────┐
                      │ Delivery Layer │
                      ├────┬─────┬─────┤
                      │Email│SMS │Hook │
                      └────┴─────┴─────┘
```

## Project Structure

```
orderbook/
├── cmd/
│   ├── ingestion/         Normalizer gateway service
│   ├── processor/         Single-record + trend evaluator + rule engine
│   └── delivery/          Alert delivery service
├── internal/
│   ├── model/             Shared types (DataRecord, AlertRule, TriggeredAlert)
│   ├── adapter/           Source adapters (WebSocket, REST, dedup)
│   ├── evaluator/         Single-record and trend evaluation logic
│   ├── ruleengine/        Rule matching, cooldown, suppression
│   ├── delivery/          Channel dispatchers (email, SMS, webhook)
│   └── metrics/           Prometheus instrumentation
├── convex/
│   ├── schema.ts          Convex DB schema (alerts, alertHistory)
│   ├── alerts.ts          Alert CRUD mutations/queries
│   └── alertHistory.ts    Alert history queries
├── web/                   React + TanStack Router UI
│   └── src/
│       ├── App.tsx         Main app with tabs
│       └── components/     AlertsTable, AlertHistory, CreateAlertModal
├── deploy/
│   ├── docker-compose.yml Local dev (Kafka, Redis, Prometheus, Grafana)
│   ├── prometheus.yml     Prometheus scrape config
│   └── k8s/               Kubernetes manifests
├── main.go                Original orderbook entry point
├── orderbook.go           Core OrderBook data structure
├── messages.go            JSON message types and parser
├── client.go              WebSocket client + median ticker
├── mock_server.go         Built-in mock WebSocket server
├── orderbook_test.go      Unit tests (12 tests)
├── integration_test.go    Integration tests (2 tests)
├── go.mod
└── go.sum
```

## Alerting System

### Services

The alerting system runs as three independent Go services communicating via Kafka:

**Ingestion** (`cmd/ingestion/`) — Connects to data sources via pluggable adapters, normalizes records into a common `DataRecord` envelope, deduplicates via Redis `SET NX`, and publishes to Kafka `raw-data` topic.

```bash
go run ./cmd/ingestion -redis localhost:6379 -kafka localhost:9092 -sources pricefeed=ws://localhost:8080/ws
```

**Processor** (`cmd/processor/`) — Consumes from `raw-data`, runs two parallel evaluation paths (single-record via `expr-lang/expr` and trend via Redis sliding windows), applies cooldown/suppression, and publishes triggered alerts to `alerts-triggered`.

```bash
go run ./cmd/processor -redis localhost:6379 -kafka localhost:9092
```

**Delivery** (`cmd/delivery/`) — Consumes from `alerts-triggered`, dispatches to email/SMS/webhook handlers with exponential backoff retry, failed deliveries go to a dead-letter topic.

```bash
go run ./cmd/delivery -kafka localhost:9092
```

### Alert Configuration UI

A React app using TanStack Router and Convex for real-time data:

```bash
cd web && npm install && npm run dev
```

### Infrastructure (Local Dev)

```bash
cd deploy && docker-compose up -d
```

This starts Kafka, Redis, Prometheus, and Grafana. Prometheus scrapes metrics from all three services; Grafana is available at `http://localhost:3000` (admin/admin).

### Alert Rule Types

- **Single Record**: Evaluates each record individually against an expression (e.g., `payload.price > 150`)
- **Trend**: Aggregates values over a sliding time window using Redis sorted sets. Supports avg, max, min, and rate_of_change aggregations.

### Delivery Channels

- **Email**: Stub handler (plug in SendGrid/AWS SES)
- **SMS**: Stub handler (plug in Twilio)
- **Webhook**: Full implementation with HTTP POST, exponential backoff retry (3 attempts)

### Key Dependencies

| Dependency | Purpose |
|---|---|
| [IBM/sarama](https://github.com/IBM/sarama) | Kafka producer and consumer |
| [redis/go-redis](https://github.com/redis/go-redis) | Redis client (dedup, windows, cooldown) |
| [expr-lang/expr](https://github.com/expr-lang/expr) | Expression evaluator for alert conditions |
| [prometheus/client_golang](https://github.com/prometheus/client_golang) | Metrics instrumentation |
| [gorilla/websocket](https://github.com/gorilla/websocket) | WebSocket client and server |

---

## Original Order Book

## Design Decisions

### Data structure: map + sorted slice

Each side of the book (`OrderSide`) uses two complementary structures:

| Structure | Purpose | Complexity |
|---|---|---|
| `map[float64]float64` | O(1) lookup, update, and delete by price | O(1) |
| `[]float64` (sorted) | Ordered prices for O(1) best-bid/best-ask access | O(log n) search, O(n) insert/delete |

Insertions use `sort.SearchFloat64s` for binary search, then shift the slice.  At the expected scale (tens to low hundreds of price levels, updates every 20ms), this is more than fast enough and avoids pulling in an external balanced-tree dependency.

**Why not a heap?**  A heap gives O(1) min/max and O(log n) insert, but does not support efficient arbitrary removal by price — a critical operation for order book updates.

**Why not a B-tree or red-black tree?**  These would improve insert/delete from O(n) to O(log n), but for typical order book depths the sorted slice is simpler, has better cache locality, and keeps the dependency count at one (gorilla/websocket).

### Concurrency

`OrderBook` is protected by a `sync.RWMutex`.  The WebSocket reader goroutine takes a write lock for `ApplySnapshot`/`ApplyUpdate`; the ticker goroutine takes a read lock for `MedianPrice`.  This allows the 200ms median calculation to proceed without blocking incoming updates (unless a write is in progress).

### Message parsing

A two-pass parse strategy: first unmarshal only the `"type"` field into a lightweight envelope struct, then unmarshal the full payload into the correct concrete type.  This avoids a monolithic union struct and keeps each message type cleanly separated.

### Mock server

The built-in mock server generates a 10-level snapshot centered around a mid-price of 100.00, then sends randomized updates every 20ms (80% insert/modify, 20% remove).  It is reusable in tests via `httptest.NewServer` without binding to a real port.

## Dependencies

| Dependency | Version | Purpose |
|---|---|---|
| [gorilla/websocket](https://github.com/gorilla/websocket) | v1.5.3 | WebSocket client and server |

No other external dependencies.

## Usage

### Run with the built-in mock server

```bash
go run . -mock
```

This starts the mock WebSocket server on `:8080` and connects the client to it.  Median prices are logged every 200ms:

```
2026/02/19 17:31:21 mock server started on :8080
2026/02/19 17:31:21 connecting to ws://localhost:8080/ws
2026/02/19 17:31:21 snapshot applied: 10 bids, 10 asks
2026/02/19 17:31:21 median=100.0000  best_bid=99.90  best_ask=100.10
2026/02/19 17:31:21 median=99.9850   best_bid=99.93  best_ask=100.04
...
```

Press `Ctrl+C` to shut down gracefully.

### Connect to an external WebSocket server

```bash
go run . -url ws://your-server:9090/feed
```

### CLI flags

| Flag | Default | Description |
|---|---|---|
| `-mock` | `false` | Start the built-in mock WebSocket server |
| `-mock-addr` | `:8080` | Listen address for the mock server |
| `-url` | `ws://localhost:8080/ws` | WebSocket URL to connect to |

## Testing

### Run all tests

```bash
go test -v ./...
```

### Unit tests (`orderbook_test.go`)

12 tests covering:

- **Empty book** — `BestBid`, `BestAsk`, `MedianPrice` all return false
- **Snapshot application** — correct best bid, best ask, and median
- **Snapshot replacement** — second snapshot fully replaces the first
- **Update insert** — new price level becomes the new best
- **Update modify** — amount changes, price ordering unchanged
- **Update remove** — removing the best level falls back to the next
- **Remove nonexistent** — no-op, book unchanged
- **Ask-side updates** — updates correctly target the ask side
- **Median with empty side** — returns false when one side is cleared
- **Message parsing** — snapshot, update, and unknown type handling

### Integration tests (`integration_test.go`)

2 tests that spin up an in-process WebSocket server via `httptest.NewServer`:

- **Mock server integration** — connects to the random mock feed, verifies bid < ask and median = (bid + ask) / 2 after 300ms of updates
- **Deterministic sequence** — sends a hand-crafted snapshot followed by specific updates (including a removal), then asserts exact expected values

### JSON message format

**Snapshot:**

```json
{
  "type": "snapshot",
  "bids": [{"price": 100.50, "amount": 10.0}],
  "asks": [{"price": 101.00, "amount": 5.0}]
}
```

**Order Update:**

```json
{
  "type": "update",
  "side": "bid",
  "price": 100.75,
  "amount": 3.0
}
```

An `amount` of `0` removes the price level from the book.
