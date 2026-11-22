package assistant_service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"api-aprop/internal/clients/groq_client"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/repositories/catalogue_repository"
)

const (
	ToolSearchEvents = "search_events"
	ToolGetEvent     = "get_event"

	DateLayout = "2006-01-02"

	defaultLimit       = 6
	maxLimit           = 10
	maxCatalogueFetch  = 250
	maxDescriptionRune = 300
)

// SearchEventsInput are the arguments of the search_events tool.
type SearchEventsInput struct {
	Category     string `json:"category,omitempty"`
	Municipality string `json:"municipality,omitempty"`
	From         string `json:"from,omitempty"`
	To           string `json:"to,omitempty"`
	Limit        int    `json:"limit,omitempty"`
}

// GetEventInput are the arguments of the get_event tool.
type GetEventInput struct {
	ID int `json:"id"`
}

// EventSummary is the compact view of an event given to the model.
type EventSummary struct {
	ID           int    `json:"id"`
	Title        string `json:"title"`
	Municipality string `json:"municipality,omitempty"`
	Place        string `json:"place,omitempty"`
	StartDate    string `json:"start-date,omitempty"`
	EndDate      string `json:"end-date,omitempty"`
	Schedule     string `json:"schedule,omitempty"`
	Free         bool   `json:"free"`
	Description  string `json:"description,omitempty"`
}

// toolError is a failure the model can read and fix, such as a malformed date.
type toolError struct{ msg string }

func (e toolError) Error() string { return e.msg }

func toolDefinitions() []groq_client.Tool {
	return []groq_client.Tool{
		{Type: "function", Function: groq_client.ToolDefinition{
			Name:        ToolSearchEvents,
			Description: "Searches the real catalogue of cultural events. Only returns events that have not ended before the start of the range, and automatically excludes events already shown in this conversation.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"category":     map[string]any{"type": "string", "description": "type or category of activity in Catalan, e.g. concert, teatre, dansa, exposició, cinema"},
					"municipality": map[string]any{"type": "string", "description": "municipality of Catalonia, e.g. Girona"},
					"from":         map[string]any{"type": "string", "description": "first day of the date range as YYYY-MM-DD; defaults to today"},
					"to":           map[string]any{"type": "string", "description": "last day of the date range as YYYY-MM-DD"},
					"limit":        map[string]any{"type": "integer", "description": "maximum number of unseen candidate events to return, from 1 to 10; defaults to 6 (recommend at most 3 of them)"},
				},
			},
		}},
		{Type: "function", Function: groq_client.ToolDefinition{
			Name:        ToolGetEvent,
			Description: "Returns an event of the catalogue by id.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{"id": map[string]any{"type": "integer", "description": "id of the event, as returned by search_events"}},
				"required":   []string{"id"},
			},
		}},
		{Type: "function", Function: groq_client.ToolDefinition{
			Name:        ToolGetAppHelp,
			Description: "Returns the user guide of the Aprop app for a topic: use it for any question about how to use the app or where to find something. Answer only from its result.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{"topic": map[string]any{"type": "string", "description": "topic of the guide", "enum": topicKeys()}},
				"required":   []string{"topic"},
			},
		}},
	}
}

func topicKeys() []string {
	_, keys := helpTopics()
	return keys
}

// callTool runs a tool and returns the JSON text for the model. A toolError is returned as an error
// result for the model; any other error aborts the reply. search is set for a successful search_events call.
func (s *AssistantService) callTool(ctx context.Context, userID string, now time.Time, name string, arguments string) (text string, search *searchResult, err error) {
	return s.callToolWithExcluded(ctx, userID, now, nil, name, arguments)
}

func (s *AssistantService) callToolWithExcluded(ctx context.Context, userID string, now time.Time, excludedIDs map[int]struct{}, name string, arguments string) (text string, search *searchResult, err error) {
	text, search, err = s.runTool(ctx, userID, now, excludedIDs, name, arguments)
	var te toolError
	if errors.As(err, &te) {
		b, _ := json.Marshal(map[string]any{"error": te.msg, "is_error": true})
		return string(b), nil, nil
	}
	return text, search, err
}

func (s *AssistantService) runTool(ctx context.Context, userID string, now time.Time, excludedIDs map[int]struct{}, name string, arguments string) (string, *searchResult, error) {
	switch name {
	case ToolSearchEvents:
		var in SearchEventsInput
		if err := unmarshalArgs(arguments, &in); err != nil {
			return "", nil, err
		}
		filter, err := toSearchFilter(in, now)
		if err != nil {
			return "", nil, toolError{err.Error()}
		}
		filter.UserID = userID
		desired := filter.Limit
		if len(excludedIDs) > 0 {
			// TODO(DP-046): fetch extra rows to filter already-shown IDs in Go without modifying sqlc queries.
			filter.Limit = min(desired+len(excludedIDs), maxCatalogueFetch)
		}
		events, err := s.catalogueRepository.Search(ctx, filter)
		if err != nil {
			return "", nil, toolError{"the catalogue is not available"}
		}
		unseen := make([]event_model.Event, 0, min(desired, len(events)))
		skipped := 0
		for _, e := range events {
			if _, shown := excludedIDs[e.ID]; shown {
				skipped++
				continue
			}
			unseen = append(unseen, e)
			if len(unseen) == desired {
				break
			}
		}
		out := struct {
			Events              []EventSummary `json:"events"`
			AlreadyShownSkipped int            `json:"already-shown-skipped,omitempty"`
		}{
			Events:              make([]EventSummary, 0, len(unseen)),
			AlreadyShownSkipped: skipped,
		}
		for _, e := range unseen {
			out.Events = append(out.Events, toSummary(e))
		}
		b, _ := json.Marshal(out)
		return string(b), &searchResult{input: in, events: unseen}, nil
	case ToolGetAppHelp:
		text, err := s.getAppHelp(ctx, userID, arguments)
		return text, nil, err
	case ToolGetEvent:
		var in GetEventInput
		if err := unmarshalArgs(arguments, &in); err != nil {
			return "", nil, err
		}
		events, err := s.catalogueRepository.ListByIDs(ctx, []int{in.ID})
		if err != nil {
			return "", nil, toolError{"the catalogue is not available"}
		}
		if len(events) == 0 {
			return "", nil, toolError{fmt.Sprintf("event %d not found", in.ID)}
		}
		b, _ := json.Marshal(toSummary(events[0]))
		return string(b), nil, nil
	}
	return "", nil, toolError{"unknown tool " + name}
}

func unmarshalArgs(arguments string, v any) error {
	if strings.TrimSpace(arguments) == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(arguments), v); err != nil {
		return toolError{"invalid arguments: " + err.Error()}
	}
	return nil
}

func toSearchFilter(in SearchEventsInput, now time.Time) (catalogue_repository.SearchFilter, error) {
	filter := catalogue_repository.SearchFilter{
		Category:     strings.TrimSpace(in.Category),
		Municipality: strings.TrimSpace(in.Municipality),
		Limit:        in.Limit,
	}
	if filter.Limit <= 0 {
		filter.Limit = defaultLimit
	}
	filter.Limit = min(filter.Limit, maxLimit)

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	from := today
	if in.From != "" {
		parsed, err := time.Parse(DateLayout, in.From)
		if err != nil {
			return filter, fmt.Errorf("invalid from date %q, expected YYYY-MM-DD", in.From)
		}
		from = parsed
	}
	filter.From = &from
	if in.To != "" {
		to, err := time.Parse(DateLayout, in.To)
		if err != nil {
			return filter, fmt.Errorf("invalid to date %q, expected YYYY-MM-DD", in.To)
		}
		if to.Before(from) {
			return filter, errors.New("the to date must not be before the from date")
		}
		filter.To = &to
	}
	return filter, nil
}

func toSummary(e event_model.Event) EventSummary {
	return EventSummary{
		ID:           e.ID,
		Title:        e.Title,
		Municipality: e.Municipality,
		Place:        e.Place,
		StartDate:    formatDate(e.StartDate),
		EndDate:      formatDate(e.EndDate),
		Schedule:     e.Schedule,
		Free:         e.Free.Bool,
		Description:  truncate(e.Description, maxDescriptionRune),
	}
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(DateLayout)
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
