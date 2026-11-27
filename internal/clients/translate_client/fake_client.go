package translate_client

import (
	"context"
	"sync"
	"time"
)

// FakeClient is a deterministic Client for tests: it returns "[target] text" unless Err is set.
type FakeClient struct {
	Err error
	// Delay makes Translate wait (or fail with the context error) to simulate a slow server.
	Delay time.Duration

	mu        sync.Mutex
	calls     int
	lastBatch int
}

var _ Client = (*FakeClient)(nil)

func (f *FakeClient) Translate(ctx context.Context, texts []string, _, target string) ([]string, error) {
	f.mu.Lock()
	f.calls++
	f.lastBatch = len(texts)
	f.mu.Unlock()
	if f.Delay > 0 {
		select {
		case <-time.After(f.Delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if f.Err != nil {
		return nil, f.Err
	}
	out := make([]string, len(texts))
	for i, t := range texts {
		out[i] = "[" + target + "] " + t
	}
	return out, nil
}

// Calls returns how many times Translate was called.
func (f *FakeClient) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// LastBatch returns how many texts the last Translate call received.
func (f *FakeClient) LastBatch() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastBatch
}
