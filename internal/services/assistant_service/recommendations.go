package assistant_service

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"api-aprop/internal/models/chatbot_model"
	"api-aprop/internal/models/event_model"
)

const (
	// maxRecommendations caps each recommendation batch to 3 events.
	maxRecommendations = 3
	// maxTrackedShownIDs bounds how many previously recommended events we exclude from future
	// searches so the database query limit stays bounded even in very long conversations.
	// TODO(DP-045): revisit once conversations can be reset or SQL supports excluding IDs directly.
	maxTrackedShownIDs = 200
	// minTitleMatchRunes prevents tiny words from accidentally matching unrelated prose during fallback.
	minTitleMatchRunes = 4
)

var (
	// Matches [RECOMMENDED_IDS: 12, 34], [RECOMMENDED_IDS: [12, 34]], [RECOMMENDED_IDS: none],
	// optionally wrapped in markdown backticks/asterisks or followed by a period.
	recommendedIDsBracketPattern = regexp.MustCompile("(?im)(?:^[ \\t]*[`*]*|`)?\\[\\s*RECOMMENDED_IDS\\s*:\\s*(\\[[^\\]]*\\]|[^\\]]*)\\s*\\][`*]*\\.?")
	// Fallback for a standalone line without brackets, e.g. "RECOMMENDED_IDS: 12, 34".
	recommendedIDsLinePattern = regexp.MustCompile("(?im)^[ \\t]*[`*\\[]*RECOMMENDED_IDS\\s*:[`*]*[ \\t]*([^\\r\\n]*)$")
	intTokenPattern           = regexp.MustCompile(`\d+`)
	shownEventsPattern        = regexp.MustCompile(`(?im)^[ \t]*\[Shown events:[^\r\n]*$`)
	excessNewlinesPattern     = regexp.MustCompile(`\n{3,}`)
)

// searchResult holds one search_events tool call's input and the unseen events returned to the model.
type searchResult struct {
	input  SearchEventsInput
	events []event_model.Event
}

// turnSearches accumulates all unseen catalogue events seen during a single assistant reply turn
// so the model can choose across multiple search_events calls without earlier results being lost.
type turnSearches struct {
	last          *searchResult
	candidates    []event_model.Event
	candidatesMap map[int]event_model.Event
	eventInput    map[int]SearchEventsInput
}

func newTurnSearches() *turnSearches {
	return &turnSearches{
		candidatesMap: make(map[int]event_model.Event),
		eventInput:    make(map[int]SearchEventsInput),
	}
}

func (t *turnSearches) record(res *searchResult) {
	if res == nil {
		return
	}
	t.last = res
	for _, e := range res.events {
		if _, exists := t.candidatesMap[e.ID]; exists {
			continue
		}
		t.candidatesMap[e.ID] = e
		t.candidates = append(t.candidates, e)
		t.eventInput[e.ID] = res.input
	}
}

// collectShownEventIDs gathers event IDs already recommended in previous turns of the conversation,
// walking from newest to oldest up to maxTrackedShownIDs.
func collectShownEventIDs(history []chatbot_model.Message) map[int]struct{} {
	shown := make(map[int]struct{})
	for i := len(history) - 1; i >= 0 && len(shown) < maxTrackedShownIDs; i-- {
		for _, e := range history[i].RecommendedEvents {
			shown[e.ID] = struct{}{}
			if len(shown) >= maxTrackedShownIDs {
				break
			}
		}
	}
	return shown
}

// parseRecommendedIDs extracts and removes the structured [RECOMMENDED_IDS: ...] trailer
// from the model's reply. When the tag is present, hasTag is true even if it lists no IDs
// (e.g. [RECOMMENDED_IDS: none]).
// TODO(DP-045): confirm whether to keep the trailer tag or move to a structured JSON response schema.
func parseRecommendedIDs(text string) (cleanText string, ids []int, hasTag bool) {
	matches := recommendedIDsBracketPattern.FindAllStringSubmatch(text, -1)
	if len(matches) > 0 {
		hasTag = true
		ids = extractIntIDs(matches[len(matches)-1][1])
		cleanText = recommendedIDsBracketPattern.ReplaceAllString(text, "")
	} else if lineMatches := recommendedIDsLinePattern.FindAllStringSubmatch(text, -1); len(lineMatches) > 0 {
		hasTag = true
		ids = extractIntIDs(lineMatches[len(lineMatches)-1][1])
		cleanText = recommendedIDsLinePattern.ReplaceAllString(text, "")
	} else {
		cleanText = text
	}

	cleanText = shownEventsPattern.ReplaceAllString(cleanText, "")
	cleanText = strings.ReplaceAll(cleanText, "\r\n", "\n")
	cleanText = excessNewlinesPattern.ReplaceAllString(cleanText, "\n\n")
	cleanText = strings.TrimSpace(cleanText)
	return cleanText, ids, hasTag
}

func extractIntIDs(raw string) []int {
	tokens := intTokenPattern.FindAllString(raw, -1)
	if len(tokens) == 0 {
		return nil
	}
	ids := make([]int, 0, len(tokens))
	for _, tok := range tokens {
		id, err := strconv.Atoi(tok)
		if err == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

// resolveRecommendedIDs determines the exact, ordered event IDs (at most maxRecommendations) to
// attach as cards under the assistant message, guaranteeing cards come only from unseen events
// returned by search_events in this turn and match what the model selected.
func resolveRecommendedIDs(
	cleanText string,
	hasTag bool,
	taggedIDs []int,
	searches *turnSearches,
	shownIDs map[int]struct{},
) []int {
	if searches == nil || len(searches.candidates) == 0 {
		return nil
	}

	if hasTag {
		if len(taggedIDs) == 0 {
			// Model explicitly signalled no recommendations (e.g. [RECOMMENDED_IDS: none]).
			return nil
		}
		chosen := filterValidChosenIDs(taggedIDs, searches.candidatesMap, shownIDs, maxRecommendations)
		if len(chosen) > 0 {
			return chosen
		}
		// Every tagged ID was invalid or already shown: rescue real events named in the text if any.
		return matchTitlesInText(cleanText, searches.candidates, maxRecommendations)
	}

	// Fallback when the model omitted the tag: prefer exact candidate titles mentioned in the text,
	// in the order they appear.
	if matched := matchTitlesInText(cleanText, searches.candidates, maxRecommendations); len(matched) > 0 {
		return matched
	}

	// Final fallback for brief generic intros or unit-test stubs that do not repeat event titles:
	// cap the latest search's unseen results to maxRecommendations.
	if searches.last == nil || len(searches.last.events) == 0 {
		return nil
	}
	limit := min(len(searches.last.events), maxRecommendations)
	ids := make([]int, 0, limit)
	for _, e := range searches.last.events[:limit] {
		if _, shown := shownIDs[e.ID]; shown {
			continue
		}
		ids = append(ids, e.ID)
	}
	return ids
}

func filterValidChosenIDs(
	ids []int,
	candidatesMap map[int]event_model.Event,
	shownIDs map[int]struct{},
	limit int,
) []int {
	chosen := make([]int, 0, min(len(ids), limit))
	seen := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			continue
		}
		if _, ok := candidatesMap[id]; !ok {
			continue
		}
		if _, shown := shownIDs[id]; shown {
			continue
		}
		seen[id] = struct{}{}
		chosen = append(chosen, id)
		if len(chosen) == limit {
			break
		}
	}
	return chosen
}

// interpretationFor chooses the search criteria to display as chips under the reply.
// When all chosen events came from one search call, that call's criteria are used; otherwise
// the latest search call's criteria are used.
func interpretationFor(searches *turnSearches, chosenIDs []int) *chatbot_model.Interpretation {
	if searches == nil || searches.last == nil {
		return nil
	}
	if len(chosenIDs) > 0 {
		first, ok := searches.eventInput[chosenIDs[0]]
		if ok {
			same := true
			for _, id := range chosenIDs[1:] {
				if in, exists := searches.eventInput[id]; !exists || in != first {
					same = false
					break
				}
			}
			if same || len(searches.last.events) == 0 {
				return toInterpretation(first)
			}
		}
	}
	return toInterpretation(searches.last.input)
}

type titleMatch struct {
	start   int
	end     int
	eventID int
}

// matchTitlesInText finds candidate events whose title appears in text (case- and accent-insensitive),
// returning up to limit event IDs ordered by their first appearance in text.
func matchTitlesInText(text string, candidates []event_model.Event, limit int) []int {
	normText := normalizeForMatch(text)
	if normText == "" {
		return nil
	}

	var matches []titleMatch
	for _, c := range candidates {
		normTitle := normalizeForMatch(c.Title)
		if len([]rune(normTitle)) < minTitleMatchRunes {
			continue
		}
		if idx := indexWithWordBoundary(normText, normTitle); idx >= 0 {
			matches = append(matches, titleMatch{
				start:   idx,
				end:     idx + len(normTitle),
				eventID: c.ID,
			})
		}
	}
	if len(matches) == 0 {
		return nil
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].start != matches[j].start {
			return matches[i].start < matches[j].start
		}
		return (matches[i].end - matches[i].start) > (matches[j].end - matches[j].start)
	})

	ids := make([]int, 0, min(len(matches), limit))
	var accepted []titleMatch
	for _, m := range matches {
		overlaps := false
		for _, prev := range accepted {
			if m.start < prev.end && prev.start < m.end {
				overlaps = true
				break
			}
		}
		if overlaps {
			continue
		}
		accepted = append(accepted, m)
		ids = append(ids, m.eventID)
		if len(ids) == limit {
			break
		}
	}
	return ids
}

func indexWithWordBoundary(haystack, needle string) int {
	offset := 0
	for offset <= len(haystack)-len(needle) {
		rel := strings.Index(haystack[offset:], needle)
		if rel == -1 {
			return -1
		}
		start := offset + rel
		end := start + len(needle)
		leftOK := start == 0 || !isWordChar(lastRune(haystack[:start]))
		rightOK := end == len(haystack) || !isWordChar(firstRune(haystack[end:]))
		if leftOK && rightOK {
			return start
		}
		offset = start + 1
	}
	return -1
}

func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func firstRune(s string) rune {
	for _, r := range s {
		return r
	}
	return 0
}

func lastRune(s string) rune {
	var last rune
	for _, r := range s {
		last = r
	}
	return last
}

// normalizeForMatch lowercases, strips common Catalan/Spanish/Latin accents and collapses spaces.
func normalizeForMatch(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := true
	for _, r := range strings.ToLower(s) {
		r = foldAccent(r)
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}
	return strings.TrimSpace(b.String())
}

func foldAccent(r rune) rune {
	switch r {
	case 'à', 'á', 'ä', 'â', 'ã', 'å':
		return 'a'
	case 'è', 'é', 'ë', 'ê':
		return 'e'
	case 'ì', 'í', 'ï', 'î':
		return 'i'
	case 'ò', 'ó', 'ö', 'ô', 'õ':
		return 'o'
	case 'ù', 'ú', 'ü', 'û':
		return 'u'
	case 'ç':
		return 'c'
	case 'ñ':
		return 'n'
	case '·':
		return ' '
	default:
		return r
	}
}

const clarifyFirstResult = `{"error":"The request is too vague to search. Do NOT recommend events yet: ask the user 1-2 short questions (type of plan or interests, and city/zone), then end with [RECOMMENDED_IDS: none].","is_error":true}`

// needsClarification reports whether a search_events call is a fresh, unfiltered request (no category
// and no municipality) that should be answered with clarifying questions first. It is skipped when the user
// is answering a question from the assistant or asking for more of an earlier recommendation.
// shortcut: relies on the last assistant message ending with "?"; upgrade to an explicit conversation state.
func needsClarification(history []chatbot_model.Message, arguments string) bool {
	var in SearchEventsInput
	if err := unmarshalArgs(arguments, &in); err != nil {
		return false
	}
	if strings.TrimSpace(in.Category) != "" || strings.TrimSpace(in.Municipality) != "" {
		return false
	}
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Sender != chatbot_model.SenderAssistant {
			continue
		}
		last := history[i]
		return len(last.RecommendedEvents) == 0 && !strings.HasSuffix(strings.TrimSpace(last.Text), "?")
	}
	return true
}
