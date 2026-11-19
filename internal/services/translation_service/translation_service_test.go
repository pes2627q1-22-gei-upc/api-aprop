package translation_service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"api-aprop/internal/clients/translate_client"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/repositories/translation_repository"
)

type fakeRepo struct {
	mu       sync.Mutex
	data     map[int]translation_repository.Translation
	getErr   error
	saveErr  error
	saved    int
	language string
}

func (f *fakeRepo) GetTranslations(_ context.Context, _ string, ids []int) (map[int]translation_repository.Translation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	out := map[int]translation_repository.Translation{}
	for _, id := range ids {
		if t, ok := f.data[id]; ok {
			out[id] = t
		}
	}
	return out, nil
}

func (f *fakeRepo) SaveTranslation(_ context.Context, _ string, id int, t translation_repository.Translation) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.saveErr != nil {
		return f.saveErr
	}
	if f.data == nil {
		f.data = map[int]translation_repository.Translation{}
	}
	f.data[id] = t
	f.saved++
	return nil
}

func (f *fakeRepo) GetUserLanguage(context.Context, string) (string, error) { return f.language, nil }

func event(id int, title, description string) event_model.Event {
	return event_model.Event{ID: id, Title: title, Description: description}
}

func TestCatalanDoesNotTranslate(t *testing.T) {
	client := &translate_client.FakeClient{}
	svc := NewTranslationService(&fakeRepo{}, client)
	got := svc.TranslateEvent(context.Background(), "ca", event(1, "Títol", "Descripció"))
	if got.Title != "Títol" || client.Calls() != 0 {
		t.Errorf("title = %q, calls = %d", got.Title, client.Calls())
	}
}

func TestCacheMissTranslatesAndStores(t *testing.T) {
	repo, client := &fakeRepo{}, &translate_client.FakeClient{}
	svc := NewTranslationService(repo, client)
	got := svc.TranslateEvent(context.Background(), "es", event(1, "Títol", "Descripció"))
	if got.Title != "[es] Títol" || got.Description != "[es] Descripció" {
		t.Errorf("got %q / %q", got.Title, got.Description)
	}
	if client.Calls() != 1 || repo.saved != 1 {
		t.Errorf("calls = %d, saved = %d", client.Calls(), repo.saved)
	}

	// The second request is served from the cache
	got = svc.TranslateEvent(context.Background(), "es", event(1, "Títol", "Descripció"))
	if got.Title != "[es] Títol" || client.Calls() != 1 {
		t.Errorf("cache hit: title = %q, calls = %d", got.Title, client.Calls())
	}
}

func TestChangedEventInvalidatesCache(t *testing.T) {
	repo, client := &fakeRepo{}, &translate_client.FakeClient{}
	svc := NewTranslationService(repo, client)
	svc.TranslateEvent(context.Background(), "en", event(1, "Títol", "Descripció"))
	got := svc.TranslateEvent(context.Background(), "en", event(1, "Títol nou", "Descripció"))
	if got.Title != "[en] Títol nou" || client.Calls() != 2 {
		t.Errorf("title = %q, calls = %d", got.Title, client.Calls())
	}
}

func TestTranslateEventsMixesHitsAndMisses(t *testing.T) {
	repo := &fakeRepo{data: map[int]translation_repository.Translation{
		1: {Title: "cached", Description: "cached d", SourceHash: sourceHash("A", "B")},
	}}
	client := &translate_client.FakeClient{}
	svc := NewTranslationService(repo, client)
	in := []event_model.Event{event(1, "A", "B"), event(2, "C", "")}
	got := svc.TranslateEvents(context.Background(), "en", in)
	if got[0].Title != "cached" || got[1].Title != "[en] C" || got[1].Description != "" {
		t.Errorf("got %+v", got)
	}
	if client.Calls() != 1 {
		t.Errorf("calls = %d, want 1", client.Calls())
	}
	if in[1].Title != "C" {
		t.Error("the input slice must not be modified")
	}
}

func TestTranslateMapEvents(t *testing.T) {
	svc := NewTranslationService(&fakeRepo{}, &translate_client.FakeClient{})
	in := []event_model.MapEvent{{Event: event(1, "A", "B"), HasFriends: true}}
	got := svc.TranslateMapEvents(context.Background(), "es", in)
	if got[0].Title != "[es] A" || !got[0].HasFriends {
		t.Errorf("got %+v", got[0])
	}
}

func TestFallbackToOriginal(t *testing.T) {
	tests := []struct {
		name   string
		repo   *fakeRepo
		client *translate_client.FakeClient
	}{
		{"client error", &fakeRepo{}, &translate_client.FakeClient{Err: errors.New("down")}},
		{"client disabled", &fakeRepo{}, &translate_client.FakeClient{Err: translate_client.ErrDisabled}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewTranslationService(tt.repo, tt.client).TranslateEvent(context.Background(), "es", event(1, "Títol", "Descripció"))
			if got.Title != "Títol" || got.Description != "Descripció" || tt.repo.saved != 0 {
				t.Errorf("got %+v, saved = %d", got, tt.repo.saved)
			}
		})
	}
}

func TestRepositoryErrorsDoNotFail(t *testing.T) {
	repo := &fakeRepo{getErr: errors.New("db"), saveErr: errors.New("db")}
	got := NewTranslationService(repo, &translate_client.FakeClient{}).TranslateEvent(context.Background(), "es", event(1, "A", "B"))
	if got.Title != "[es] A" {
		t.Errorf("title = %q", got.Title)
	}
}

func TestUserLanguage(t *testing.T) {
	lang, err := NewTranslationService(&fakeRepo{language: "en"}, &translate_client.FakeClient{}).UserLanguage(context.Background(), "u")
	if err != nil || lang != "en" {
		t.Errorf("lang = %q, err = %v", lang, err)
	}
}

func TestMissesAreBatchedInOneRequest(t *testing.T) {
	client := &translate_client.FakeClient{}
	svc := NewTranslationService(&fakeRepo{}, client)
	got := svc.TranslateEvents(context.Background(), "es", []event_model.Event{event(1, "A", "B"), event(2, "C", ""), event(3, "D", "E")})
	if client.Calls() != 1 || client.LastBatch() != 5 {
		t.Errorf("calls = %d, batch = %d, want 1 and 5", client.Calls(), client.LastBatch())
	}
	if got[0].Title != "[es] A" || got[1].Description != "" || got[2].Description != "[es] E" {
		t.Errorf("got %+v", got)
	}
}

func TestTimeoutFallsBackToOriginal(t *testing.T) {
	old := translateTimeout
	translateTimeout = 20 * time.Millisecond
	defer func() { translateTimeout = old }()
	repo := &fakeRepo{}
	svc := NewTranslationService(repo, &translate_client.FakeClient{Delay: time.Second})
	got := svc.TranslateEvent(context.Background(), "es", event(1, "Títol", "Descripció"))
	if got.Title != "Títol" || repo.saved != 0 {
		t.Errorf("got %+v, saved = %d", got, repo.saved)
	}
}

func TestMissesAreCapped(t *testing.T) {
	client, repo := &translate_client.FakeClient{}, &fakeRepo{}
	svc := NewTranslationService(repo, client)
	in := make([]event_model.Event, maxMisses+10)
	for i := range in {
		in[i] = event(i+1, "T", "")
	}
	got := svc.TranslateEvents(context.Background(), "en", in)
	wantCalls := (maxMisses + batchSize - 1) / batchSize
	if client.Calls() != wantCalls || repo.saved != maxMisses {
		t.Errorf("calls = %d (want %d), saved = %d (want %d)", client.Calls(), wantCalls, repo.saved, maxMisses)
	}
	if got[maxMisses-1].Title != "[en] T" || got[maxMisses].Title != "T" {
		t.Errorf("titles = %q / %q", got[maxMisses-1].Title, got[maxMisses].Title)
	}
}

type failingFirstBatchClient struct {
	failPrefix string
}

func (c *failingFirstBatchClient) Translate(_ context.Context, texts []string, _, target string) ([]string, error) {
	if len(texts) > 0 && texts[0] == c.failPrefix {
		return nil, errors.New("upstream timeout")
	}
	out := make([]string, len(texts))
	for i, text := range texts {
		out[i] = "[" + target + "] " + text
	}
	return out, nil
}

func TestPartialBatchFailureStillCachesOtherBatches(t *testing.T) {
	repo := &fakeRepo{}
	in := make([]event_model.Event, batchSize*2)
	for i := range in {
		title := "ok"
		if i == 0 {
			title = "fail-batch"
		}
		in[i] = event(i+1, title, "")
	}
	svc := NewTranslationService(repo, &failingFirstBatchClient{failPrefix: "fail-batch"})
	got := svc.TranslateEvents(context.Background(), "es", in)

	if got[0].Title != "fail-batch" {
		t.Errorf("failed batch event[0] = %q, want original", got[0].Title)
	}
	if got[batchSize].Title != "[es] ok" {
		t.Errorf("succeeded batch event[%d] = %q, want translated", batchSize, got[batchSize].Title)
	}
	if repo.saved != batchSize {
		t.Errorf("saved = %d, want %d", repo.saved, batchSize)
	}
}
