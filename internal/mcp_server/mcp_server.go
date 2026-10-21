package mcp_server

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/repositories/catalogue_repository"
)

const (
	ToolSearchEvents = "search_events"
	ToolGetEvent     = "get_event"

	DateLayout = "2006-01-02"

	defaultLimit       = 5
	maxLimit           = 10
	maxDescriptionRune = 300
)

// SearchEventsInput are the arguments of the search_events tool.
type SearchEventsInput struct {
	Category     string `json:"category,omitempty" jsonschema:"type or category of activity in Catalan, e.g. concert, teatre, dansa, exposició, cinema"`
	Municipality string `json:"municipality,omitempty" jsonschema:"municipality of Catalonia, e.g. Girona"`
	From         string `json:"from,omitempty" jsonschema:"first day of the date range as YYYY-MM-DD; defaults to today"`
	To           string `json:"to,omitempty" jsonschema:"last day of the date range as YYYY-MM-DD"`
	Limit        int    `json:"limit,omitempty" jsonschema:"maximum number of events to return, from 1 to 10; defaults to 5"`
}

// SearchEventsOutput is the result of the search_events tool.
type SearchEventsOutput struct {
	Events []EventSummary `json:"events"`
}

// GetEventInput are the arguments of the get_event tool.
type GetEventInput struct {
	ID int `json:"id" jsonschema:"id of the event, as returned by search_events"`
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

// NewServer builds the MCP server that exposes the events catalogue as read only tools.
// now gives the current time, used to default the start of the date range to today.
func NewServer(catalogue catalogue_repository.Repository, now func() time.Time) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "aprop-catalogue", Version: "0.1.0"}, &mcp.ServerOptions{
		Instructions: "Read only access to the cultural events catalogue of Aprop (Catalonia).",
	})
	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true}

	mcp.AddTool(server, &mcp.Tool{
		Name:        ToolSearchEvents,
		Description: "Searches the real catalogue of cultural events. Only returns events that have not ended before the start of the range.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in SearchEventsInput) (*mcp.CallToolResult, SearchEventsOutput, error) {
		filter, err := toSearchFilter(in, now())
		if err != nil {
			return nil, SearchEventsOutput{}, err
		}
		events, err := catalogue.Search(ctx, filter)
		if err != nil {
			return nil, SearchEventsOutput{}, fmt.Errorf("the catalogue is not available: %w", err)
		}
		out := SearchEventsOutput{Events: make([]EventSummary, 0, len(events))}
		for _, e := range events {
			out.Events = append(out.Events, toSummary(e))
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        ToolGetEvent,
		Description: "Returns an event of the catalogue by id.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in GetEventInput) (*mcp.CallToolResult, EventSummary, error) {
		events, err := catalogue.ListByIDs(ctx, []int{in.ID})
		if err != nil {
			return nil, EventSummary{}, fmt.Errorf("the catalogue is not available: %w", err)
		}
		if len(events) == 0 {
			return nil, EventSummary{}, fmt.Errorf("event %d not found", in.ID)
		}
		return nil, toSummary(events[0]), nil
	})

	return server
}

// NewHandler serves the MCP server over stateless Streamable HTTP, requiring the bearer token.
func NewHandler(server *mcp.Server, token string) http.Handler {
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})
	return requireBearerToken(token, handler)
}

func requireBearerToken(token string, next http.Handler) http.Handler {
	expected := []byte("Bearer " + token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := []byte(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare(got, expected) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
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
		Free:         e.Free,
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
