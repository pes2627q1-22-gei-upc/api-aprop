package user_handler_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"api-aprop/internal/models/user_model"
	"api-aprop/internal/services/user_service"
)

func TestSearchUsers(t *testing.T) {
	page := user_model.UserSearchResult{
		Query: "joan", Total: 1, PageNum: 0, PageSize: 20,
		Users: []user_model.UserSummary{{Username: "joanpuig", Name: "Joan Puig", ProfilePicture: "https://img/joan.png"}},
	}

	tests := []struct {
		name         string
		url          string
		header       string
		svcErr       error
		wantStatus   int
		wantCalls    int
		wantPageNum  int
		wantPageSize int
	}{
		{name: "search", url: "/users?q=joan", header: aliceID, wantStatus: http.StatusOK, wantCalls: 1},
		{name: "with pagination", url: "/users?q=joan&pageNum=2&pageSize=10", header: aliceID, wantStatus: http.StatusOK, wantCalls: 1, wantPageNum: 2, wantPageSize: 10},
		{name: "no identity", url: "/users?q=joan", wantStatus: http.StatusUnauthorized},
		{name: "page num not a number", url: "/users?q=joan&pageNum=x", header: aliceID, wantStatus: http.StatusBadRequest},
		{name: "page size not a number", url: "/users?q=joan&pageSize=1.5", header: aliceID, wantStatus: http.StatusBadRequest},
		{name: "invalid pagination", url: "/users?q=joan&pageSize=500", header: aliceID, svcErr: fmt.Errorf("%w: pageSize", user_service.ErrInvalidInput), wantStatus: http.StatusBadRequest, wantCalls: 1, wantPageSize: 500},
		{name: "internal error", url: "/users?q=joan", header: aliceID, svcErr: errors.New("db down"), wantStatus: http.StatusInternalServerError, wantCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeUserService{searchResult: page, searchErr: tt.svcErr}
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			if tt.header != "" {
				req.Header.Set("X-User-ID", tt.header)
			}
			w := httptest.NewRecorder()

			setupMux(svc).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (%s)", w.Code, tt.wantStatus, w.Body.String())
			}
			if svc.searchCalls != tt.wantCalls {
				t.Fatalf("service calls = %d, want %d", svc.searchCalls, tt.wantCalls)
			}
			if tt.wantCalls == 0 {
				return
			}
			if svc.searchRequester != aliceID || svc.searchQuery != "joan" || svc.searchPageNum != tt.wantPageNum || svc.searchPageSize != tt.wantPageSize {
				t.Errorf("service got (%q, %q, %d, %d)", svc.searchRequester, svc.searchQuery, svc.searchPageNum, svc.searchPageSize)
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			var body map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			for _, key := range []string{"query", "total", "page-num", "page-size", "users"} {
				if _, ok := body[key]; !ok {
					t.Errorf("response without %q: %s", key, w.Body.String())
				}
			}
			users, _ := body["users"].([]any)
			if len(users) != 1 {
				t.Fatalf("users = %v", body["users"])
			}
			user, _ := users[0].(map[string]any)
			if user["profile-picture"] != "https://img/joan.png" {
				t.Errorf("user = %v, want kebab-case keys", user)
			}
			if _, leaks := user["email"]; leaks {
				t.Errorf("the search must not show the email: %v", user)
			}
		})
	}
}

// GET /users/me and GET /users/{id} must keep working next to GET /users.
func TestSearchUsersDoesNotShadowProfileRoutes(t *testing.T) {
	svc := &fakeUserService{}
	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req.Header.Set("X-User-ID", aliceID)
	setupMux(svc).ServeHTTP(httptest.NewRecorder(), req)
	if svc.searchCalls != 0 {
		t.Errorf("GET /users/me reached the search")
	}
}
