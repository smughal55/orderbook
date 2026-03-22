package source

import (
	"fmt"
	"log/slog"
	"math/rand"
	"sort"
	"time"

	"github.com/shazmughal/orderbook/internal/model"
)

const (
	maxFieldCount  = 100
	maxFieldKeyLen = 128
	maxMetaValLen  = 1024
)

// Normalize constructs a model.DataRecord from raw source data.
// It validates and sanitises fields and meta, assigns a UUID, and
// stamps the current UTC time.
//
// Rules applied:
//   - Empty source is replaced with "unknown"
//   - Fields are sorted alphabetically; only the first maxFieldCount valid
//     entries are retained — excess are dropped and a warning is logged
//   - Field keys exceeding maxFieldKeyLen characters are skipped
//   - Meta values exceeding maxMetaValLen characters are truncated
//   - nil fields / meta produce empty non-nil maps in the result (defensive copy)
func Normalize(source string, fields map[string]float64, meta map[string]string) model.DataRecord {
	if source == "" {
		slog.Warn("normalize: empty source, substituting 'unknown'")
		source = "unknown"
	}

	outFields := make(map[string]float64, min(len(fields), maxFieldCount))
	if len(fields) > 0 {
		// Sort keys so the selection of the first maxFieldCount entries is
		// deterministic and independent of map-iteration order.
		keys := make([]string, 0, len(fields))
		for k := range fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		dropped := 0
		for _, k := range keys {
			if len(k) > maxFieldKeyLen {
				slog.Warn("normalize: field key exceeds max length, skipping",
					"key_prefix", k[:min(32, len(k))], "length", len(k))
				continue
			}
			if len(outFields) >= maxFieldCount {
				dropped++
				continue
			}
			outFields[k] = fields[k]
		}
		if dropped > 0 {
			slog.Warn("normalize: field count exceeded limit, dropped excess",
				"limit", maxFieldCount, "dropped", dropped)
		}
	}

	outMeta := make(map[string]string, len(meta))
	for k, v := range meta {
		if len(v) > maxMetaValLen {
			v = v[:maxMetaValLen]
		}
		outMeta[k] = v
	}

	return model.DataRecord{
		ID:        newFastUUID(),
		Source:    source,
		Timestamp: time.Now().UTC(),
		Fields:    outFields,
		Meta:      outMeta,
	}
}

// newFastUUID generates a version-4 UUID using math/rand.
// Using math/rand instead of crypto/rand avoids syscall overhead on
// high-throughput normalisation paths.
func newFastUUID() string {
	hi := rand.Uint64()
	lo := rand.Uint64()
	// Set version 4: bits 12–15 of the third 16-bit group = 0100.
	hi = (hi & 0xffffffffffff0fff) | 0x0000000000004000
	// Set variant bits: bits 6–7 of byte 8 = 10.
	lo = (lo & 0x3fffffffffffffff) | 0x8000000000000000
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uint32(hi>>32),
		uint16(hi>>16),
		uint16(hi),
		uint16(lo>>48),
		lo&0x0000ffffffffffff,
	)
}
