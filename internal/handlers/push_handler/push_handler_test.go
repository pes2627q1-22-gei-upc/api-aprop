package push_handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"api-aprop/internal/handlers/requester"
	"api-aprop/internal/services/push_service"
)

const validUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"

type fakeService struct {
	err                         error
	registerCalls, deleteCalls  int
	gotUser, gotToken, gotPlatf string
}

func (f *fakeService) RegisterDevice(_ context.Context, userID, token, platform string) error {
	f.registerCalls++
	f.gotUser, f.gotToken, f.gotPlatf = userID, token, platform
	return f.err
}

func (f *fakeService) UnregisterDevice(_ context.Context, userID, token string) error {
	f.deleteCalls++
	f.gotUser, f.gotToken = userID, token
	return f.err
}

func (f *fakeService) DispatchPending(context.Context) (int, error) { return 0, nil }

func do(svc *fakeService, method, path, body, userID string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	NewPushHandler(svc, nil).RegisterRoutes(mux)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if userID != "" {
		req.Header.Set(requester.UserIDHeader, userID)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestDeviceStatusCodes(t *testing.T) {
	const okBody = `{"token":"fid:APA91b","platform":"android"}`
	cases := []struct {
		name, method, path, body, user string
		svcErr                         error
		want                           int
	}{
		{"register ok", http.MethodPut, "/users/me/devices", okBody, validUserID, nil, http.StatusNoContent},
		{"unregister ok", http.MethodDelete, "/users/me/devices/fid:APA91b", "", validUserID, nil, http.StatusNoContent},
		{"register without identity", http.MethodPut, "/users/me/devices", okBody, "", nil, http.StatusUnauthorized},
		{"unregister without identity", http.MethodDelete, "/users/me/devices/fid:APA91b", "", "", nil, http.StatusUnauthorized},
		{"malformed json", http.MethodPut, "/users/me/devices", `{"token":`, validUserID, nil, http.StatusBadRequest},
		{"missing token", http.MethodPut, "/users/me/devices", `{"platform":"ios"}`, validUserID, nil, http.StatusBadRequest},
		{"missing platform", http.MethodPut, "/users/me/devices", `{"token":"t"}`, validUserID, nil, http.StatusBadRequest},
		{"invalid token", http.MethodPut, "/users/me/devices", okBody, validUserID, push_service.ErrInvalidToken, http.StatusBadRequest},
		{"invalid platform", http.MethodPut, "/users/me/devices", okBody, validUserID, push_service.ErrInvalidPlatform, http.StatusBadRequest},
		{"invalid user", http.MethodPut, "/users/me/devices", okBody, "nope", fmt.Errorf("x: %w", push_service.ErrInvalidUserID), http.StatusBadRequest},
		{"internal error", http.MethodPut, "/users/me/devices", okBody, validUserID, errors.New("db down"), http.StatusInternalServerError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := do(&fakeService{err: c.svcErr}, c.method, c.path, c.body, c.user)
			if rec.Code != c.want {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, c.want, rec.Body.String())
			}
		})
	}
}

func TestRegisterDevicePassesTheRequestToTheService(t *testing.T) {
	svc := &fakeService{}
	do(svc, http.MethodPut, "/users/me/devices", `{"token":"fid:APA91b","platform":"ios"}`, validUserID)
	if svc.registerCalls != 1 || svc.gotUser != validUserID || svc.gotToken != "fid:APA91b" || svc.gotPlatf != "ios" {
		t.Errorf("service got calls=%d user=%q token=%q platform=%q", svc.registerCalls, svc.gotUser, svc.gotToken, svc.gotPlatf)
	}
}

func TestUnregisterDeviceReadsTheTokenFromThePath(t *testing.T) {
	svc := &fakeService{}
	do(svc, http.MethodDelete, "/users/me/devices/fid%3AAPA91b-x_y", "", validUserID)
	if svc.deleteCalls != 1 || svc.gotToken != "fid:APA91b-x_y" {
		t.Errorf("service got calls=%d token=%q, want the unescaped token", svc.deleteCalls, svc.gotToken)
	}
}
