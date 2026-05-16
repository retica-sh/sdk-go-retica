# sdk-go-retica

Framework-neutral Go SDK for [Retica](https://retica.sh). This is the core
package; pair it with a framework adapter (e.g. `sdk-go-fiber-v2`,
`sdk-go-net-http`).

## Install

```bash
go get github.com/retica-sh/sdk-go-retica
```

## Use

Most users import a framework adapter rather than this package directly. The
adapter creates a `*retica.Retica` and wires it into the framework's middleware
pipeline.

For custom (non-HTTP) instrumentation:

```go
import retica "github.com/retica-sh/sdk-go-retica"

r := retica.New(
    retica.WithIngestKey("ik_live_..."),
    retica.WithServiceName("my-service"),
)
defer r.Shutdown()

// Manually record a span:
r.Submit(retica.SpanInput{
    TraceID:    retica.NewTraceID(),
    SpanID:     retica.NewSpanID(),
    ServiceName: "my-service",
    Name:       "db.query users",
    SpanKind:   int16(retica.SpanKindClient),
    StatusCode: int16(retica.StatusCodeOK),
    StartedAt:  time.Now(),
    DurationMs: 12.4,
})
```

## Options

| Option | Default | Description |
|---|---|---|
| `WithIngestKey(string)` | env `RETICA_INGEST_KEY` | API key (`ik_live_...` / `ik_test_...`) |
| `WithIngestURL(string)` | env `RETICA_INGEST_URL` or `http://localhost:3000` | Ingest API base URL |
| `WithServiceName(string)` | env `RETICA_SERVICE_NAME` or `unknown` | Service name in spans |
| `WithBatchSize(int)` | 256 | Spans per flush; clamped to [1, 1000] |
| `WithFlushInterval(time.Duration)` | 5s | How often the buffer flushes |
| `WithSampleRate(float64)` | 1.0 | Fraction of requests to trace; clamped to [0, 1] |
| `WithErrorHandler(func(error))` | nil (silent) | Called on ingest failures |

Code-based options override environment variables.

## Adapter API

Adapters call `r.Begin(req)` before invoking the framework handler so headers
can be set on the response before the body writes:

```go
traceID, spanID, finish := r.Begin(retica.RequestInfo{
    Method:       req.Method,
    Path:         req.URL.Path,
    TraceID:      req.Header.Get("X-Retica-Trace-ID"),
    ParentSpanID: req.Header.Get("X-Retica-Span-ID"),
    // ...
})
if traceID != "" {
    w.Header().Set("X-Retica-Trace-ID", traceID)
    w.Header().Set("X-Retica-Span-ID", spanID)
}
// run handler ...
finish(retica.ResponseInfo{StatusCode: 200, ResponseSize: n})
```

For adapters that can set headers after the handler runs (e.g. fiber), use
the `TraceHTTP(req, run)` convenience wrapper.

When the request is sampled out, `Begin` returns empty IDs and a no-op
`finish`.

## Shutdown

Call `Shutdown` before exit to flush pending spans:

```go
defer r.Shutdown()
```

## License

MIT.
