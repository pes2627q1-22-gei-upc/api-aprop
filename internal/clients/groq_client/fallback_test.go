package groq_client

import (
	"context"
	"errors"
	"testing"
)

type stubClient struct {
	err   error
	calls int
}

func (s *stubClient) ChatCompletion(context.Context, ChatCompletionRequest) (ChatCompletionResponse, error) {
	s.calls++
	return ChatCompletionResponse{ID: "ok"}, s.err
}

func TestFallbackClient(t *testing.T) {
	boom := errors.New("boom")
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	tests := map[string]struct {
		ctx       context.Context
		first     error
		second    error
		wantErr   bool
		wantCalls [2]int
	}{
		"first answers":            {context.Background(), nil, nil, false, [2]int{1, 0}},
		"falls back on failure":    {context.Background(), boom, nil, false, [2]int{1, 1}},
		"all fail":                 {context.Background(), boom, boom, true, [2]int{1, 1}},
		"no fallback when expired": {canceled, boom, nil, true, [2]int{1, 0}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			a, b := &stubClient{err: tt.first}, &stubClient{err: tt.second}
			_, err := NewFallbackClient(a, b).ChatCompletion(tt.ctx, ChatCompletionRequest{})
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v", err)
			}
			if [2]int{a.calls, b.calls} != tt.wantCalls {
				t.Errorf("calls = %d,%d want %v", a.calls, b.calls, tt.wantCalls)
			}
		})
	}
	if _, err := NewFallbackClient().ChatCompletion(context.Background(), ChatCompletionRequest{}); err == nil {
		t.Error("an empty fallback client must fail")
	}
}
