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
	"api-aprop/internal/models/preference_model"
	"api-aprop/internal/repositories/translation_repository"
)

const (
	// maxMisses bounds how many events are translated per request; the rest keep the original text and are cached
	// by later requests.
	maxMisses = 20

	// maxQuestionMisses covers a full interest test (25 questions, only title and description per event).
	maxQuestionMisses = 25

	// batchSize splits uncached events into small translation calls so each batch finishes quickly and caches
	// independently even if another batch fails or times out (kept at 3 now that up to 6 fields per event are translated).
	batchSize = 3

	// questionBatchSize batches uncached interest-test questions (at most 2 texts per question).
	questionBatchSize = 5

	// maxConcurrentBatches limits how many batches run in parallel for a single request.
	maxConcurrentBatches = 4
)

// translateTimeout bounds the total translation round trip of a request (a variable so tests can shorten it).
// Kept below the HTTP server WriteTimeout (15s) so responses are never aborted mid-flight.
var translateTimeout = 12 * time.Second

// TODO(DP-043): translates title, subtitle, description, schedule, tickets and url-names with no cache expiry; see the DP for the assumptions.
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

// TranslateTestQuestions translates the title and description of interest-test questions into lang,
// reusing cached EventTranslations rows without overwriting fuller 6-field event translations.
// TODO(DP-047): cached rows are reused whenever translated title/description are present without verifying
// the full 6-field SourceHash (TestQuestion only carries title and description); misses cache a 2-field hash
// that GET /events/{id} later upgrades if the event has additional fields.
func (s *TranslationService) TranslateTestQuestions(ctx context.Context, lang string, questions []preference_model.TestQuestion) []preference_model.TestQuestion {
	if !needsTranslation(lang) || len(questions) == 0 {
		return questions
	}
	out := make([]preference_model.TestQuestion, len(questions))
	copy(out, questions)

	ids := make([]int, len(out))
	for i, q := range out {
		ids[i] = q.EventID
	}
	cached := s.loadCachedTranslations(ctx, lang, ids, "could not read the cached translations for test questions")

	var misses []*preference_model.TestQuestion
	for i := range out {
		q := &out[i]
		if strings.TrimSpace(q.Title) == "" && strings.TrimSpace(q.Description) == "" {
			continue
		}
		if t, ok := cached[q.EventID]; ok && hasQuestionTranslation(q, t) {
			if strings.TrimSpace(q.Title) != "" {
				q.Title = t.Title
			}
			if strings.TrimSpace(q.Description) != "" {
				q.Description = t.Description
			}
			continue
		}
		misses = append(misses, q)
	}

	if len(misses) > maxQuestionMisses {
		misses = misses[:maxQuestionMisses]
	}
	if len(misses) > 0 {
		runTranslationBatches(ctx, len(misses), questionBatchSize, func(tctx context.Context, start, end int) {
			s.translateQuestionBatch(tctx, ctx, lang, misses[start:end])
		})
	}
	return out
}

func hasQuestionTranslation(q *preference_model.TestQuestion, t translation_repository.Translation) bool {
	if strings.TrimSpace(q.Title) != "" && strings.TrimSpace(t.Title) == "" {
		return false
	}
	if strings.TrimSpace(q.Description) != "" && strings.TrimSpace(t.Description) == "" {
		return false
	}
	return true
}

func (s *TranslationService) translateQuestionBatch(tctx, saveCtx context.Context, lang string, batch []*preference_model.TestQuestion) {
	texts := newTextBatch()
	for _, q := range batch {
		texts.add(q.Title)
		texts.add(q.Description)
	}
	lookup := s.translateTexts(tctx, lang, texts, "test questions", "questions", len(batch))
	if lookup == nil {
		return
	}

	for _, q := range batch {
		hash := questionSourceHash(q.Title, q.Description)
		title, okTitle := lookup(q.Title)
		desc, okDesc := lookup(q.Description)
		q.Title = title
		q.Description = desc
		if !okTitle || !okDesc {
			slog.Warn("upstream translator returned blank text for test question; skipping cache", "event", q.EventID, "lang", lang)
			continue
		}
		if err := s.repository.SaveTranslation(saveCtx, lang, q.EventID, translation_repository.Translation{
			Title:       q.Title,
			Description: q.Description,
			SourceHash:  hash,
		}); err != nil {
			slog.Warn("could not cache the translation of the test question", "event", q.EventID, "lang", lang, "error", err)
		}
	}
}

func needsTranslation(lang string) bool {
	return lang != language_model.Default && language_model.Valid(lang)
}

func (s *TranslationService) loadCachedTranslations(
	ctx context.Context,
	lang string,
	ids []int,
	warnMsg string,
) map[int]translation_repository.Translation {
	cached, err := s.repository.GetTranslations(ctx, lang, ids)
	if err != nil {
		slog.Warn(warnMsg, "lang", lang, "error", err)
		return nil
	}
	return cached
}

// runTranslationBatches splits total uncached items into bounded concurrent batches under a shared timeout.
func runTranslationBatches(ctx context.Context, total, size int, fn func(tctx context.Context, start, end int)) {
	tctx, cancel := context.WithTimeout(ctx, translateTimeout)
	defer cancel()

	if total <= size {
		fn(tctx, 0, total)
		return
	}

	sem := make(chan struct{}, maxConcurrentBatches)
	var wg sync.WaitGroup
	for start := 0; start < total; start += size {
		end := min(start+size, total)
		wg.Add(1)
		sem <- struct{}{}
		go func(start, end int) {
			defer wg.Done()
			defer func() { <-sem }()
			fn(tctx, start, end)
		}(start, end)
	}
	wg.Wait()
}

type textBatch struct {
	texts []string
	index map[string]int
}

func newTextBatch() *textBatch {
	return &textBatch{index: make(map[string]int)}
}

func (b *textBatch) add(raw string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return
	}
	if _, ok := b.index[trimmed]; ok {
		return
	}
	b.index[trimmed] = len(b.texts)
	b.texts = append(b.texts, trimmed)
}

// translateTexts calls the upstream translator for the unique texts in b and returns a lookup
// function mapping each raw input to its translation (reporting ok=false if the upstream returned
// blank text for a non-empty input). Returns nil when nothing needs translation or the call fails.
func (s *TranslationService) translateTexts(
	tctx context.Context,
	lang string,
	b *textBatch,
	label string,
	countKey string,
	count int,
) func(string) (string, bool) {
	if len(b.texts) == 0 {
		return nil
	}
	out, err := s.client.Translate(tctx, b.texts, language_model.Default, lang)
	if err == nil && len(out) != len(b.texts) {
		err = fmt.Errorf("expected %d translations, got %d", len(b.texts), len(out))
	}
	if err != nil {
		if !errors.Is(err, translate_client.ErrDisabled) {
			slog.Warn("could not translate "+label+"; using the original text", countKey, count, "lang", lang, "error", err)
		}
		return nil
	}
	return func(raw string) (string, bool) {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return raw, true
		}
		translated := out[b.index[trimmed]]
		if strings.TrimSpace(translated) == "" {
			return raw, false
		}
		return translated, true
	}
}

// translate replaces the free-text fields of the events with their translation into lang: from the cache when
// the Catalan texts have not changed, otherwise asking the translation server and caching the result.
// TODO(DP-049): preserves valid 2-field cached translations (from TranslateTestQuestions) as fallback if the
// 6-field upgrade is capped by maxMisses or fails/times out, and rejects blank upstream translations so cache
// rows are never poisoned; per-request caps, timeouts and upstream NMT differences remain design limitations.
func (s *TranslationService) translate(ctx context.Context, lang string, events []*event_model.Event) {
	if !needsTranslation(lang) || len(events) == 0 {
		return
	}

	ids := make([]int, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}
	cached := s.loadCachedTranslations(ctx, lang, ids, "could not read the cached translations")

	var misses []*event_model.Event
	partialCached := make(map[int]translation_repository.Translation)
	for _, e := range events {
		hash := sourceHash(e)
		if t, ok := cached[e.ID]; ok {
			if t.SourceHash == hash {
				e.Title = t.Title
				e.Subtitle = t.Subtitle
				e.Description = t.Description
				e.Schedule = t.Schedule
				e.Tickets = t.Tickets
				e.URLnames = t.URLNames
				continue
			}
			if t.SourceHash == questionSourceHash(e.Title, e.Description) {
				partialCached[e.ID] = t
			}
		}
		misses = append(misses, e)
	}

	toTranslate := misses
	if len(toTranslate) > maxMisses {
		toTranslate = toTranslate[:maxMisses]
	}
	upgraded := make([]bool, len(toTranslate))
	if len(toTranslate) > 0 {
		runTranslationBatches(ctx, len(toTranslate), batchSize, func(tctx context.Context, start, end int) {
			s.translateBatch(tctx, ctx, lang, toTranslate[start:end], upgraded[start:end])
		})
	}
	if len(partialCached) > 0 {
		for i, e := range misses {
			if i < len(upgraded) && upgraded[i] {
				continue
			}
			if t, ok := partialCached[e.ID]; ok {
				if strings.TrimSpace(e.Title) != "" && strings.TrimSpace(t.Title) != "" {
					e.Title = t.Title
				}
				if strings.TrimSpace(e.Description) != "" && strings.TrimSpace(t.Description) != "" {
					e.Description = t.Description
				}
			}
		}
	}
}

func (s *TranslationService) translateBatch(tctx, saveCtx context.Context, lang string, batch []*event_model.Event, upgraded []bool) {
	lookup := s.translateTexts(tctx, lang, collectTexts(batch), "the events", "events", len(batch))
	if lookup == nil {
		return
	}
	for i, e := range batch {
		hash := sourceHash(e)
		complete := true
		var ok bool
		for _, field := range []*string{&e.Title, &e.Subtitle, &e.Description, &e.Schedule, &e.Tickets} {
			if *field, ok = lookup(*field); !ok {
				complete = false
			}
		}
		if strings.TrimSpace(e.URLnames) != "" {
			parts := strings.Split(e.URLnames, ",")
			for j, part := range parts {
				if parts[j], ok = lookup(part); !ok {
					complete = false
				}
			}
			e.URLnames = strings.Join(parts, ",")
		}
		if !complete {
			slog.Warn("upstream translator returned blank text for event; skipping cache", "event", e.ID, "lang", lang)
			continue
		}
		upgraded[i] = true
		if err := s.repository.SaveTranslation(saveCtx, lang, e.ID, translation_repository.Translation{
			Title:       e.Title,
			Subtitle:    e.Subtitle,
			Description: e.Description,
			Schedule:    e.Schedule,
			Tickets:     e.Tickets,
			URLNames:    e.URLnames,
			SourceHash:  hash,
		}); err != nil {
			slog.Warn("could not cache the translation of the event", "event", e.ID, "lang", lang, "error", err)
		}
	}
}

// collectTexts gathers the unique non-empty trimmed texts to translate in a batch.
// URLnames is split by commas so each link label is translated independently without losing the comma alignment
// with related-urls.
func collectTexts(batch []*event_model.Event) *textBatch {
	b := newTextBatch()
	for _, e := range batch {
		for _, field := range []string{e.Title, e.Subtitle, e.Description, e.Schedule, e.Tickets} {
			b.add(field)
		}
		if strings.TrimSpace(e.URLnames) != "" {
			for _, part := range strings.Split(e.URLnames, ",") {
				b.add(part)
			}
		}
	}
	return b
}

// questionSourceHash identifies the 2 fields translated by TranslateTestQuestions.
func questionSourceHash(title, description string) string {
	return sourceHash(&event_model.Event{Title: title, Description: description})
}

// sourceHash identifies the original translatable texts of an event; NUL separators avoid collisions across fields.
func sourceHash(e *event_model.Event) string {
	h := sha256.New()
	fields := []string{e.Title, e.Subtitle, e.Description, e.Schedule, e.Tickets, e.URLnames}
	for i, f := range fields {
		if i > 0 {
			h.Write([]byte{0})
		}
		h.Write([]byte(f))
	}
	return hex.EncodeToString(h.Sum(nil))
}
