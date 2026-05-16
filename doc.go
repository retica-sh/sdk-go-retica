// Package retica is the framework-neutral core of the Retica Go SDK. It
// provides span construction, batched ingest, and a TraceHTTP entrypoint
// that framework adapters (fiber, net-http, etc.) call into.
//
// # Direct use
//
// Most users import a framework adapter (e.g. github.com/retica-sh/sdk-go-fiber-v2)
// rather than this package directly. The adapter creates a *retica.Retica
// and wires it into the framework's middleware pipeline.
//
// # Configuration
//
// All options use the functional options pattern:
//
//	r := retica.New(
//	    retica.WithIngestKey("ik_live_..."),       // required (or env)
//	    retica.WithIngestURL("https://..."),        // default: http://localhost:3000
//	    retica.WithServiceName("my-service"),       // default: "unknown"
//	    retica.WithBatchSize(256),                  // default: 256, max: 1000
//	    retica.WithFlushInterval(5 * time.Second),  // default: 5s
//	    retica.WithSampleRate(0.1),                 // default: 1.0
//	    retica.WithSkipPaths("/livez", "/healthz"), // never trace these
//	    retica.WithSkipPathPrefixes("/debug/"),     // never trace /debug/*
//	    retica.WithErrorHandler(func(err error) {   // default: silent
//	        log.Println("retica:", err)
//	    }),
//	)
//	defer r.Shutdown()
//
// # Environment variables
//
//	RETICA_INGEST_KEY     API key (ik_live_... or ik_test_...)
//	RETICA_INGEST_URL     Ingest API base URL (default: http://localhost:3000)
//	RETICA_SERVICE_NAME   Service name in traces (default: "unknown")
//
// With... options take precedence over environment variables.
//
// # TraceHTTP
//
// Adapters call TraceHTTP with a neutral RequestInfo and a callback that
// invokes the framework's handler chain. The callback returns ResponseInfo
// describing what happened. TraceHTTP handles sampling, ID generation,
// duration timing, status mapping, attribute construction, and enqueues
// the span into the batched buffer.
//
// # Batching
//
// Spans are buffered in memory and flushed to the ingest API in batches.
// Two triggers control when a flush occurs:
//   - Size: when the buffer reaches batchSize (default 256)
//   - Time: every flushInterval (default 5s)
//
// Flushing is non-blocking. Call Shutdown before exit to drain remaining spans.
package retica
