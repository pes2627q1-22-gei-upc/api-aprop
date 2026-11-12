package user_service_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/clients/s3_client"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/notification_model"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/user_repository"
	"api-aprop/internal/services/user_service"
)

type mockUserRepo struct {
	usersByID      map[pgtype.UUID]*user_model.User
	showPlans      map[int]bool
	friends        map[pgtype.UUID]map[pgtype.UUID]bool
	plansCount     map[pgtype.UUID]int
	completedPlans map[pgtype.UUID]int
	reliability    map[pgtype.UUID]*float32
	createdEvents  map[pgtype.UUID][]event_model.Event
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

func (m *mockUserRepo) GetUserFriends(ctx context.Context, userid pgtype.UUID) ([]string, error) {
	return []string{}, nil
}
func (m *mockUserRepo) GetFriendRequests(ctx context.Context, userid pgtype.UUID) ([]user_model.FriendRequest, error) {
	return []user_model.FriendRequest{}, nil
}
func (m *mockUserRepo) PendingRequestExists(ctx context.Context, userid pgtype.UUID, targetid pgtype.UUID) (bool, error) {
	return false, nil
}
func (m *mockUserRepo) IsBlocked(ctx context.Context, userid pgtype.UUID, targetid pgtype.UUID) (bool, error) {
	return false, nil
}
func (m *mockUserRepo) SendFriendRequest(ctx context.Context, userid pgtype.UUID, targetid pgtype.UUID) (user_model.FriendRequest, error) {
	return user_model.FriendRequest{}, nil
}
func (m *mockUserRepo) AcceptFriendRequest(ctx context.Context, userid pgtype.UUID, targetid pgtype.UUID, date time.Time) (user_model.FriendRequest, error) {
	return user_model.FriendRequest{}, nil
}
func (m *mockUserRepo) DenyFriendRequest(ctx context.Context, userid pgtype.UUID, targetid pgtype.UUID, date time.Time) (user_model.FriendRequest, error) {
	return user_model.FriendRequest{}, nil
}
func (m *mockUserRepo) Unfriend(ctx context.Context, userid pgtype.UUID, targetuuid pgtype.UUID) error {
	return nil
}

type mockNotificationRepo struct {
}

func (m *mockNotificationRepo) Get(id int) (notification_model.Notification, error) {
	return notification_model.Notification{}, nil
}
func (m *mockNotificationRepo) Create(notification_model.Notification) (int, error) { return 0, nil }
func (m *mockNotificationRepo) GetUserNotifications(id pgtype.UUID, params map[string]any) ([]notification_model.Notification, error) {
	return []notification_model.Notification{}, nil
}
func (m *mockNotificationRepo) ReadNotification(id int) error              { return nil }
func (m *mockNotificationRepo) ReadUserNotifications(id pgtype.UUID) error { return nil }
func (m *mockNotificationRepo) GetNotificationTypes() ([]notification_model.NotificationType, error) {
	return []notification_model.NotificationType{}, nil
}
func (m *mockNotificationRepo) GetNotificationTypeByName(name string) (int, error) { return 0, nil }
func (m *mockNotificationRepo) CreateReminders() ([]int32, error)                  { return []int32{}, nil }
func (f *mockNotificationRepo) CreateFriendSignupNotifications(id pgtype.UUID, event int, text string) ([]int32, error) {
	return []int32{}, nil
}

type mockS3Client struct {
	uploadedURL string
	failErr     error
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

	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{})

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

	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{})

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

	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{})

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

	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{})

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

	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{})

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
	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{})

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

	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{})

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

	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{})

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
	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, s3)

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

	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{})

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
