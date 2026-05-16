package retica

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type buffer struct {
	mu      sync.Mutex
	spans   []SpanInput
	o       opts
	client  *http.Client
	stopCh  chan struct{}
	stopped bool
}

func newBuffer(o opts) *buffer {
	b := &buffer{
		spans:  make([]SpanInput, 0, o.batchSize),
		o:      o,
		client: &http.Client{Timeout: 10 * time.Second},
		stopCh: make(chan struct{}),
	}
	go b.flushLoop()
	return b
}

func (b *buffer) add(span SpanInput) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.stopped {
		return
	}

	b.spans = append(b.spans, span)

	if len(b.spans) >= b.o.batchSize {
		b.flushLocked()
	}
}

func (b *buffer) stop() {
	b.mu.Lock()
	b.stopped = true
	b.flushLocked()
	b.mu.Unlock()

	close(b.stopCh)
}

func (b *buffer) flushLoop() {
	ticker := time.NewTicker(b.o.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.mu.Lock()
			b.flushLocked()
			b.mu.Unlock()
		case <-b.stopCh:
			return
		}
	}
}

func (b *buffer) flushLocked() {
	if len(b.spans) == 0 {
		return
	}

	spans := b.spans
	b.spans = make([]SpanInput, 0, b.o.batchSize)

	go b.send(spans)
}

func (b *buffer) send(spans []SpanInput) {
	req := createSpansRequest{Spans: spans}
	body, err := json.Marshal(req)
	if err != nil {
		b.handleError(fmt.Errorf("retica: marshal spans: %w", err))
		return
	}

	url := b.o.ingestURL + "/v1/spans"
	httpReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		b.handleError(fmt.Errorf("retica: create request: %w", err))
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Ingest-Key", b.o.ingestKey)

	resp, err := b.client.Do(httpReq)
	if err != nil {
		b.handleError(fmt.Errorf("retica: send spans: %w", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b.handleError(fmt.Errorf("retica: ingest returned %d", resp.StatusCode))
	}
}

func (b *buffer) handleError(err error) {
	if b.o.errorHandler != nil {
		b.o.errorHandler(err)
	}
}
