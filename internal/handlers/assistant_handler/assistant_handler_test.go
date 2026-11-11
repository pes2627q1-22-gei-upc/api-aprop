package assistant_handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"api-aprop/internal/clients/firebase_client"
	"api-aprop/internal/handlers/requester"
	"api-aprop/internal/middleware/auth_middleware"
	"api-aprop/internal/models/chatbot_model"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/services/assistant_service"
	"api-aprop/internal/services/auth_service"
)

const validUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"

type fakeService struct {
	exchange     chatbot_model.Exchange
	conversation chatbot_model.Conversation
	err          error

	gotUserID  string
	gotText    string
	gotByVoice bool
}

func (f *fakeService) SendMessage(_ context.Context, userID, text string, byVoice bool) (chatbot_model.Exchange, error) {
	f.gotUserID, f.gotText, f.gotByVoice = userID, text, byVoice
	return f.exchange, f.err
}

func (f *fakeService) GetConversation(_ context.Context, userID string) (chatbot_model.Conversation, error) {
	f.gotUserID = userID
	return f.conversation, f.err
}

func date(s string) *time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return &t
}

func sendMessage(t *testing.T, service *fakeService, userID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/assistant/messages", strings.NewReader(body))
	if userID != "" {
		req.Header.Set(requester.UserIDHeader, userID)
	}
	rec := httptest.NewRecorder()
	NewAssistantHandler(service, nil).SendMessage(rec, req)
	return rec
}

func TestSendMessage_Created(t *testing.T) {
	sentAt := time.Date(2026, 10, 19, 14, 30, 0, 0, time.UTC)
	service := &fakeService{exchange: chatbot_model.Exchange{
		ConversationStartedAt: sentAt,
		UserMessage:           chatbot_model.Message{SentAt: sentAt, Sender: chatbot_model.SenderUser, Text: "concerts a Girona", ByVoice: true},
		AssistantMessage: chatbot_model.Message{
			SentAt: sentAt.Add(time.Second), Sender: chatbot_model.SenderAssistant, Text: "He trobat 1 concert.",
			Interpretation:    &chatbot_model.Interpretation{Category: "concert", Municipality: "Girona", DateFrom: date("2026-10-24"), DateTo: date("2026-10-25")},
			RecommendedEvents: []event_model.Event{{ID: 7, Title: "Jazz"}},
		},
	}}

	rec := sendMessage(t, service, validUserID, `{"text":"concerts a Girona","by-voice":true}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if service.gotUserID != validUserID || service.gotText != "concerts a Girona" || !service.gotByVoice {
		t.Errorf("service got %q %q %v", service.gotUserID, service.gotText, service.gotByVoice)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	user := body["user-message"].(map[string]any)
	if user["sender"] != "user" || user["by-voice"] != true {
		t.Errorf("user message = %v", user)
	}
	if _, ok := user["interpretation"]; ok {
		t.Errorf("R12: the user message must not carry an interpretation")
	}
	if _, ok := user["recommended-events"]; ok {
		t.Errorf("R12: the user message must not recommend events")
	}

	reply := body["assistant-message"].(map[string]any)
	interpretation := reply["interpretation"].(map[string]any)
	want := map[string]any{
		"category": "concert", "municipality": "Girona",
		"date-from": "2026-10-24", "date-to": "2026-10-25", "date-range": "del 24/10/2026 al 25/10/2026",
	}
	for k, v := range want {
		if interpretation[k] != v {
			t.Errorf("interpretation[%s] = %v, want %v", k, interpretation[k], v)
		}
	}
	events := reply["recommended-events"].([]any)
	if len(events) != 1 || events[0].(map[string]any)["id"] != float64(7) {
		t.Errorf("recommended-events = %v", events)
	}
	if body["conversation-started-at"] != "2026-10-19T14:30:00Z" {
		t.Errorf("conversation-started-at = %v", body["conversation-started-at"])
	}
}

func TestSendMessage_BadRequest(t *testing.T) {
	if rec := sendMessage(t, &fakeService{}, "", `{"text":"hola"}`); rec.Code != http.StatusUnauthorized {
		t.Errorf("missing header: status = %d", rec.Code)
	}
	if rec := sendMessage(t, &fakeService{}, validUserID, `{"text":`); rec.Code != http.StatusBadRequest {
		t.Errorf("malformed body: status = %d", rec.Code)
	}
}

func TestSendMessage_ErrorMapping(t *testing.T) {
	tests := map[string]struct {
		err  error
		want int
	}{
		"invalid user":  {assistant_service.ErrInvalidUserID, http.StatusBadRequest},
		"empty text":    {assistant_service.ErrInvalidMessage, http.StatusBadRequest},
		"gemini down":   {assistant_service.ErrAssistantUnavailable, http.StatusBadGateway},
		"gemini slow":   {assistant_service.ErrAssistantTimeout, http.StatusGatewayTimeout},
		"database down": {errors.New("db down"), http.StatusInternalServerError},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rec := sendMessage(t, &fakeService{err: tt.err}, validUserID, `{"text":"hola"}`)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
			var apiError map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &apiError); err != nil || apiError["error"] == "" {
				t.Errorf("the client needs a clear error message: %s", rec.Body)
			}
		})
	}
}

func TestGetConversation(t *testing.T) {
	startedAt := time.Date(2026, 10, 19, 14, 0, 0, 0, time.UTC)
	service := &fakeService{conversation: chatbot_model.Conversation{StartedAt: startedAt, Messages: []chatbot_model.Message{
		{SentAt: startedAt, Sender: chatbot_model.SenderUser, Text: "hola"},
		{SentAt: startedAt.Add(time.Second), Sender: chatbot_model.SenderAssistant, Text: "Hola!", Interpretation: &chatbot_model.Interpretation{}},
	}}}
	req := httptest.NewRequest(http.MethodGet, "/assistant/conversation", nil)
	req.Header.Set(requester.UserIDHeader, validUserID)
	rec := httptest.NewRecorder()
	NewAssistantHandler(service, nil).GetConversation(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	var body conversationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Messages) != 2 || body.Messages[1].Interpretation != nil {
		t.Errorf("an empty interpretation is omitted: %+v", body.Messages)
	}

	service = &fakeService{err: assistant_service.ErrConversationNotFound}
	rec = httptest.NewRecorder()
	NewAssistantHandler(service, nil).GetConversation(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("no conversation: status = %d", rec.Code)
	}
}

func TestDateRangeLabel(t *testing.T) {
	tests := []struct {
		from, to *time.Time
		want     string
	}{
		{date("2026-10-24"), date("2026-10-25"), "del 24/10/2026 al 25/10/2026"},
		{date("2026-11-01"), date("2026-11-11"), "de l'1/11/2026 a l'11/11/2026"},
		{date("2026-10-24"), date("2026-10-24"), "el 24/10/2026"},
		{date("2026-10-24"), nil, "a partir del 24/10/2026"},
		{nil, date("2026-10-31"), "fins al 31/10/2026"},
		{nil, nil, ""},
	}
	for _, tt := range tests {
		if got := dateRangeLabel(tt.from, tt.to); got != tt.want {
			t.Errorf("dateRangeLabel = %q, want %q", got, tt.want)
		}
	}
}

type fakeResolver struct {
	id  string
	err error
}

func (f fakeResolver) ResolveUserID(context.Context, string) (string, error) { return f.id, f.err }

func withFirebaseUser(req *http.Request) *http.Request {
	return req.WithContext(auth_middleware.ContextWithUser(req.Context(), firebase_client.User{UID: "Vw3kU1pZ8rQ2hXyT6mN9aB4cD7e1"}))
}

// A Firebase user is translated to its database UUID and the header is ignored.
func TestGetConversation_TranslatesFirebaseUID(t *testing.T) {
	service := &fakeService{}
	req := withFirebaseUser(httptest.NewRequest(http.MethodGet, "/assistant/conversation", nil))
	req.Header.Set(requester.UserIDHeader, "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d")
	rec := httptest.NewRecorder()
	NewAssistantHandler(service, fakeResolver{id: validUserID}).GetConversation(rec, req)

	if service.gotUserID != validUserID {
		t.Fatalf("user passed to service = %q, want %q", service.gotUserID, validUserID)
	}

	rec = httptest.NewRecorder()
	NewAssistantHandler(service, fakeResolver{err: auth_service.ErrUserNotRegistered}).GetConversation(rec, withFirebaseUser(httptest.NewRequest(http.MethodGet, "/assistant/conversation", nil)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("not registered: status = %d, want 401", rec.Code)
	}
}
