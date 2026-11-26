package user_handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/clients/firebase_client"
	"api-aprop/internal/handlers/user_handler"
	"api-aprop/internal/middleware/auth_middleware"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/language_model"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/services/auth_service"
	"api-aprop/internal/services/translation_service"
	"api-aprop/internal/services/user_service"
)

type fakeUserService struct {
	profiles      map[string]user_model.UserProfile
	createdEvents map[string][]event_model.Event
	ratings       map[string][]user_model.ReceivedRating
	failUsername  bool

	deleteErr  error
	deletedIDs []string

	uploadErr error

	searchResult    user_model.UserSearchResult
	searchErr       error
	searchCalls     int
	searchRequester string
	searchQuery     string
	searchPageNum   int
	searchPageSize  int

	blocked      user_model.BlockedUser
	blockedList  []user_model.BlockedUser
	blockErr     error
	blockCalls   int
	unblockCalls int
	listCalls    int
	gotRequester string
	gotTarget    string
}

func (f *fakeUserService) SearchUsers(_ context.Context, requesterID, query string, pageNum, pageSize int) (user_model.UserSearchResult, error) {
	f.searchCalls++
	f.searchRequester, f.searchQuery, f.searchPageNum, f.searchPageSize = requesterID, query, pageNum, pageSize
	return f.searchResult, f.searchErr
}

func (f *fakeUserService) DeleteAccount(_ context.Context, requesterID string) error {
	f.deletedIDs = append(f.deletedIDs, requesterID)
	return f.deleteErr
}

func (f *fakeUserService) GetUserProfile(_ context.Context, targetIdentifier, requesterIdentifier string) (user_model.UserProfile, error) {
	if targetIdentifier == "me" {
		if requesterIdentifier == "" {
			return user_model.UserProfile{}, user_service.ErrUnauthorized
		}
		p, ok := f.profiles[requesterIdentifier]
		if !ok {
			return user_model.UserProfile{}, user_service.ErrUserNotFound
		}
		return p, nil
	}
	p, ok := f.profiles[targetIdentifier]
	if !ok {
		return user_model.UserProfile{}, user_service.ErrUserNotFound
	}
	return p, nil
}

func (f *fakeUserService) ListCreatedEvents(_ context.Context, targetIdentifier string) ([]event_model.Event, error) {
	events, ok := f.createdEvents[targetIdentifier]
	if !ok {
		return nil, user_service.ErrUserNotFound
	}
	return events, nil
}

func (f *fakeUserService) ListReceivedRatings(_ context.Context, targetIdentifier string) ([]user_model.ReceivedRating, error) {
	r, ok := f.ratings[targetIdentifier]
	if !ok {
		return nil, user_service.ErrUserNotFound
	}
	return r, nil
}

func (f *fakeUserService) UpdateProfile(_ context.Context, requesterIdentifier string, input user_service.UpdateProfileInput) (user_model.UserProfile, error) {
	if requesterIdentifier == "" {
		return user_model.UserProfile{}, user_service.ErrUnauthorized
	}
	if f.failUsername {
		return user_model.UserProfile{}, user_service.ErrUsernameTaken
	}
	p, ok := f.profiles[requesterIdentifier]
	if !ok {
		return user_model.UserProfile{}, user_service.ErrUserNotFound
	}
	if input.Name != nil {
		p.Name = *input.Name
	}
	if input.Username != nil {
		p.Username = *input.Username
	}
	if input.Biography != nil {
		p.Biography = *input.Biography
	}
	if input.ShowPlans != nil {
		p.ShowPlans = *input.ShowPlans
	}
	if input.Language != nil {
		if !language_model.Valid(*input.Language) {
			return user_model.UserProfile{}, user_service.ErrInvalidInput
		}
		p.Language = *input.Language
	}
	f.profiles[requesterIdentifier] = p
	return p, nil
}

func (f *fakeUserService) UploadAvatar(_ context.Context, requesterIdentifier string, filename string, contentType string, _ io.Reader) (user_model.UserProfile, error) {
	if requesterIdentifier == "" {
		return user_model.UserProfile{}, user_service.ErrUnauthorized
	}
	if f.uploadErr != nil {
		return user_model.UserProfile{}, f.uploadErr
	}
	p, ok := f.profiles[requesterIdentifier]
	if !ok {
		return user_model.UserProfile{}, user_service.ErrUserNotFound
	}
	p.ProfilePicture = "https://uploaded.example.com/" + filename
	f.profiles[requesterIdentifier] = p
	return p, nil
}

func (f *fakeUserService) GetUserFriends(ctx context.Context, userid string) ([]string, error) {
	return []string{}, nil
}
func (f *fakeUserService) GetFriendRequests(ctx context.Context, userid string) ([]user_model.FriendRequest, error) {
	return []user_model.FriendRequest{}, nil
}
func (f *fakeUserService) SendFriendRequest(ctx context.Context, userid string, targetid string) (user_model.FriendRequest, error) {
	return user_model.FriendRequest{}, nil
}
func (f *fakeUserService) AcceptFriendRequest(ctx context.Context, userid string, targetid string, date time.Time) (user_model.FriendRequest, error) {
	return user_model.FriendRequest{}, nil
}
func (f *fakeUserService) DenyFriendRequest(ctx context.Context, userid string, targetid string, date time.Time) (user_model.FriendRequest, error) {
	return user_model.FriendRequest{}, nil
}
func (f *fakeUserService) BlockUser(_ context.Context, requesterID, targetID string) (user_model.BlockedUser, error) {
	f.blockCalls++
	f.gotRequester, f.gotTarget = requesterID, targetID
	return f.blocked, f.blockErr
}
func (f *fakeUserService) UnblockUser(_ context.Context, requesterID, targetID string) error {
	f.unblockCalls++
	f.gotRequester, f.gotTarget = requesterID, targetID
	return f.blockErr
}
func (f *fakeUserService) ListBlockedUsers(_ context.Context, requesterID string) ([]user_model.BlockedUser, error) {
	f.listCalls++
	f.gotRequester = requesterID
	return f.blockedList, f.blockErr
}
func (f *fakeUserService) Unfriend(ctx context.Context, userid string, targetid string) error {
	return nil
}
func (f *fakeUserService) GetSettings(ctx context.Context, userid string) (user_model.Settings, error) {
	return user_model.Settings{ShowPlans: true, NotificationBefore: "3 days", AllowedNotifTypes: []int{7, 8, 9}}, nil
}

const (
	aliceID          = "11111111-1111-1111-1111-111111111111"
	aliceFirebaseUID = "alice-firebase-uid"
)

// fakeResolver translates the Firebase UID of alice and fails for unknown accounts, like auth_service does.
type fakeResolver struct{}

func (fakeResolver) ResolveUserID(_ context.Context, firebaseUID string) (string, error) {
	if firebaseUID == aliceFirebaseUID {
		return aliceID, nil
	}
	return "", auth_service.ErrUserNotRegistered
}

func setupMux(svc user_service.Service) *http.ServeMux {
	h := user_handler.NewUserHandler(svc, fakeResolver{})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

func TestGetMyProfile_Success(t *testing.T) {
	svc := &fakeUserService{
		profiles: map[string]user_model.UserProfile{
			aliceID: {
				ID:             testID(1),
				Username:       "alice",
				Name:           "Alice Smith",
				Email:          "alice@example.com",
				PlansCount:     3,
				FriendsCount:   10,
				DateRegistered: time.Now(),
				PlansVisible:   true,
			},
		},
	}
	mux := setupMux(svc)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req.Header.Set("X-User-ID", aliceID)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var res user_model.UserProfile
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}

	if res.Username != "alice" || res.Email != "alice@example.com" {
		t.Errorf("unexpected profile: %+v", res)
	}
}

func TestGetMyProfile_WithAuthContext(t *testing.T) {
	svc := &fakeUserService{
		profiles: map[string]user_model.UserProfile{
			aliceID: {
				ID:             testID(1),
				Username:       "alice",
				Email:          "alice@example.com",
				PlansVisible:   true,
				DateRegistered: time.Now(),
			},
		},
	}
	mux := setupMux(svc)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	ctx := auth_middleware.ContextWithUser(req.Context(), firebase_client.User{
		UID:   aliceFirebaseUID,
		Email: "alice@example.com",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetMyProfile_UnregisteredFirebaseUser(t *testing.T) {
	mux := setupMux(&fakeUserService{profiles: make(map[string]user_model.UserProfile)})

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req = req.WithContext(auth_middleware.ContextWithUser(req.Context(), firebase_client.User{UID: "not-synced"}))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// The requester is optional in a public profile, but a signed in user is translated to the database ID.
func TestGetUserProfile_ResolvesRequester(t *testing.T) {
	svc := &recordingUserService{fakeUserService: fakeUserService{
		profiles: map[string]user_model.UserProfile{"bob": {ID: testID(2), Username: "bob"}},
	}}
	mux := setupMux(svc)

	req := httptest.NewRequest(http.MethodGet, "/users/bob", nil)
	req = req.WithContext(auth_middleware.ContextWithUser(req.Context(), firebase_client.User{UID: aliceFirebaseUID}))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if svc.gotRequester != aliceID {
		t.Errorf("requester = %q, want %q", svc.gotRequester, aliceID)
	}
}

type recordingUserService struct {
	fakeUserService
	gotRequester string
}

func (r *recordingUserService) GetUserProfile(ctx context.Context, target, requester string) (user_model.UserProfile, error) {
	r.gotRequester = requester
	return r.fakeUserService.GetUserProfile(ctx, target, requester)
}

func TestGetMyProfile_Unauthorized(t *testing.T) {
	svc := &fakeUserService{profiles: make(map[string]user_model.UserProfile)}
	mux := setupMux(svc)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetUserProfile_Success(t *testing.T) {
	svc := &fakeUserService{
		profiles: map[string]user_model.UserProfile{
			"bob": {
				ID:             testID(2),
				Username:       "bob",
				Name:           "Bob Builder",
				PlansVisible:   false,
				FriendsCount:   5,
				DateRegistered: time.Now(),
			},
		},
	}
	mux := setupMux(svc)

	req := httptest.NewRequest(http.MethodGet, "/users/bob", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var res user_model.UserProfile
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if res.Username != "bob" {
		t.Errorf("expected bob, got %s", res.Username)
	}
}

func TestGetUserProfile_NotFound(t *testing.T) {
	svc := &fakeUserService{profiles: make(map[string]user_model.UserProfile)}
	mux := setupMux(svc)

	req := httptest.NewRequest(http.MethodGet, "/users/unknown", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestUpdateMyProfile_Success(t *testing.T) {
	svc := &fakeUserService{
		profiles: map[string]user_model.UserProfile{
			aliceID: {
				ID:             testID(1),
				Username:       "alice",
				Name:           "Alice Smith",
				ShowPlans:      true,
				DateRegistered: time.Now(),
			},
		},
	}
	mux := setupMux(svc)

	body := []byte(`{"name":"Alice Updated","biography":"New bio","show-plans":false}`)
	req := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewReader(body))
	req.Header.Set("X-User-ID", aliceID)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var res user_model.UserProfile
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if res.Name != "Alice Updated" || res.ShowPlans != false {
		t.Errorf("unexpected updated profile: %+v", res)
	}
}

func TestUpdateMyProfile_UsernameConflict(t *testing.T) {
	svc := &fakeUserService{
		failUsername: true,
		profiles: map[string]user_model.UserProfile{
			aliceID: {ID: testID(1), Username: "alice", DateRegistered: time.Now()},
		},
	}
	mux := setupMux(svc)

	body := []byte(`{"username":"taken_name"}`)
	req := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewReader(body))
	req.Header.Set("X-User-ID", aliceID)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestUploadAvatar_Success(t *testing.T) {
	svc := &fakeUserService{
		profiles: map[string]user_model.UserProfile{
			aliceID: {ID: testID(1), Username: "alice"},
		},
	}
	mux := setupMux(svc)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("avatar", "profile.png")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = part.Write([]byte("fake-image-bytes"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/users/me/avatar", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-User-ID", aliceID)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var res user_model.UserProfile
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.ProfilePicture != "https://uploaded.example.com/profile.png" {
		t.Errorf("expected updated avatar, got %s", res.ProfilePicture)
	}
}

// #247: storage errors of the avatar upload.
func TestUploadAvatar_StorageErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"storage not configured", user_service.ErrStorageUnavailable, http.StatusServiceUnavailable},
		{"storage failed", user_service.ErrStorageFailed, http.StatusBadGateway},
		{"invalid image", user_service.ErrInvalidInput, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeUserService{profiles: map[string]user_model.UserProfile{aliceID: {ID: testID(1)}}, uploadErr: tt.err}
			mux := setupMux(svc)

			var buf bytes.Buffer
			writer := multipart.NewWriter(&buf)
			part, _ := writer.CreateFormFile("avatar", "profile.png")
			_, _ = part.Write([]byte("fake-image-bytes"))
			writer.Close()
			req := httptest.NewRequest(http.MethodPost, "/users/me/avatar", &buf)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			req.Header.Set("X-User-ID", aliceID)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d: %s", w.Code, tt.want, w.Body.String())
			}
		})
	}
}

func TestUploadAvatar_Unauthorized(t *testing.T) {
	svc := &fakeUserService{profiles: make(map[string]user_model.UserProfile)}
	mux := setupMux(svc)

	req := httptest.NewRequest(http.MethodPost, "/users/me/avatar", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUploadAvatar_MissingField(t *testing.T) {
	svc := &fakeUserService{
		profiles: map[string]user_model.UserProfile{aliceID: {ID: testID(1)}},
	}
	mux := setupMux(svc)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/users/me/avatar", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-User-ID", aliceID)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListCreatedEvents_Success(t *testing.T) {
	svc := &fakeUserService{
		createdEvents: map[string][]event_model.Event{
			"alice": {
				{ID: 10, Title: "Exposicio d'Art"},
			},
		},
	}
	mux := setupMux(svc)

	req := httptest.NewRequest(http.MethodGet, "/users/alice/created-events", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var events []event_model.Event
	if err := json.NewDecoder(w.Body).Decode(&events); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if len(events) != 1 || events[0].Title != "Exposicio d'Art" {
		t.Errorf("unexpected events: %+v", events)
	}
}

func TestListReceivedRatings(t *testing.T) {
	svc := &fakeUserService{ratings: map[string][]user_model.ReceivedRating{
		"alice": {{EventID: 10, EventTitle: "Jazz", Rating: 4}},
	}}
	mux := setupMux(svc)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users/alice/ratings", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var got []map[string]any
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0]["event-id"] != float64(10) || got[0]["rating"] != float64(4) {
		t.Errorf("unexpected body: %+v", got)
	}

	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users/bob/ratings", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// testID builds a deterministic user UUID for the fakes.
func testID(n byte) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte{15: n}, Valid: true}
}

// langTranslator prefixes the title with the language it was asked for.
type langTranslator struct{ translation_service.Service }

func (langTranslator) UserLanguage(context.Context, string) (string, error) { return "ca", nil }
func (langTranslator) TranslateEvents(_ context.Context, lang string, events []event_model.Event) []event_model.Event {
	out := slices.Clone(events)
	for i := range out {
		out[i].Title = "[" + lang + "] " + out[i].Title
	}
	return out
}

func TestUpdateMyProfile_Language(t *testing.T) {
	tests := []struct {
		body     string
		wantCode int
		wantLang string
	}{
		{`{"language":"en"}`, http.StatusOK, "en"},
		{`{"language":"fr"}`, http.StatusBadRequest, "ca"},
	}
	for _, tt := range tests {
		t.Run(tt.body, func(t *testing.T) {
			svc := &fakeUserService{profiles: map[string]user_model.UserProfile{aliceID: {ID: testID(1), Username: "alice", Language: "ca"}}}
			req := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("X-User-ID", aliceID)
			w := httptest.NewRecorder()
			setupMux(svc).ServeHTTP(w, req)
			if w.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d: %s", w.Code, tt.wantCode, w.Body.String())
			}
			if got := svc.profiles[aliceID].Language; got != tt.wantLang {
				t.Errorf("stored language = %q, want %q", got, tt.wantLang)
			}
		})
	}
}

func TestListCreatedEvents_Lang(t *testing.T) {
	svc := &fakeUserService{createdEvents: map[string][]event_model.Event{"alice": {{ID: 10, Title: "Expo"}}}}
	mux := http.NewServeMux()
	user_handler.NewUserHandler(svc, fakeResolver{}).WithTranslator(langTranslator{}).RegisterRoutes(mux)

	for target, want := range map[string]string{"/users/alice/created-events?lang=es": "[es] Expo", "/users/alice/created-events": "[ca] Expo"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
		var events []event_model.Event
		if err := json.NewDecoder(w.Body).Decode(&events); err != nil || w.Code != http.StatusOK || events[0].Title != want {
			t.Errorf("%s: status %d, events %+v, err %v; want title %q", target, w.Code, events, err, want)
		}
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users/alice/created-events?lang=fr", nil))
	if w.Code != http.StatusBadRequest {
		t.Errorf("lang=fr: status = %d, want 400", w.Code)
	}
}
