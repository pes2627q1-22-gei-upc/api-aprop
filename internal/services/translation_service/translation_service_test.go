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

func fullEvent(id int) event_model.Event {
	return event_model.Event{
		ID:          id,
		Title:       "Títol",
		Subtitle:    "Subtítol",
		Description: "Descripció",
		Schedule:    "De dimarts a diumenge, de 10 a 19 h",
		Tickets:     "Entrada gratuïta amb reserva prèvia",
		URLnames:    "Web oficial,Compra d'entrades",
	}
}

func TestCatalanDoesNotTranslate(t *testing.T) {
	client := &translate_client.FakeClient{}
	svc := NewTranslationService(&fakeRepo{}, client)
	in := fullEvent(1)
	got := svc.TranslateEvent(context.Background(), "ca", in)
	if got.Title != in.Title || got.Subtitle != in.Subtitle || got.Schedule != in.Schedule || got.Tickets != in.Tickets || got.URLnames != in.URLnames || client.Calls() != 0 {
		t.Errorf("got = %+v, calls = %d", got, client.Calls())
	}
}

func TestCacheMissTranslatesAndStoresAllFields(t *testing.T) {
	repo, client := &fakeRepo{}, &translate_client.FakeClient{}
	svc := NewTranslationService(repo, client)
	in := fullEvent(1)
	got := svc.TranslateEvent(context.Background(), "es", in)
	if got.Title != "[es] Títol" ||
		got.Subtitle != "[es] Subtítol" ||
		got.Description != "[es] Descripció" ||
		got.Schedule != "[es] De dimarts a diumenge, de 10 a 19 h" ||
		got.Tickets != "[es] Entrada gratuïta amb reserva prèvia" ||
		got.URLnames != "[es] Web oficial,[es] Compra d'entrades" {
		t.Errorf("unexpected translated event: %+v", got)
	}
	if client.Calls() != 1 || client.LastBatch() != 7 || repo.saved != 1 {
		t.Errorf("calls = %d, lastBatch = %d, saved = %d", client.Calls(), client.LastBatch(), repo.saved)
	}

	// The second request is served from the cache with every field intact
	got = svc.TranslateEvent(context.Background(), "es", in)
	if got.Title != "[es] Títol" ||
		got.Subtitle != "[es] Subtítol" ||
		got.Description != "[es] Descripció" ||
		got.Schedule != "[es] De dimarts a diumenge, de 10 a 19 h" ||
		got.Tickets != "[es] Entrada gratuïta amb reserva prèvia" ||
		got.URLnames != "[es] Web oficial,[es] Compra d'entrades" ||
		client.Calls() != 1 {
		t.Errorf("cache hit: got = %+v, calls = %d", got, client.Calls())
	}
}

func TestChangedEventFieldInvalidatesCache(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*event_model.Event)
		check  func(event_model.Event) bool
	}{
		{"title", func(e *event_model.Event) { e.Title = "Títol nou" }, func(e event_model.Event) bool { return e.Title == "[en] Títol nou" }},
		{"subtitle", func(e *event_model.Event) { e.Subtitle = "Subtítol nou" }, func(e event_model.Event) bool { return e.Subtitle == "[en] Subtítol nou" }},
		{"description", func(e *event_model.Event) { e.Description = "Descripció nova" }, func(e event_model.Event) bool { return e.Description == "[en] Descripció nova" }},
		{"schedule", func(e *event_model.Event) { e.Schedule = "Dissabtes a les 18 h" }, func(e event_model.Event) bool { return e.Schedule == "[en] Dissabtes a les 18 h" }},
		{"tickets", func(e *event_model.Event) { e.Tickets = "10 euros" }, func(e event_model.Event) bool { return e.Tickets == "[en] 10 euros" }},
		{"url-names", func(e *event_model.Event) { e.URLnames = "Més informació" }, func(e event_model.Event) bool { return e.URLnames == "[en] Més informació" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, client := &fakeRepo{}, &translate_client.FakeClient{}
			svc := NewTranslationService(repo, client)
			base := fullEvent(1)
			svc.TranslateEvent(context.Background(), "en", base)

			updated := base
			tt.mutate(&updated)
			got := svc.TranslateEvent(context.Background(), "en", updated)
			if !tt.check(got) || client.Calls() != 2 {
				t.Errorf("got = %+v, calls = %d, want 2", got, client.Calls())
			}
		})
	}
}

func TestURLNamesPreservesEmptySlotsAndTrimsParts(t *testing.T) {
	repo, client := &fakeRepo{}, &translate_client.FakeClient{}
	svc := NewTranslationService(repo, client)
	in := event_model.Event{
		ID:       1,
		Title:    "Festa",
		URLnames: ", Web oficial , Més informació",
	}
	got := svc.TranslateEvent(context.Background(), "es", in)
	if got.URLnames != ",[es] Web oficial,[es] Més informació" {
		t.Errorf("url-names = %q, want %q", got.URLnames, ",[es] Web oficial,[es] Més informació")
	}
	if client.LastBatch() != 3 {
		t.Errorf("lastBatch = %d, want 3", client.LastBatch())
	}
}

func TestDeduplicatesIdenticalTextsInBatch(t *testing.T) {
	repo, client := &fakeRepo{}, &translate_client.FakeClient{}
	svc := NewTranslationService(repo, client)
	in := []event_model.Event{
		{ID: 1, Title: "Concert", Tickets: "Entrada gratuïta", URLnames: "Més informació"},
		{ID: 2, Title: "Exposició", Tickets: "Entrada gratuïta", URLnames: "Més informació,Web"},
	}
	got := svc.TranslateEvents(context.Background(), "es", in)
	// Unique texts: Concert, Entrada gratuïta, Més informació, Exposició, Web = 5 (not 7)
	if client.Calls() != 1 || client.LastBatch() != 5 {
		t.Errorf("calls = %d, lastBatch = %d, want 1 and 5", client.Calls(), client.LastBatch())
	}
	if got[0].Tickets != "[es] Entrada gratuïta" || got[1].Tickets != "[es] Entrada gratuïta" {
		t.Errorf("tickets = %q / %q", got[0].Tickets, got[1].Tickets)
	}
	if got[0].URLnames != "[es] Més informació" || got[1].URLnames != "[es] Més informació,[es] Web" {
		t.Errorf("url-names = %q / %q", got[0].URLnames, got[1].URLnames)
	}
}

func TestTranslateEventsMixesHitsAndMisses(t *testing.T) {
	cachedEv := event(1, "A", "B")
	repo := &fakeRepo{data: map[int]translation_repository.Translation{
		1: {Title: "cached", Description: "cached d", SourceHash: sourceHash(&cachedEv)},
	}}
	client := &translate_client.FakeClient{}
	svc := NewTranslationService(repo, client)
	in := []event_model.Event{cachedEv, event(2, "C", "")}
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
	in := []event_model.MapEvent{{Event: fullEvent(1), HasFriends: true}}
	got := svc.TranslateMapEvents(context.Background(), "es", in)
	if got[0].Title != "[es] Títol" || got[0].Subtitle != "[es] Subtítol" || got[0].Schedule != "[es] De dimarts a diumenge, de 10 a 19 h" || !got[0].HasFriends {
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
			in := fullEvent(1)
			got := NewTranslationService(tt.repo, tt.client).TranslateEvent(context.Background(), "es", in)
			if got.Title != in.Title || got.Subtitle != in.Subtitle || got.Description != in.Description || got.Schedule != in.Schedule || got.Tickets != in.Tickets || got.URLnames != in.URLnames || tt.repo.saved != 0 {
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
