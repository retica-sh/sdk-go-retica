package retica

import (
	"testing"
	"time"
)

func newTestRetica(t *testing.T) *Retica {
	t.Helper()
	r := New()
	t.Cleanup(r.Shutdown)
	return r
}

func TestTraceHTTP_RootRequestGetsNewIDs(t *testing.T) {
	r := newTestRetica(t)

	traceID, spanID := r.TraceHTTP(RequestInfo{Method: "GET", Path: "/hello"}, func() ResponseInfo {
		return ResponseInfo{StatusCode: 200}
	})

	if len(traceID) != 32 {
		t.Errorf("traceID length = %d, want 32", len(traceID))
	}
	if len(spanID) != 16 {
		t.Errorf("spanID length = %d, want 16", len(spanID))
	}
}

func TestTraceHTTP_PropagatesIncomingTrace(t *testing.T) {
	r := newTestRetica(t)

	const incoming = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	traceID, _ := r.TraceHTTP(RequestInfo{Method: "GET", Path: "/hello", TraceID: incoming}, func() ResponseInfo {
		return ResponseInfo{StatusCode: 200}
	})

	if traceID != incoming {
		t.Errorf("traceID = %q, want propagated %q", traceID, incoming)
	}
}

func TestTraceHTTP_ParentSpanIDRecorded(t *testing.T) {
	r := newTestRetica(t)

	r.TraceHTTP(RequestInfo{Method: "GET", Path: "/hello", ParentSpanID: "bbbbbbbbbbbbbbbb"}, func() ResponseInfo {
		return ResponseInfo{StatusCode: 200}
	})

	r.buf.mu.Lock()
	defer r.buf.mu.Unlock()
	if len(r.buf.spans) != 1 {
		t.Fatalf("spans = %d, want 1", len(r.buf.spans))
	}
	if r.buf.spans[0].ParentSpanID == nil || *r.buf.spans[0].ParentSpanID != "bbbbbbbbbbbbbbbb" {
		t.Errorf("parent span ID not recorded")
	}
}

func TestTraceHTTP_SpansBuffered(t *testing.T) {
	r := newTestRetica(t)

	for i := 0; i < 5; i++ {
		r.TraceHTTP(RequestInfo{Method: "GET", Path: "/hello"}, func() ResponseInfo {
			return ResponseInfo{StatusCode: 200}
		})
	}

	r.buf.mu.Lock()
	count := len(r.buf.spans)
	r.buf.mu.Unlock()

	if count != 5 {
		t.Errorf("buffered spans = %d, want 5", count)
	}
}

func TestTraceHTTP_SampleZeroDropsAll(t *testing.T) {
	r := New(WithSampleRate(0.0))
	t.Cleanup(r.Shutdown)

	for i := 0; i < 10; i++ {
		traceID, spanID := r.TraceHTTP(RequestInfo{Method: "GET", Path: "/hello"}, func() ResponseInfo {
			return ResponseInfo{StatusCode: 200}
		})
		if traceID != "" || spanID != "" {
			t.Errorf("expected empty IDs when sampled out, got %q/%q", traceID, spanID)
		}
	}

	r.buf.mu.Lock()
	count := len(r.buf.spans)
	r.buf.mu.Unlock()

	if count != 0 {
		t.Errorf("buffered spans = %d, want 0", count)
	}
}

func TestTraceHTTP_ErrorStatusMarked(t *testing.T) {
	r := newTestRetica(t)

	r.TraceHTTP(RequestInfo{Method: "GET", Path: "/error"}, func() ResponseInfo {
		return ResponseInfo{StatusCode: 500}
	})

	r.buf.mu.Lock()
	defer r.buf.mu.Unlock()
	if len(r.buf.spans) != 1 {
		t.Fatalf("spans = %d, want 1", len(r.buf.spans))
	}
	if r.buf.spans[0].StatusCode != int16(StatusCodeError) {
		t.Errorf("status = %d, want ERROR (%d)", r.buf.spans[0].StatusCode, StatusCodeError)
	}
}

func TestOptions_Defaults(t *testing.T) {
	o := defaultOpts()

	if o.ingestURL != "https://ingest.retica.sh" {
		t.Errorf("ingestURL = %q", o.ingestURL)
	}
	if o.serviceName != "unknown" {
		t.Errorf("serviceName = %q", o.serviceName)
	}
	if o.batchSize != 256 {
		t.Errorf("batchSize = %d", o.batchSize)
	}
	if o.flushInterval != 5*time.Second {
		t.Errorf("flushInterval = %v", o.flushInterval)
	}
	if o.sampleRate != 1.0 {
		t.Errorf("sampleRate = %f", o.sampleRate)
	}
}

func TestOptions_Custom(t *testing.T) {
	called := false
	o := defaultOpts()

	WithIngestKey("ik_test_mykey")(&o)
	WithIngestURL("https://example.com")(&o)
	WithServiceName("my-svc")(&o)
	WithBatchSize(500)(&o)
	WithFlushInterval(10 * time.Second)(&o)
	WithSampleRate(0.5)(&o)
	WithErrorHandler(func(err error) { called = true })(&o)

	if o.ingestKey != "ik_test_mykey" {
		t.Errorf("ingestKey = %q", o.ingestKey)
	}
	if o.ingestURL != "https://example.com" {
		t.Errorf("ingestURL = %q", o.ingestURL)
	}
	if o.serviceName != "my-svc" {
		t.Errorf("serviceName = %q", o.serviceName)
	}
	if o.batchSize != 500 {
		t.Errorf("batchSize = %d", o.batchSize)
	}
	if o.flushInterval != 10*time.Second {
		t.Errorf("flushInterval = %v", o.flushInterval)
	}
	if o.sampleRate != 0.5 {
		t.Errorf("sampleRate = %f", o.sampleRate)
	}
	o.errorHandler(nil)
	if !called {
		t.Error("error handler not called")
	}
}

func TestOptions_Boundaries(t *testing.T) {
	o := defaultOpts()

	WithBatchSize(0)(&o)
	if o.batchSize != 1 {
		t.Errorf("batchSize min: got %d", o.batchSize)
	}

	WithBatchSize(5000)(&o)
	if o.batchSize != 1000 {
		t.Errorf("batchSize max: got %d", o.batchSize)
	}

	WithSampleRate(-1)(&o)
	if o.sampleRate != 0 {
		t.Errorf("sampleRate min: got %f", o.sampleRate)
	}

	WithSampleRate(99)(&o)
	if o.sampleRate != 1.0 {
		t.Errorf("sampleRate max: got %f", o.sampleRate)
	}
}

func TestSkipPaths_ExactMatch(t *testing.T) {
	r := New(WithSkipPaths("/healthz"))
	t.Cleanup(r.Shutdown)

	traceID, spanID := r.TraceHTTP(RequestInfo{Method: "GET", Path: "/healthz"}, func() ResponseInfo {
		return ResponseInfo{StatusCode: 200}
	})
	if traceID != "" || spanID != "" {
		t.Errorf("expected empty IDs for skipped path, got %q/%q", traceID, spanID)
	}

	traceID, spanID = r.TraceHTTP(RequestInfo{Method: "GET", Path: "/hello"}, func() ResponseInfo {
		return ResponseInfo{StatusCode: 200}
	})
	if traceID == "" || spanID == "" {
		t.Error("non-skipped path should produce IDs")
	}

	r.buf.mu.Lock()
	count := len(r.buf.spans)
	r.buf.mu.Unlock()
	if count != 1 {
		t.Errorf("buffered spans = %d, want 1", count)
	}
}

func TestSkipPaths_NoPartialMatch(t *testing.T) {
	r := New(WithSkipPaths("/health"))
	t.Cleanup(r.Shutdown)

	traceID, _ := r.TraceHTTP(RequestInfo{Method: "GET", Path: "/healthz"}, func() ResponseInfo {
		return ResponseInfo{StatusCode: 200}
	})
	if traceID == "" {
		t.Error("/healthz should NOT be skipped by exact /health")
	}
}

func TestSkipPathPrefixes_Match(t *testing.T) {
	r := New(WithSkipPathPrefixes("/debug/"))
	t.Cleanup(r.Shutdown)

	traceID, _ := r.TraceHTTP(RequestInfo{Method: "GET", Path: "/debug/pprof"}, func() ResponseInfo {
		return ResponseInfo{StatusCode: 200}
	})
	if traceID != "" {
		t.Error("/debug/pprof should be skipped by prefix /debug/")
	}

	traceID, _ = r.TraceHTTP(RequestInfo{Method: "GET", Path: "/api/debug/"}, func() ResponseInfo {
		return ResponseInfo{StatusCode: 200}
	})
	if traceID == "" {
		t.Error("/api/debug/ should NOT be skipped by prefix /debug/")
	}
}

func TestSkipPaths_HandlerStillRuns(t *testing.T) {
	r := New(WithSkipPaths("/healthz"))
	t.Cleanup(r.Shutdown)

	ran := false
	traceID, _ := r.TraceHTTP(RequestInfo{Method: "GET", Path: "/healthz"}, func() ResponseInfo {
		ran = true
		return ResponseInfo{StatusCode: 200}
	})
	if !ran {
		t.Fatal("handler must run even when path is skipped")
	}
	if traceID != "" {
		t.Error("skipped path should return empty traceID")
	}

	r.buf.mu.Lock()
	count := len(r.buf.spans)
	r.buf.mu.Unlock()
	if count != 0 {
		t.Errorf("buffered spans = %d, want 0", count)
	}
}

func TestSkipPaths_OverridesSampleRate(t *testing.T) {
	r := New(WithSkipPaths("/healthz"), WithSampleRate(1.0))
	t.Cleanup(r.Shutdown)

	traceID, _ := r.TraceHTTP(RequestInfo{Method: "GET", Path: "/healthz"}, func() ResponseInfo {
		return ResponseInfo{StatusCode: 200}
	})
	if traceID != "" {
		t.Error("skip should override 100% sample rate")
	}
}

func TestIDGen(t *testing.T) {
	a := NewTraceID()
	b := NewTraceID()
	if a == b {
		t.Error("trace IDs collide")
	}
	if len(a) != 32 {
		t.Errorf("trace ID length = %d", len(a))
	}

	s1 := NewSpanID()
	s2 := NewSpanID()
	if s1 == s2 {
		t.Error("span IDs collide")
	}
	if len(s1) != 16 {
		t.Errorf("span ID length = %d", len(s1))
	}
}
