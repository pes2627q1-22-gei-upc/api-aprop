package firebase_client

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestSplitTargets(t *testing.T) {
	tokens, fids := splitTargets([]string{"fid1:APA91b-token", "fidOnly", "fid2:APA91b-other"})
	if want := []string{"fid1:APA91b-token", "fid2:APA91b-other"}; !reflect.DeepEqual(tokens, want) {
		t.Errorf("tokens = %v, want %v", tokens, want)
	}
	if want := []string{"fidOnly"}; !reflect.DeepEqual(fids, want) {
		t.Errorf("fids = %v, want %v", fids, want)
	}
}

func TestChunkTargets(t *testing.T) {
	targets := make([]string, 1001)
	for i := range targets {
		targets[i] = fmt.Sprintf("t%d", i)
	}
	chunks := chunkTargets(targets, maxPushTargets)
	if len(chunks) != 3 || len(chunks[0]) != 500 || len(chunks[1]) != 500 || len(chunks[2]) != 1 {
		t.Fatalf("chunk sizes = %d chunks, want 500/500/1", len(chunks))
	}
	if chunks[2][0] != "t1000" {
		t.Errorf("last target = %q, want t1000", chunks[2][0])
	}
	if got := chunkTargets(nil, maxPushTargets); len(got) != 0 {
		t.Errorf("chunkTargets(nil) = %v, want no chunks", got)
	}
}

func TestAddResults(t *testing.T) {
	errInvalid := errors.New("unregistered")
	errDown := errors.New("unavailable")
	var result PushResult
	addResults(&result, []string{"ok", "gone", "retry", "ok2"}, []error{nil, errInvalid, errDown, nil},
		func(err error) bool { return errors.Is(err, errInvalid) })

	if result.SuccessCount != 2 {
		t.Errorf("SuccessCount = %d, want 2", result.SuccessCount)
	}
	if !reflect.DeepEqual(result.InvalidTokens, []string{"gone"}) {
		t.Errorf("InvalidTokens = %v, want [gone]", result.InvalidTokens)
	}
	if result.TransientFailures != 1 {
		t.Errorf("TransientFailures = %d, want 1", result.TransientFailures)
	}
}
