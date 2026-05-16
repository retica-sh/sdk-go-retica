package retica

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RequestInfo captures the inbound side of an HTTP request, framework-neutral.
// TraceID and ParentSpanID come from incoming X-Retica-Trace-ID and
// X-Retica-Span-ID headers; both empty means this is a root span.
type RequestInfo struct {
	Method       string
	Path         string
	ClientIP     string
	UserAgent    string
	TraceID      string
	ParentSpanID string
	RequestSize  int
}

// ResponseInfo captures the outbound side of an HTTP request, passed to the
// finalize callback returned by Begin.
type ResponseInfo struct {
	StatusCode   int
	ResponseSize int
	Err          error
}

// FinishFunc is returned by Begin and must be called once the handler is done.
type FinishFunc func(ResponseInfo)

// Begin starts a span. Returns the trace/span IDs for response header
// propagation and a Finish callback. When sampled out, returns empty IDs and
// a no-op Finish — callers can omit response headers in that case.
//
// Adapters call Begin before invoking the framework handler so headers can be
// set on the response before the body is written.
func (r *Retica) Begin(req RequestInfo) (traceID, spanID string, finish FinishFunc) {
	if r.o.sampleRate < 1.0 && rand.Float64() > r.o.sampleRate {
		return "", "", func(ResponseInfo) {}
	}

	start := time.Now()

	traceID = req.TraceID
	if traceID == "" {
		traceID = NewTraceID()
	}
	spanID = NewSpanID()

	finish = func(resp ResponseInfo) {
		durationMs := float64(time.Since(start).Microseconds()) / 1000.0
		durationMs = math.Round(durationMs*100) / 100

		status := StatusCodeOK
		if resp.Err != nil || resp.StatusCode >= 400 {
			status = StatusCodeError
		}

		var parentSpanIDPtr *string
		if req.ParentSpanID != "" {
			p := req.ParentSpanID
			parentSpanIDPtr = &p
		}

		r.buf.add(SpanInput{
			TraceID:      traceID,
			SpanID:       spanID,
			ParentSpanID: parentSpanIDPtr,
			ServiceName:  r.o.serviceName,
			SpanKind:     int16(SpanKindServer),
			Name:         fmt.Sprintf("%s %s", req.Method, req.Path),
			StatusCode:   int16(status),
			StartedAt:    start,
			DurationMs:   durationMs,
			Attributes:   buildHTTPAttributes(req, resp),
		})
	}
	return traceID, spanID, finish
}

// TraceHTTP is a convenience wrapper around Begin for adapters that can set
// response headers after running the handler (e.g. fiber). For frameworks
// where headers must be set before body write (e.g. net/http), call Begin
// directly.
func (r *Retica) TraceHTTP(req RequestInfo, run func() ResponseInfo) (string, string) {
	traceID, spanID, finish := r.Begin(req)
	finish(run())
	return traceID, spanID
}

func buildHTTPAttributes(req RequestInfo, resp ResponseInfo) json.RawMessage {
	attrs := map[string]any{
		"http.method":        req.Method,
		"http.url":           req.Path,
		"http.status_code":   resp.StatusCode,
		"http.request_size":  req.RequestSize,
		"http.response_size": resp.ResponseSize,
		"http.client_ip":     req.ClientIP,
		"http.user_agent":    req.UserAgent,
	}
	data, _ := json.Marshal(attrs)
	return data
}
