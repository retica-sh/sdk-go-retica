package retica

import (
	"os"
	"time"
)

type opts struct {
	ingestKey     string
	ingestURL     string
	serviceName   string
	batchSize     int
	flushInterval time.Duration
	sampleRate    float64
	errorHandler  func(err error)
}

// OptFunc is a functional option for configuring the Retica SDK.
type OptFunc func(*opts)

func defaultOpts() opts {
	return opts{
		ingestKey:     os.Getenv("RETICA_INGEST_KEY"),
		ingestURL:     envOrDefault("RETICA_INGEST_URL", "http://localhost:3000"),
		serviceName:   envOrDefault("RETICA_SERVICE_NAME", "unknown"),
		batchSize:     256,
		flushInterval: 5 * time.Second,
		sampleRate:    1.0,
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// WithIngestKey sets the API key for authentication.
func WithIngestKey(key string) OptFunc {
	return func(o *opts) { o.ingestKey = key }
}

// WithIngestURL sets the ingest API base URL.
func WithIngestURL(url string) OptFunc {
	return func(o *opts) { o.ingestURL = url }
}

// WithServiceName sets the service name reported in spans.
func WithServiceName(name string) OptFunc {
	return func(o *opts) { o.serviceName = name }
}

// WithBatchSize sets the maximum spans per flush. Clamped to [1, 1000].
func WithBatchSize(n int) OptFunc {
	return func(o *opts) {
		if n < 1 {
			n = 1
		}
		if n > 1000 {
			n = 1000
		}
		o.batchSize = n
	}
}

// WithFlushInterval sets how often spans are sent to the server.
func WithFlushInterval(d time.Duration) OptFunc {
	return func(o *opts) { o.flushInterval = d }
}

// WithSampleRate sets the fraction of requests to trace. Clamped to [0.0, 1.0].
func WithSampleRate(rate float64) OptFunc {
	return func(o *opts) {
		if rate < 0 {
			rate = 0
		}
		if rate > 1.0 {
			rate = 1.0
		}
		o.sampleRate = rate
	}
}

// WithErrorHandler sets a callback for ingestion errors. If nil, errors are
// silently dropped.
func WithErrorHandler(fn func(err error)) OptFunc {
	return func(o *opts) { o.errorHandler = fn }
}
