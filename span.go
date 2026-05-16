package retica

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"
)

// SpanKind represents the OpenTelemetry span kind.
type SpanKind int16

const (
	SpanKindUnspecified SpanKind = 0
	SpanKindInternal    SpanKind = 1
	SpanKindServer      SpanKind = 2
	SpanKindClient      SpanKind = 3
	SpanKindProducer    SpanKind = 4
	SpanKindConsumer    SpanKind = 5
)

// StatusCode represents the OpenTelemetry status code.
type StatusCode int16

const (
	StatusCodeUnset StatusCode = 0
	StatusCodeOK    StatusCode = 1
	StatusCodeError StatusCode = 2
)

// SpanInput is the JSON payload for a single span sent to the ingest API.
// Adapters construct values of this type via TraceHTTP; direct construction
// is supported for custom (non-HTTP) instrumentation.
type SpanInput struct {
	TraceID            string          `json:"trace_id"`
	SpanID             string          `json:"span_id"`
	ParentSpanID       *string         `json:"parent_span_id,omitempty"`
	ServiceName        string          `json:"service_name"`
	SpanKind           int16           `json:"span_kind"`
	Name               string          `json:"name"`
	StatusCode         int16           `json:"status_code"`
	StartedAt          time.Time       `json:"started_at"`
	DurationMs         float64         `json:"duration_ms"`
	Attributes         json.RawMessage `json:"attributes,omitempty"`
	ResourceAttributes json.RawMessage `json:"resource_attributes,omitempty"`
	Events             json.RawMessage `json:"events,omitempty"`
}

type createSpansRequest struct {
	Spans []SpanInput `json:"spans"`
}

type createSpansResponse struct {
	Inserted int     `json:"inserted"`
	Skipped  int     `json:"skipped"`
	IDs      []int64 `json:"ids"`
}

// NewSpanID generates a random 8-byte span ID as a hex string.
func NewSpanID() string {
	var b [8]byte
	rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// NewTraceID generates a random 16-byte trace ID as a hex string.
func NewTraceID() string {
	var b [16]byte
	rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
