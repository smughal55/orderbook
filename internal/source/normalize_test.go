package source

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// newFastUUID
// ---------------------------------------------------------------------------

func TestNewFastUUID_Format(t *testing.T) {
	id := newFastUUID()
	// UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx  (36 chars)
	if len(id) != 36 {
		t.Fatalf("UUID length: want 36, got %d (%q)", len(id), id)
	}
	parts := strings.Split(id, "-")
	if len(parts) != 5 {
		t.Fatalf("UUID must have 5 dash-separated groups, got %d (%q)", len(parts), id)
	}
	wantLens := []int{8, 4, 4, 4, 12}
	for i, p := range parts {
		if len(p) != wantLens[i] {
			t.Errorf("UUID group %d: want len %d, got %d (%q)", i, wantLens[i], len(p), p)
		}
	}
	// Version nibble must be '4'.
	if parts[2][0] != '4' {
		t.Errorf("UUID version nibble: want '4', got %q", string(parts[2][0]))
	}
	// Variant nibble must be '8', '9', 'a', or 'b'.
	v := parts[3][0]
	if v != '8' && v != '9' && v != 'a' && v != 'b' {
		t.Errorf("UUID variant nibble: want 8/9/a/b, got %q", string(v))
	}
}

func TestNewFastUUID_Unique(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := range 1000 {
		id := newFastUUID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate UUID generated on iteration %d: %q", i, id)
		}
		seen[id] = struct{}{}
	}
}

// ---------------------------------------------------------------------------
// Normalize — happy path
// ---------------------------------------------------------------------------

func TestNormalize_ValidInput(t *testing.T) {
	before := time.Now().UTC()
	r := Normalize("wss",
		map[string]float64{"price": 100.5, "qty": 3},
		map[string]string{"url": "ws://host/feed"},
	)
	after := time.Now().UTC()

	if r.ID == "" {
		t.Error("ID must not be empty")
	}
	if len(r.ID) != 36 {
		t.Errorf("ID length: want 36, got %d", len(r.ID))
	}
	if r.Timestamp.IsZero() {
		t.Error("Timestamp must not be zero")
	}
	if r.Timestamp.Before(before) || r.Timestamp.After(after) {
		t.Errorf("Timestamp %v not in expected range [%v, %v]", r.Timestamp, before, after)
	}
	if r.Source != "wss" {
		t.Errorf("Source: want %q, got %q", "wss", r.Source)
	}
	if r.Fields["price"] != 100.5 || r.Fields["qty"] != 3 {
		t.Errorf("Fields not copied correctly: %v", r.Fields)
	}
	if r.Meta["url"] != "ws://host/feed" {
		t.Errorf("Meta not copied correctly: %v", r.Meta)
	}
}

// ---------------------------------------------------------------------------
// Normalize — source validation
// ---------------------------------------------------------------------------

func TestNormalize_EmptySource(t *testing.T) {
	r := Normalize("", map[string]float64{"x": 1}, nil)
	if r.Source != "unknown" {
		t.Errorf("Source: want %q, got %q", "unknown", r.Source)
	}
}

// ---------------------------------------------------------------------------
// Normalize — nil map handling
// ---------------------------------------------------------------------------

func TestNormalize_NilFields(t *testing.T) {
	r := Normalize("src", nil, nil)
	if r.Fields == nil {
		t.Error("Fields must be a non-nil map, got nil")
	}
	if len(r.Fields) != 0 {
		t.Errorf("Fields must be empty, got %v", r.Fields)
	}
}

func TestNormalize_NilMeta(t *testing.T) {
	r := Normalize("src", map[string]float64{"a": 1}, nil)
	if r.Meta == nil {
		t.Error("Meta must be a non-nil map, got nil")
	}
	if len(r.Meta) != 0 {
		t.Errorf("Meta must be empty, got %v", r.Meta)
	}
}

// ---------------------------------------------------------------------------
// Normalize — field count cap (100)
// ---------------------------------------------------------------------------

func TestNormalize_FieldCountExceeds100(t *testing.T) {
	fields := make(map[string]float64, 150)
	for i := range 150 {
		fields[fmt.Sprintf("field-%03d", i)] = float64(i)
	}

	r := Normalize("src", fields, nil)

	if got := len(r.Fields); got != maxFieldCount {
		t.Errorf("field count: want %d, got %d", maxFieldCount, got)
	}
	// Keys are sorted alphabetically: field-000 … field-099 are kept;
	// field-100 … field-149 are dropped.
	for i := range 100 {
		k := fmt.Sprintf("field-%03d", i)
		if _, ok := r.Fields[k]; !ok {
			t.Errorf("expected field %q to be present", k)
		}
	}
	for i := 100; i < 150; i++ {
		k := fmt.Sprintf("field-%03d", i)
		if _, ok := r.Fields[k]; ok {
			t.Errorf("expected field %q to be dropped, but it is present", k)
		}
	}
}

func TestNormalize_FieldCountExactly100(t *testing.T) {
	fields := make(map[string]float64, 100)
	for i := range 100 {
		fields[fmt.Sprintf("f%02d", i)] = float64(i)
	}
	r := Normalize("src", fields, nil)
	if got := len(r.Fields); got != 100 {
		t.Errorf("field count: want 100, got %d", got)
	}
}

// ---------------------------------------------------------------------------
// Normalize — field key length cap (128)
// ---------------------------------------------------------------------------

func TestNormalize_FieldKeyTooLong(t *testing.T) {
	longKey := strings.Repeat("k", maxFieldKeyLen+1)
	r := Normalize("src", map[string]float64{
		"valid": 1,
		longKey: 2,
	}, nil)

	if _, ok := r.Fields[longKey]; ok {
		t.Error("field with key > 128 chars must be dropped")
	}
	if r.Fields["valid"] != 1 {
		t.Error("valid short-key field must be retained")
	}
}

func TestNormalize_FieldKeyExactly128(t *testing.T) {
	k128 := strings.Repeat("x", maxFieldKeyLen)
	r := Normalize("src", map[string]float64{k128: 42}, nil)
	if _, ok := r.Fields[k128]; !ok {
		t.Error("field with key of exactly 128 chars must be kept")
	}
}

// ---------------------------------------------------------------------------
// Normalize — meta value length cap (1024)
// ---------------------------------------------------------------------------

func TestNormalize_MetaValueTruncated(t *testing.T) {
	longVal := strings.Repeat("v", 2000)
	r := Normalize("src", nil, map[string]string{"key": longVal})

	got := r.Meta["key"]
	if len(got) != maxMetaValLen {
		t.Errorf("meta value length: want %d, got %d", maxMetaValLen, len(got))
	}
	if got != longVal[:maxMetaValLen] {
		t.Error("meta value must be the first 1024 characters of the original")
	}
}

func TestNormalize_MetaValueExactly1024(t *testing.T) {
	val1024 := strings.Repeat("v", maxMetaValLen)
	r := Normalize("src", nil, map[string]string{"k": val1024})
	if len(r.Meta["k"]) != maxMetaValLen {
		t.Errorf("meta value of exactly 1024 chars must not be truncated")
	}
}

// ---------------------------------------------------------------------------
// Normalize — defensive copy
// ---------------------------------------------------------------------------

func TestNormalize_DefensiveCopy_Fields(t *testing.T) {
	input := map[string]float64{"price": 50}
	r := Normalize("src", input, nil)

	// Mutate the input after normalisation.
	input["price"] = 999
	input["new"] = 1

	if r.Fields["price"] != 50 {
		t.Errorf("mutating input fields affected the DataRecord: price=%v", r.Fields["price"])
	}
	if _, ok := r.Fields["new"]; ok {
		t.Error("key added to input after Normalize must not appear in DataRecord")
	}
}

func TestNormalize_DefensiveCopy_Meta(t *testing.T) {
	input := map[string]string{"k": "original"}
	r := Normalize("src", nil, input)

	input["k"] = "modified"
	input["extra"] = "new"

	if r.Meta["k"] != "original" {
		t.Errorf("mutating input meta affected the DataRecord: got %q", r.Meta["k"])
	}
	if _, ok := r.Meta["extra"]; ok {
		t.Error("key added to input after Normalize must not appear in DataRecord")
	}
}

// ---------------------------------------------------------------------------
// Normalize — zero-valued fields are retained
// ---------------------------------------------------------------------------

// A float64 zero value must not be confused with "absent" — the map lookup
// returns 0.0 for missing keys too, so only an explicit presence check is correct.
func TestNormalize_ZeroValuedFieldRetained(t *testing.T) {
	r := Normalize("src", map[string]float64{"zero": 0.0, "nonzero": 1.5}, nil)

	if _, ok := r.Fields["zero"]; !ok {
		t.Error("field with value 0.0 must be retained, not treated as absent")
	}
	if r.Fields["zero"] != 0.0 {
		t.Errorf("field 'zero': want 0.0, got %v", r.Fields["zero"])
	}
}

// ---------------------------------------------------------------------------
// Normalize — long keys don't count toward the 100-field cap
// ---------------------------------------------------------------------------

// Long-key fields are silently dropped before the count cap is checked.
// 5 oversized keys + 100 valid keys → 100 results, 0 counted as "dropped".
func TestNormalize_LongKeysDoNotCountTowardCap(t *testing.T) {
	fields := make(map[string]float64, 105)
	longKey := strings.Repeat("x", maxFieldKeyLen+1)
	for i := range 5 {
		fields[fmt.Sprintf("%s-%d", longKey, i)] = 0
	}
	for i := range 100 {
		fields[fmt.Sprintf("valid-%03d", i)] = float64(i)
	}

	r := Normalize("src", fields, nil)

	if got := len(r.Fields); got != maxFieldCount {
		t.Errorf("field count: want %d (long keys must not consume cap), got %d", maxFieldCount, got)
	}
	for i := range 100 {
		k := fmt.Sprintf("valid-%03d", i)
		if _, ok := r.Fields[k]; !ok {
			t.Errorf("valid field %q must be present", k)
		}
	}
}

// ---------------------------------------------------------------------------
// Normalize — Timestamp is UTC
// ---------------------------------------------------------------------------

func TestNormalize_TimestampIsUTC(t *testing.T) {
	r := Normalize("src", nil, nil)
	if loc := r.Timestamp.Location(); loc != time.UTC {
		t.Errorf("Timestamp location: want UTC, got %v", loc)
	}
}

// ---------------------------------------------------------------------------
// Normalize — successive calls produce distinct IDs
// ---------------------------------------------------------------------------

func TestNormalize_UniqueIDs(t *testing.T) {
	r1 := Normalize("src", nil, nil)
	r2 := Normalize("src", nil, nil)
	if r1.ID == r2.ID {
		t.Errorf("successive Normalize calls must produce distinct IDs, both got %q", r1.ID)
	}
}
