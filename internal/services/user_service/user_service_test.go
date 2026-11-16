package user_service_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/clients/s3_client"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/notification_model"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/user_repository"
	"api-aprop/internal/services/user_service"
)

type mockUserRepo struct {
	usersByID               map[pgtype.UUID]*user_model.User
	showPlans               map[int]bool
	friends                 map[pgtype.UUID]map[pgtype.UUID]bool
	plansCount              map[pgtype.UUID]int
	completedPlans          map[pgtype.UUID]int
	reliability             map[pgtype.UUID]*float32
	createdEvents           map[pgtype.UUID][]event_model.Event
	getUserFriendsResult    []string
	getUserFriendsErr       error
	getFriendRequestsResult []user_model.FriendRequest
	getFriendRequestsErr    error
	pendingRequestExists    bool
	pendingRequestErr       error
	isBlockedResult         bool
	isBlockedErr            error
	areFriendsErr           error
	sendFriendRequestErr    error
	acceptFriendRequestErr  error
	denyFriendRequestErr    error
	unfriendErr             error
	deleteUserErr           error
	deleteUserCalls         int
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		usersByID:      make(map[pgtype.UUID]*user_model.User),
		showPlans:      make(map[int]bool),
		friends:        make(map[pgtype.UUID]map[pgtype.UUID]bool),
		plansCount:     make(map[pgtype.UUID]int),
		completedPlans: make(map[pgtype.UUID]int),
		reliability:    make(map[pgtype.UUID]*float32),
		createdEvents:  make(map[pgtype.UUID][]event_model.Event),
	}
}

func (m *mockUserRepo) addUser(u user_model.User, showPlans bool) {
	copy := u
	m.usersByID[u.ID] = &copy
	m.showPlans[u.Settings] = showPlans
}

func (m *mockUserRepo) GetByID(_ context.Context, id pgtype.UUID) (user_model.User, bool, error) {
	u, ok := m.usersByID[id]
	if !ok {
		return user_model.User{}, false, user_repository.ErrUserNotFound
	}
	return *u, m.showPlans[u.Settings], nil
}

func (m *mockUserRepo) GetByUsername(_ context.Context, username string) (user_model.User, bool, error) {
	for _, u := range m.usersByID {
		if u.Username == username {
			return *u, m.showPlans[u.Settings], nil
		}
	}
	return user_model.User{}, false, user_repository.ErrUserNotFound
}

func (m *mockUserRepo) GetByFirebaseUID(_ context.Context, firebaseUID string) (user_model.User, bool, error) {
	for _, u := range m.usersByID {
		if u.FirebaseUID == firebaseUID {
			return *u, m.showPlans[u.Settings], nil
		}
	}
	return user_model.User{}, false, user_repository.ErrUserNotFound
}

func (m *mockUserRepo) CountFriends(_ context.Context, userUUID pgtype.UUID) (int, error) {
	fMap := m.friends[userUUID]
	count := 0
	for _, areFriends := range fMap {
		if areFriends {
			count++
		}
	}
	return count, nil
}

func (m *mockUserRepo) AreFriends(_ context.Context, user1UUID, user2UUID pgtype.UUID) (bool, error) {
	if m.areFriendsErr != nil {
		return false, m.areFriendsErr
	}

	if m.friends[user1UUID] != nil && m.friends[user1UUID][user2UUID] {
		return true, nil
	}

	if m.friends[user2UUID] != nil && m.friends[user2UUID][user1UUID] {
		return true, nil
	}

	return false, nil
}

func (m *mockUserRepo) CountPlans(_ context.Context, userUUID pgtype.UUID) (int, error) {
	return m.plansCount[userUUID], nil
}

func (m *mockUserRepo) CountCompletedPlans(_ context.Context, userUUID pgtype.UUID) (int, error) {
	return m.completedPlans[userUUID], nil
}

func (m *mockUserRepo) GetReliability(_ context.Context, creatorUUID pgtype.UUID) (*float32, error) {
	return m.reliability[creatorUUID], nil
}

func (m *mockUserRepo) ListCreatedEvents(_ context.Context, creatorUUID pgtype.UUID) ([]event_model.Event, error) {
	return m.createdEvents[creatorUUID], nil
}

func (m *mockUserRepo) UpdateProfile(_ context.Context, id pgtype.UUID, name, username, biography, profilePicture *string) (user_model.User, error) {
	target, ok := m.usersByID[id]
	if !ok {
		return user_model.User{}, user_repository.ErrUserNotFound
	}
	if username != nil && *username != target.Username {
		for _, u := range m.usersByID {
			if u.Username == *username {
				return user_model.User{}, user_repository.ErrUsernameTaken
			}
		}
		target.Username = *username
	}
	if name != nil {
		target.Name = *name
	}
	if biography != nil {
		target.Biography = *biography
	}
	if profilePicture != nil {
		target.ProfilePicture = *profilePicture
	}
	return *target, nil
}

func (m *mockUserRepo) UpdateSettings(_ context.Context, settingsID int, showPlans bool) error {
	m.showPlans[settingsID] = showPlans
	return nil
}

func (m *mockUserRepo) GetUserFriends(_ context.Context, _ pgtype.UUID) ([]string, error) {
	return m.getUserFriendsResult, m.getUserFriendsErr
}
func (m *mockUserRepo) GetFriendRequests(_ context.Context, _ pgtype.UUID) ([]user_model.FriendRequest, error) {
	return m.getFriendRequestsResult, m.getFriendRequestsErr
}

func (m *mockUserRepo) PendingRequestExists(_ context.Context, _, _ pgtype.UUID) (bool, error) {
	return m.pendingRequestExists, m.pendingRequestErr
}

func (m *mockUserRepo) IsBlocked(_ context.Context, _, _ pgtype.UUID) (bool, error) {
	return m.isBlockedResult, m.isBlockedErr
}

func (m *mockUserRepo) SendFriendRequest(_ context.Context, _, _ pgtype.UUID) (user_model.FriendRequest, error) {
	return user_model.FriendRequest{}, m.sendFriendRequestErr
}

func (m *mockUserRepo) AcceptFriendRequest(_ context.Context, _, _ pgtype.UUID, _ time.Time) (user_model.FriendRequest, error) {
	return user_model.FriendRequest{}, m.acceptFriendRequestErr
}

func (m *mockUserRepo) DenyFriendRequest(_ context.Context, _, _ pgtype.UUID, _ time.Time) (user_model.FriendRequest, error) {
	return user_model.FriendRequest{}, m.denyFriendRequestErr
}

func (m *mockUserRepo) Unfriend(_ context.Context, _, _ pgtype.UUID) error {
	return m.unfriendErr
}

// DeleteUser simulates the transaction: the user is only removed if beforeCommit succeeds.
func (m *mockUserRepo) DeleteUser(ctx context.Context, id pgtype.UUID, beforeCommit func(context.Context) error) error {
	m.deleteUserCalls++
	if m.deleteUserErr != nil {
		return m.deleteUserErr
	}
	if _, ok := m.usersByID[id]; !ok {
		return user_repository.ErrUserNotFound
	}
	if beforeCommit != nil {
		if err := beforeCommit(ctx); err != nil {
			return err
		}
	}
	delete(m.usersByID, id)
	return nil
}

type mockNotificationRepo struct {
	getTypeErr error
	createErr  error
	created    []notification_model.Notification
}

func (m *mockNotificationRepo) Get(id int) (notification_model.Notification, error) {
	return notification_model.Notification{}, nil
}
func (m *mockNotificationRepo) ReadNotification(id int) error              { return nil }
func (m *mockNotificationRepo) ReadUserNotifications(id pgtype.UUID) error { return nil }
func (m *mockNotificationRepo) GetNotificationTypes() ([]notification_model.NotificationType, error) {
	return []notification_model.NotificationType{}, nil
}
func (m *mockNotificationRepo) CreateReminders() ([]int32, error) { return []int32{}, nil }
func (f *mockNotificationRepo) CreateFriendSignupNotifications(id pgtype.UUID, event int, text string) ([]int32, error) {
	return []int32{}, nil
}
func (m *mockNotificationRepo) GetUserNotifications(id pgtype.UUID, params map[string]any) ([]notification_model.Notification, error) {
	return []notification_model.Notification{}, nil
}

func (m *mockNotificationRepo) Create(n notification_model.Notification) (int, error) {
	m.created = append(m.created, n)
	return 1, m.createErr
}

func (m *mockNotificationRepo) GetNotificationTypeByName(name string) (int, error) {
	return 1, m.getTypeErr
}

type mockS3Client struct {
	uploadedURL string
	failErr     error

	deleteErr       error
	deleteCalls     int
	deletedUserID   string
	deletedAvatarAt string
}

func (m *mockS3Client) DeleteAvatar(_ context.Context, userID, url string) error {
	m.deleteCalls++
	m.deletedUserID, m.deletedAvatarAt = userID, url
	return m.deleteErr
}

var _ s3_client.Client = (*mockS3Client)(nil)

func (m *mockS3Client) UploadAvatar(_ context.Context, userID, filename, contentType string, _ io.Reader) (string, error) {
	if m.failErr != nil {
		return "", m.failErr
	}
	if m.uploadedURL != "" {
		return m.uploadedURL, nil
	}
	return "https://my-bucket.s3.amazonaws.com/avatars/" + userID + "/" + filename, nil
}

func (m *mockS3Client) UploadEventImage(_ context.Context, _ int, _, _ string, _ io.Reader) (string, error) {
	return "", nil
}

func TestGetUserProfile_OwnProfile(t *testing.T) {
	repo := newMockUserRepo()
	u := user_model.User{
		ID:             testID(1),
		FirebaseUID:    "fb-user-1",
		Username:       "alice",
		Name:           "Alice Smith",
		Email:          "alice@example.com",
		ProfilePicture: "https://example.com/alice.png",
		Biography:      "Culture lover",
		DateRegistered: time.Now(),
		AccountState:   1,
		Experience:     120,
		Settings:       1,
	}
	repo.addUser(u, true)
	repo.plansCount[u.ID] = 5
	repo.completedPlans[u.ID] = 3

	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{}, nil)

	profile, err := svc.GetUserProfile(context.Background(), "me", "fb-user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if profile.ID != testID(1) {
		t.Errorf("expected ID 1, got %v", profile.ID)
	}
	if profile.Email != "alice@example.com" {
		t.Errorf("expected own email exposed, got %q", profile.Email)
	}
	if !profile.PlansVisible {
		t.Errorf("expected plans to be visible for self")
	}
	if profile.IsFriend != nil {
		t.Errorf("expected IsFriend to be nil for self")
	}
	if profile.PlansCount != 5 || profile.CompletedPlansCount != 3 {
		t.Errorf("unexpected counters: plans=%d, completed=%d", profile.PlansCount, profile.CompletedPlansCount)
	}
}

func TestGetUserProfile_OtherUser_FriendAndShowPlans(t *testing.T) {
	repo := newMockUserRepo()
	alice := user_model.User{
		ID:          testID(1),
		FirebaseUID: "fb-alice",
		Username:    "alice",
		Settings:    1,
	}
	bob := user_model.User{
		ID:          testID(2),
		FirebaseUID: "fb-bob",
		Username:    "bob",
		Settings:    2,
	}
	repo.addUser(alice, true)
	repo.addUser(bob, true)

	repo.friends[alice.ID] = map[pgtype.UUID]bool{bob.ID: true}

	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{}, nil)

	profile, err := svc.GetUserProfile(context.Background(), "bob", "fb-alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if profile.IsFriend == nil || !*profile.IsFriend {
		t.Errorf("expected IsFriend to be true")
	}
	if !profile.PlansVisible {
		t.Errorf("expected plans to be visible for friend with showPlans=true")
	}
	if profile.Email != "" {
		t.Errorf("expected email to be hidden for other user, got %q", profile.Email)
	}
}

func TestGetUserProfile_OtherUser_FriendAndHidePlans(t *testing.T) {
	repo := newMockUserRepo()
	alice := user_model.User{
		ID:          testID(1),
		FirebaseUID: "fb-alice",
		Username:    "alice",
		Settings:    1,
	}
	bob := user_model.User{
		ID:          testID(2),
		FirebaseUID: "fb-bob",
		Username:    "bob",
		Settings:    2,
	}
	repo.addUser(alice, true)
	repo.addUser(bob, false)

	repo.friends[alice.ID] = map[pgtype.UUID]bool{bob.ID: true}

	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{}, nil)

	profile, err := svc.GetUserProfile(context.Background(), "bob", "fb-alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if profile.IsFriend == nil || !*profile.IsFriend {
		t.Errorf("expected IsFriend to be true")
	}
	if profile.PlansVisible {
		t.Errorf("expected plans to NOT be visible when showPlans=false")
	}
}

func TestGetUserProfile_OtherUser_NotFriend(t *testing.T) {
	repo := newMockUserRepo()
	alice := user_model.User{
		ID:          testID(1),
		FirebaseUID: "fb-alice",
		Username:    "alice",
		Settings:    1,
	}
	bob := user_model.User{
		ID:          testID(2),
		FirebaseUID: "fb-bob",
		Username:    "bob",
		Settings:    2,
	}
	repo.addUser(alice, true)
	repo.addUser(bob, true)

	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{}, nil)

	profile, err := svc.GetUserProfile(context.Background(), "bob", "fb-alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if profile.IsFriend == nil || *profile.IsFriend {
		t.Errorf("expected IsFriend to be false")
	}
	if profile.PlansVisible {
		t.Errorf("expected plans to NOT be visible for non-friend")
	}
}

func TestGetUserProfile_Unauthenticated_NotVisible(t *testing.T) {
	repo := newMockUserRepo()
	bob := user_model.User{
		ID:          testID(2),
		FirebaseUID: "fb-bob",
		Username:    "bob",
		Settings:    2,
	}
	repo.addUser(bob, true)

	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{}, nil)

	profile, err := svc.GetUserProfile(context.Background(), "bob", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if profile.IsFriend != nil {
		t.Errorf("expected IsFriend to be nil for unauthenticated")
	}
	if profile.PlansVisible {
		t.Errorf("expected plans to NOT be visible for unauthenticated")
	}
}

func TestGetUserProfile_NotFound(t *testing.T) {
	repo := newMockUserRepo()
	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{}, nil)

	_, err := svc.GetUserProfile(context.Background(), "nonexistent", "")
	if err != user_service.ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUpdateProfile_Valid(t *testing.T) {
	repo := newMockUserRepo()
	u := user_model.User{
		ID:          testID(1),
		FirebaseUID: "fb-alice",
		Username:    "alice",
		Settings:    1,
	}
	repo.addUser(u, true)

	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{}, nil)

	newName := "Alice Wonderland"
	newBio := "Exploring wonderland"
	newShowPlans := false

	profile, err := svc.UpdateProfile(context.Background(), "fb-alice", user_service.UpdateProfileInput{
		Name:      &newName,
		Biography: &newBio,
		ShowPlans: &newShowPlans,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if profile.Name != newName {
		t.Errorf("expected name %q, got %q", newName, profile.Name)
	}
	if profile.Biography != newBio {
		t.Errorf("expected biography %q, got %q", newBio, profile.Biography)
	}
	if profile.ShowPlans != false {
		t.Errorf("expected showPlans to be updated to false")
	}
}

func TestUpdateProfile_R1_Validations(t *testing.T) {
	repo := newMockUserRepo()
	u := user_model.User{
		ID:          testID(1),
		FirebaseUID: "fb-alice",
		Username:    "alice",
		Settings:    1,
	}
	repo.addUser(u, true)

	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{}, nil)

	// Username with invalid characters
	invalidUser := "bad username!"
	_, err := svc.UpdateProfile(context.Background(), "fb-alice", user_service.UpdateProfileInput{
		Username: &invalidUser,
	})
	if !errors.Is(err, user_service.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for invalid username characters, got %v", err)
	}

	// Username too short
	shortUser := "ab"
	_, err = svc.UpdateProfile(context.Background(), "fb-alice", user_service.UpdateProfileInput{
		Username: &shortUser,
	})
	if !errors.Is(err, user_service.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for too short username, got %v", err)
	}

	// Name too long (> 100)
	longName := strings.Repeat("A", 101)
	_, err = svc.UpdateProfile(context.Background(), "fb-alice", user_service.UpdateProfileInput{
		Name: &longName,
	})
	if !errors.Is(err, user_service.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for too long name, got %v", err)
	}

	// Bio too long (> 500)
	longBio := strings.Repeat("B", 501)
	_, err = svc.UpdateProfile(context.Background(), "fb-alice", user_service.UpdateProfileInput{
		Biography: &longBio,
	})
	if !errors.Is(err, user_service.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for too long biography, got %v", err)
	}
}

func TestUploadAvatar_Success(t *testing.T) {
	repo := newMockUserRepo()
	u := user_model.User{
		ID:          testID(1),
		FirebaseUID: "fb-alice",
		Username:    "alice",
		Settings:    1,
	}
	repo.addUser(u, true)

	s3 := &mockS3Client{uploadedURL: "https://storage.example.com/avatar.png"}
	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, s3, nil)

	profile, err := svc.UploadAvatar(context.Background(), "fb-alice", "photo.png", "image/png", strings.NewReader("fake-image"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if profile.ProfilePicture != "https://storage.example.com/avatar.png" {
		t.Errorf("expected profile picture to be updated, got %s", profile.ProfilePicture)
	}
}

func TestListCreatedEvents(t *testing.T) {
	repo := newMockUserRepo()
	u := user_model.User{
		ID:          testID(1),
		FirebaseUID: "fb-alice",
		Username:    "alice",
		Settings:    1,
	}
	repo.addUser(u, true)
	repo.createdEvents[u.ID] = []event_model.Event{
		{ID: 10, Title: "Concert de Jazz"},
	}

	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{}, nil)

	events, err := svc.ListCreatedEvents(context.Background(), "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 || events[0].Title != "Concert de Jazz" {
		t.Errorf("unexpected created events: %+v", events)
	}
}

// testID builds a deterministic user UUID for the mocks.
func testID(n byte) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte{15: n}, Valid: true}
}

var errDatabase = errors.New("database error")

func TestGetUserFriends(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		result  []string
		repoErr error
		wantErr error
		wantLen int
	}{
		{"success", "fb-alice", []string{"bob", "charlie"}, nil, nil, 2},
		{"user not found", "unknown", nil, nil, user_service.ErrUserNotFound, 0},
		{"repository error", "fb-alice", nil, errDatabase, errDatabase, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockUserRepo()
			repo.addUser(user_model.User{ID: testID(1), FirebaseUID: "fb-alice"}, true)
			repo.getUserFriendsResult = tt.result
			repo.getUserFriendsErr = tt.repoErr

			svc := user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{}, nil)
			friends, err := svc.GetUserFriends(context.Background(), tt.userID)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
			if len(friends) != tt.wantLen {
				t.Errorf("expected %d friends, got %d", tt.wantLen, len(friends))
			}
		})
	}
}

func TestGetFriendRequests(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		result  []user_model.FriendRequest
		repoErr error
		wantErr error
		wantLen int
	}{
		{"success", "fb-alice", []user_model.FriendRequest{{}}, nil, nil, 1},
		{"user not found", "unknown", nil, nil, user_service.ErrUserNotFound, 0},
		{"repository error", "fb-alice", nil, errDatabase, errDatabase, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockUserRepo()
			repo.addUser(user_model.User{ID: testID(1), FirebaseUID: "fb-alice"}, true)
			repo.getFriendRequestsResult = tt.result
			repo.getFriendRequestsErr = tt.repoErr

			svc := user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{}, nil)
			requests, err := svc.GetFriendRequests(context.Background(), tt.userID)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
			if len(requests) != tt.wantLen {
				t.Errorf("expected %d requests, got %d", tt.wantLen, len(requests))
			}
		})
	}
}

func TestSendFriendRequest(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		targetID string
		friends  bool
		pending  bool
		blocked  bool
		repoErr  error
		wantErr  error
	}{
		{"success", "fb-alice", "fb-bob", false, false, false, nil, nil},
		{"user not found", "unknown", "fb-bob", false, false, false, nil, user_service.ErrUserNotFound},
		{"target not found", "fb-alice", "unknown", false, false, false, nil, user_service.ErrUserNotFound},
		{"same user", "fb-alice", "fb-alice", false, false, false, nil, user_service.ErrSameUser},
		{"already friends", "fb-alice", "fb-bob", true, false, false, nil, user_service.ErrAlreadyFriends},
		{"request already exists", "fb-alice", "fb-bob", false, true, false, nil, user_service.ErrRequestAlreadyExists},
		{"blocked", "fb-alice", "fb-bob", false, false, true, nil, user_service.ErrIsBlocked},
		{"are friends error", "fb-alice", "fb-bob", false, false, false, errDatabase, errDatabase},
		{"pending request error", "fb-alice", "fb-bob", false, false, false, errDatabase, errDatabase},
		{"blocked check error", "fb-alice", "fb-bob", false, false, false, errDatabase, errDatabase},
		{"send request error", "fb-alice", "fb-bob", false, false, false, errDatabase, errDatabase},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockUserRepo()
			repo.addUser(user_model.User{ID: testID(1), FirebaseUID: "fb-alice"}, true)
			repo.addUser(user_model.User{ID: testID(2), FirebaseUID: "fb-bob"}, true)

			if tt.friends {
				repo.friends[testID(1)] = map[pgtype.UUID]bool{testID(2): true}
			}
			repo.pendingRequestExists = tt.pending
			repo.isBlockedResult = tt.blocked

			switch tt.name {
			case "are friends error":
				repo.areFriendsErr = tt.repoErr
			case "pending request error":
				repo.pendingRequestErr = tt.repoErr
			case "blocked check error":
				repo.isBlockedErr = tt.repoErr
			case "send request error":
				repo.sendFriendRequestErr = tt.repoErr
			}

			svc := user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{}, nil)
			_, err := svc.SendFriendRequest(context.Background(), tt.userID, tt.targetID)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestFriendRequestActions(t *testing.T) {
	actions := []struct {
		name   string
		call   func(user_service.Service, string, string, time.Time) (user_model.FriendRequest, error)
		setErr func(*mockUserRepo, error)
	}{
		{
			"accept",
			func(s user_service.Service, u, target string, d time.Time) (user_model.FriendRequest, error) {
				return s.AcceptFriendRequest(context.Background(), u, target, d)
			},
			func(r *mockUserRepo, err error) { r.acceptFriendRequestErr = err },
		},
		{
			"deny",
			func(s user_service.Service, u, target string, d time.Time) (user_model.FriendRequest, error) {
				return s.DenyFriendRequest(context.Background(), u, target, d)
			},
			func(r *mockUserRepo, err error) { r.denyFriendRequestErr = err },
		},
	}

	tests := []struct {
		name     string
		userID   string
		targetID string
		repoErr  error
		wantErr  error
	}{
		{"success", "fb-alice", "fb-bob", nil, nil},
		{"user not found", "unknown", "fb-bob", nil, user_service.ErrUserNotFound},
		{"target not found", "fb-alice", "unknown", nil, user_service.ErrUserNotFound},
		{"same user", "fb-alice", "fb-alice", nil, user_service.ErrSameUser},
		{"request not found", "fb-alice", "fb-bob", pgx.ErrNoRows, user_service.ErrRequestNotFound},
		{"repository error", "fb-alice", "fb-bob", errDatabase, errDatabase},
	}

	for _, action := range actions {
		t.Run(action.name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					repo := newMockUserRepo()
					notifRepo := &mockNotificationRepo{}

					repo.addUser(user_model.User{ID: testID(1), FirebaseUID: "fb-alice", Username: "alice"}, true)
					repo.addUser(user_model.User{ID: testID(2), FirebaseUID: "fb-bob", Username: "bob"}, true)

					if tt.repoErr != nil {
						action.setErr(repo, tt.repoErr)
					}

					svc := user_service.NewUserService(repo, notifRepo, &mockS3Client{}, nil)
					_, err := action.call(svc, tt.userID, tt.targetID, time.Now())

					if !errors.Is(err, tt.wantErr) {
						t.Errorf("expected error %v, got %v", tt.wantErr, err)
					}

					if tt.wantErr == nil && len(notifRepo.created) != 1 {
						t.Errorf("expected 1 notification, got %d", len(notifRepo.created))
					}
				})
			}
		})
	}
}

func TestUnfriend(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		targetID string
		friends  bool
		repoErr  error
		wantErr  error
	}{
		{"success", "fb-alice", "fb-bob", true, nil, nil},
		{"user not found", "unknown", "fb-bob", false, nil, user_service.ErrUserNotFound},
		{"target not found", "fb-alice", "unknown", false, nil, user_service.ErrUserNotFound},
		{"same user", "fb-alice", "fb-alice", false, nil, user_service.ErrSameUser},
		{"not friends", "fb-alice", "fb-bob", false, nil, user_service.ErrNotFriends},
		{"are friends error", "fb-alice", "fb-bob", false, errDatabase, errDatabase},
		{"unfriend error", "fb-alice", "fb-bob", true, errDatabase, errDatabase},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockUserRepo()
			notifRepo := &mockNotificationRepo{}

			repo.addUser(user_model.User{ID: testID(1), FirebaseUID: "fb-alice", Username: "alice"}, true)
			repo.addUser(user_model.User{ID: testID(2), FirebaseUID: "fb-bob", Username: "bob"}, true)

			if tt.friends {
				repo.friends[testID(1)] = map[pgtype.UUID]bool{testID(2): true}
			}

			if tt.name == "are friends error" {
				repo.areFriendsErr = tt.repoErr
			} else {
				repo.unfriendErr = tt.repoErr
			}

			svc := user_service.NewUserService(repo, notifRepo, &mockS3Client{}, nil)
			err := svc.Unfriend(context.Background(), tt.userID, tt.targetID)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}

			if tt.wantErr == nil && len(notifRepo.created) != 1 {
				t.Errorf("expected 1 notification, got %d", len(notifRepo.created))
			}
		})
	}
}

func TestFriendActions_NotificationFailure(t *testing.T) {
	tests := []struct {
		name       string
		getTypeErr error
		createErr  error
	}{
		{"type lookup fails", errors.New("type error"), nil},
		{"creation fails", nil, errors.New("creation error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockUserRepo()
			alice := user_model.User{ID: testID(1), FirebaseUID: "fb-alice", Username: "alice"}
			bob := user_model.User{ID: testID(2), FirebaseUID: "fb-bob", Username: "bob"}

			repo.addUser(alice, true)
			repo.addUser(bob, true)
			repo.friends[alice.ID] = map[pgtype.UUID]bool{bob.ID: true}

			notifRepo := &mockNotificationRepo{
				getTypeErr: tt.getTypeErr,
				createErr:  tt.createErr,
			}

			svc := user_service.NewUserService(repo, notifRepo, &mockS3Client{}, nil)

			_, acceptErr := svc.AcceptFriendRequest(context.Background(), "fb-alice", "fb-bob", time.Now())
			if acceptErr != nil {
				t.Errorf("accept should succeed despite notification failure: %v", acceptErr)
			}

			_, denyErr := svc.DenyFriendRequest(context.Background(), "fb-alice", "fb-bob", time.Now())
			if denyErr != nil {
				t.Errorf("deny should succeed despite notification failure: %v", denyErr)
			}

			err := svc.Unfriend(context.Background(), "fb-alice", "fb-bob")
			if err != nil {
				t.Errorf("unfriend should succeed despite notification failure: %v", err)
			}
		})
	}
}
