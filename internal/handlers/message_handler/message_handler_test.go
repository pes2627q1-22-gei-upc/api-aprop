package message_handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"api-aprop/internal/handlers/requester"
	"api-aprop/internal/models/message_model"
	"api-aprop/internal/services/message_service"
)

const validUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"

type fakeService struct {
	err          error
	sendCalls    int
	listCalls    int
	gotText      string
	gotEventID   int
	gotPageNum   int
	gotPageSize  int
	sentDateTime time.Time
}

func (f *fakeService) ListMessages(_ context.Context, _ string, eventID, pageNum, pageSize int) (message_model.MessagePage, error) {
	f.listCalls++
	f.gotEventID, f.gotPageNum, f.gotPageSize = eventID, pageNum, pageSize
	if f.err != nil {
		return message_model.MessagePage{}, f.err
	}
	return message_model.MessagePage{EventID: eventID, Total: 1, PageNum: pageNum, PageSize: pageSize,
		Messages: []message_model.Message{{UserID: validUserID, EventID: eventID, Text: "hola"}}}, nil
}

func (f *fakeService) SendMessage(_ context.Context, userID string, eventID int, text string) (message_model.Message, error) {
	f.sendCalls++
	f.gotEventID, f.gotText = eventID, text
	if f.err != nil {
		return message_model.Message{}, f.err
	}
	return message_model.Message{UserID: userID, Username: "anna", EventID: eventID, Text: text, DateSent: f.sentDateTime}, nil
}

func do(svc *fakeService, method, path, body, userID string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	NewMessageHandler(svc, nil).RegisterRoutes(mux)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if userID != "" {
		req.Header.Set(requester.UserIDHeader, userID)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestR11_ChatStatusCodes(t *testing.T) {
	cases := []struct {
		name, method, path, body, user string
		svcErr                         error
		want                           int
	}{
		{"send ok", http.MethodPost, "/events/7/messages", `{"text":"hola"}`, validUserID, nil, http.StatusCreated},
		{"list ok", http.MethodGet, "/events/7/messages", "", validUserID, nil, http.StatusOK},
		{"send not signed up", http.MethodPost, "/events/7/messages", `{"text":"hola"}`, validUserID, fmt.Errorf("x: %w", message_service.ErrNotSignedUp), http.StatusForbidden},
		{"list not signed up", http.MethodGet, "/events/7/messages", "", validUserID, fmt.Errorf("x: %w", message_service.ErrNotSignedUp), http.StatusForbidden},
		{"unknown event", http.MethodPost, "/events/7/messages", `{"text":"hola"}`, validUserID, message_service.ErrEventNotFound, http.StatusNotFound},
		{"duplicate (R1)", http.MethodPost, "/events/7/messages", `{"text":"hola"}`, validUserID, message_service.ErrDuplicateMessage, http.StatusConflict},
		{"invalid text", http.MethodPost, "/events/7/messages", `{"text":"   "}`, validUserID, message_service.ErrInvalidText, http.StatusBadRequest},
		{"invalid pagination", http.MethodGet, "/events/7/messages?pageSize=500", "", validUserID, message_service.ErrInvalidPagination, http.StatusBadRequest},
		{"internal error", http.MethodGet, "/events/7/messages", "", validUserID, fmt.Errorf("db down"), http.StatusInternalServerError},
		{"no identity on send", http.MethodPost, "/events/7/messages", `{"text":"hola"}`, "", nil, http.StatusUnauthorized},
		{"no identity on list", http.MethodGet, "/events/7/messages", "", "", nil, http.StatusUnauthorized},
		{"bad event id", http.MethodPost, "/events/abc/messages", `{"text":"hola"}`, validUserID, nil, http.StatusBadRequest},
		{"missing text", http.MethodPost, "/events/7/messages", `{}`, validUserID, nil, http.StatusBadRequest},
		{"text not a string", http.MethodPost, "/events/7/messages", `{"text":5}`, validUserID, nil, http.StatusBadRequest},
		{"bad json", http.MethodPost, "/events/7/messages", `{`, validUserID, nil, http.StatusBadRequest},
		{"page num not a number", http.MethodGet, "/events/7/messages?pageNum=x", "", validUserID, nil, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeService{err: tc.svcErr}
			rec := do(svc, tc.method, tc.path, tc.body, tc.user)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

func TestSendMessage_CreatedBody(t *testing.T) {
	sent := time.Date(2026, 11, 19, 18, 30, 0, 0, time.UTC)
	svc := &fakeService{sentDateTime: sent}
	rec := do(svc, http.MethodPost, "/events/7/messages", `{"text":"Ens veiem a l'entrada"}`, validUserID)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d (body: %s)", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"user-id":   validUserID,
		"username":  "anna",
		"event-id":  float64(7),
		"text":      "Ens veiem a l'entrada",
		"date-sent": "2026-11-19T18:30:00Z",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, want %v", k, got[k], v)
		}
	}
	if svc.gotEventID != 7 || svc.gotText != "Ens veiem a l'entrada" {
		t.Errorf("service got event %d, text %q", svc.gotEventID, svc.gotText)
	}
}

func TestListMessages_PageParams(t *testing.T) {
	svc := &fakeService{}
	rec := do(svc, http.MethodGet, "/events/7/messages?pageNum=2&pageSize=10", "", validUserID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body: %s)", rec.Code, rec.Body.String())
	}
	if svc.gotEventID != 7 || svc.gotPageNum != 2 || svc.gotPageSize != 10 {
		t.Errorf("service got event %d, page %d, size %d", svc.gotEventID, svc.gotPageNum, svc.gotPageSize)
	}
	var page message_model.MessagePage
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.EventID != 7 || page.PageNum != 2 || page.PageSize != 10 || page.Total != 1 || len(page.Messages) != 1 {
		t.Errorf("page = %+v", page)
	}
	for _, key := range []string{`"event-id"`, `"page-num"`, `"page-size"`, `"messages"`, `"total"`} {
		if !strings.Contains(rec.Body.String(), key) {
			t.Errorf("body misses %s: %s", key, rec.Body.String())
		}
	}
}
