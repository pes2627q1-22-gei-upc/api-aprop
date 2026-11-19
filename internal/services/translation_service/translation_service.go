package translation_service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"api-aprop/internal/clients/translate_client"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/language_model"
	"api-aprop/internal/repositories/translation_repository"
)

const (
	// maxMisses bounds how many events are translated per request; the rest keep the original text and are cached
	// by later requests.
	maxMisses = 20

	// batchSize splits uncached events into small translation calls so each batch finishes quickly and caches
	// independently even if another batch fails or times out.
	batchSize = 5

	// maxConcurrentBatches limits how many batches run in parallel for a single request.
	maxConcurrentBatches = 4
)

// translateTimeout bounds the total translation round trip of a request (a variable so tests can shorten it).
// Kept below the HTTP server WriteTimeout (15s) so responses are never aborted mid-flight.
var translateTimeout = 12 * time.Second

// TODO(DP-043): only title and description are translated, with no cache expiry; see the DP for the assumptions.
type TranslationService struct {
	repository translation_repository.Repository
	client     translate_client.Client
}

var _ Service = (*TranslationService)(nil)

func NewTranslationService(repository translation_repository.Repository, client translate_client.Client) *TranslationService {
	return &TranslationService{repository: repository, client: client}
}

func (s *TranslationService) UserLanguage(ctx context.Context, userID string) (string, error) {
	return s.repository.GetUserLanguage(ctx, userID)
}

func (s *TranslationService) TranslateEvent(ctx context.Context, lang string, e event_model.Event) event_model.Event {
	s.translate(ctx, lang, []*event_model.Event{&e})
	return e
}

func (s *TranslationService) TranslateEvents(ctx context.Context, lang string, events []event_model.Event) []event_model.Event {
	if !needsTranslation(lang) || len(events) == 0 {
		return events
	}
	out := make([]event_model.Event, len(events))
	copy(out, events)
	ptrs := make([]*event_model.Event, len(out))
	for i := range out {
		ptrs[i] = &out[i]
	}
	s.translate(ctx, lang, ptrs)
	return out
}

func (s *TranslationService) TranslateMapEvents(ctx context.Context, lang string, events []event_model.MapEvent) []event_model.MapEvent {
	if !needsTranslation(lang) || len(events) == 0 {
		return events
	}
	out := make([]event_model.MapEvent, len(events))
	copy(out, events)
	ptrs := make([]*event_model.Event, len(out))
	for i := range out {
		ptrs[i] = &out[i].Event
	}
	s.translate(ctx, lang, ptrs)
	return out
}

func needsTranslation(lang string) bool {
	return lang != language_model.Default && language_model.Valid(lang)
}

// translate replaces the title and description of the events with their translation into lang: from the cache when
// the Catalan text has not changed, otherwise asking the translation server and caching the result.
func (s *TranslationService) translate(ctx context.Context, lang string, events []*event_model.Event) {
	if !needsTranslation(lang) || len(events) == 0 {
		return
	}

	ids := make([]int, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}
	cached, err := s.repository.GetTranslations(ctx, lang, ids)
	if err != nil {
		slog.Warn("could not read the cached translations", "lang", lang, "error", err)
		cached = nil
	}

	var misses []*event_model.Event
	for _, e := range events {
		hash := sourceHash(e.Title, e.Description)
		if t, ok := cached[e.ID]; ok && t.SourceHash == hash {
			e.Title, e.Description = t.Title, t.Description
			continue
		}
		misses = append(misses, e)
	}

	if len(misses) > maxMisses {
		misses = misses[:maxMisses]
	}
	if len(misses) > 0 {
		s.translateMisses(ctx, lang, misses)
	}
}

// translateMisses translates uncached events in small concurrent batches under a shared timeout and caches each
// batch as soon as it succeeds. On failure or timeout, only the affected batch keeps its original text.
func (s *TranslationService) translateMisses(ctx context.Context, lang string, misses []*event_model.Event) {
	tctx, cancel := context.WithTimeout(ctx, translateTimeout)
	defer cancel()

	if len(misses) <= batchSize {
		s.translateBatch(tctx, ctx, lang, misses)
		return
	}

	sem := make(chan struct{}, maxConcurrentBatches)
	var wg sync.WaitGroup
	for start := 0; start < len(misses); start += batchSize {
		end := min(start+batchSize, len(misses))
		batch := misses[start:end]
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			s.translateBatch(tctx, ctx, lang, batch)
		}()
	}
	wg.Wait()
}

func (s *TranslationService) translateBatch(tctx, saveCtx context.Context, lang string, batch []*event_model.Event) {
	var texts []string
	for _, e := range batch {
		for _, t := range []string{e.Title, e.Description} {
			if strings.TrimSpace(t) != "" {
				texts = append(texts, t)
			}
		}
	}
	if len(texts) == 0 {
		return
	}
	out, err := s.client.Translate(tctx, texts, language_model.Default, lang)
	if err == nil && len(out) != len(texts) {
		err = fmt.Errorf("expected %d translations, got %d", len(texts), len(out))
	}
	if err != nil {
		if !errors.Is(err, translate_client.ErrDisabled) {
			slog.Warn("could not translate the events; using the original text", "events", len(batch), "lang", lang, "error", err)
		}
		return
	}
	for _, e := range batch {
		hash := sourceHash(e.Title, e.Description)
		if strings.TrimSpace(e.Title) != "" {
			e.Title, out = out[0], out[1:]
		}
		if strings.TrimSpace(e.Description) != "" {
			e.Description, out = out[0], out[1:]
		}
		if err := s.repository.SaveTranslation(saveCtx, lang, e.ID, translation_repository.Translation{Title: e.Title, Description: e.Description, SourceHash: hash}); err != nil {
			slog.Warn("could not cache the translation of the event", "event", e.ID, "lang", lang, "error", err)
		}
	}
}

// sourceHash identifies the original text of an event; the separator avoids collisions between title and description.
func sourceHash(title, description string) string {
	sum := sha256.Sum256([]byte(title + "\x00" + description))
	return hex.EncodeToString(sum[:])
}
