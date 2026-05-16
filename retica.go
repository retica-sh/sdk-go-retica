package retica

// Retica is the framework-neutral SDK instance. Create one with New and feed
// it requests via TraceHTTP from a framework adapter (fiber, net-http, etc.).
type Retica struct {
	o   opts
	buf *buffer
}

// New creates a new Retica SDK instance.
//
//	r := retica.New(retica.WithIngestKey("ik_live_..."))
//	defer r.Shutdown()
func New(opts ...OptFunc) *Retica {
	o := defaultOpts()
	for _, fn := range opts {
		fn(&o)
	}

	return &Retica{
		o:   o,
		buf: newBuffer(o),
	}
}

// Shutdown flushes all pending spans and stops the background flusher.
// Call this before your application exits.
func (r *Retica) Shutdown() {
	r.buf.stop()
}

// Submit enqueues a fully-constructed span for batched ingestion. Adapters
// normally use TraceHTTP instead; this is for custom (non-HTTP) spans.
func (r *Retica) Submit(s SpanInput) {
	r.buf.add(s)
}
