package user_service_test

import (
	"context"
	"testing"
	"time"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/user_repository"
	"api-aprop/internal/services/user_service"
)

type mockUserRepo struct {
	users          map[string]user_model.User // indexed by id, uuid, username, firebase_uid
	showPlans      map[int]bool               // settingsID -> showPlans
	friends        map[string]map[string]bool // uuid1 -> uuid2 -> areFriends
	plansCount     map[string]int
	completedPlans map[string]int
	reliability    map[string]*float32
	createdEvents  map[string][]event_model.Event
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:          make(map[string]user_model.User),
		showPlans:      make(map[int]bool),
		friends:        make(map[string]map[string]bool),
		plansCount:     make(map[string]int),
		completedPlans: make(map[string]int),
		reliability:    make(map[string]*float32),
		createdEvents:  make(map[string][]event_model.Event),
	}
}

func (m *mockUserRepo) addUser(u user_model.User, showPlans bool) {
	m.users[u.Username] = u
	m.users[u.FirebaseUID] = u
	if u.UUID != "" {
		m.users[u.UUID] = u
	}
	m.users[string(rune(u.ID))] = u
	m.showPlans[u.Settings] = showPlans
}

func (m *mockUserRepo) GetByID(_ context.Context, id int) (user_model.User, bool, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, m.showPlans[u.Settings], nil
		}
	}
	return user_model.User{}, false, user_repository.ErrUserNotFound
}

func (m *mockUserRepo) GetByUsername(_ context.Context, username string) (user_model.User, bool, error) {
	u, ok := m.users[username]
	if !ok {
		return user_model.User{}, false, user_repository.ErrUserNotFound
	}
	return u, m.showPlans[u.Settings], nil
}

func (m *mockUserRepo) GetByFirebaseUID(_ context.Context, firebaseUID string) (user_model.User, bool, error) {
	u, ok := m.users[firebaseUID]
	if !ok {
		return user_model.User{}, false, user_repository.ErrUserNotFound
	}
	return u, m.showPlans[u.Settings], nil
}

func (m *mockUserRepo) GetByUUID(_ context.Context, uuid string) (user_model.User, bool, error) {
	u, ok := m.users[uuid]
	if !ok {
		return user_model.User{}, false, user_repository.ErrUserNotFound
	}
	return u, m.showPlans[u.Settings], nil
}

func (m *mockUserRepo) CountFriends(_ context.Context, userUUID string) (int, error) {
	fMap := m.friends[userUUID]
	count := 0
	for _, areFriends := range fMap {
		if areFriends {
			count++
		}
	}
	return count, nil
}

func (m *mockUserRepo) AreFriends(_ context.Context, user1UUID, user2UUID string) (bool, error) {
	if m.friends[user1UUID] != nil && m.friends[user1UUID][user2UUID] {
		return true, nil
	}
	if m.friends[user2UUID] != nil && m.friends[user2UUID][user1UUID] {
		return true, nil
	}
	return false, nil
}

func (m *mockUserRepo) CountPlans(_ context.Context, userUUID string) (int, error) {
	return m.plansCount[userUUID], nil
}

func (m *mockUserRepo) CountCompletedPlans(_ context.Context, userUUID string) (int, error) {
	return m.completedPlans[userUUID], nil
}

func (m *mockUserRepo) GetReliability(_ context.Context, creatorUUID string) (*float32, error) {
	return m.reliability[creatorUUID], nil
}

func (m *mockUserRepo) ListCreatedEvents(_ context.Context, creatorUUID string) ([]event_model.Event, error) {
	return m.createdEvents[creatorUUID], nil
}

func (m *mockUserRepo) UpdateProfile(_ context.Context, id int, name, username, biography, profilePicture *string) (user_model.User, error) {
	var target *user_model.User
	for _, u := range m.users {
		if u.ID == id {
			copy := u
			target = &copy
			break
		}
	}
	if target == nil {
		return user_model.User{}, user_repository.ErrUserNotFound
	}
	if username != nil && *username != target.Username {
		if _, exists := m.users[*username]; exists {
			return user_model.User{}, user_repository.ErrUsernameTaken
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
	m.users[target.Username] = *target
	m.users[target.FirebaseUID] = *target
	if target.UUID != "" {
		m.users[target.UUID] = *target
	}
	return *target, nil
}

func (m *mockUserRepo) UpdateSettings(_ context.Context, settingsID int, showPlans bool) error {
	m.showPlans[settingsID] = showPlans
	return nil
}

func TestGetUserProfile_OwnProfile(t *testing.T) {
	repo := newMockUserRepo()
	u := user_model.User{
		ID:             1,
		UUID:           "550e8400-e29b-41d4-a716-446655440001",
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
	repo.plansCount[u.UUID] = 5
	repo.completedPlans[u.UUID] = 3

	svc := user_service.NewUserService(repo)

	// Fetching "me"
	profile, err := svc.GetUserProfile(context.Background(), "me", "fb-user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if profile.ID != 1 {
		t.Errorf("expected ID 1, got %d", profile.ID)
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
		ID:          1,
		UUID:        "550e8400-e29b-41d4-a716-446655440001",
		FirebaseUID: "fb-alice",
		Username:    "alice",
		Settings:    1,
	}
	bob := user_model.User{
		ID:          2,
		UUID:        "550e8400-e29b-41d4-a716-446655440002",
		FirebaseUID: "fb-bob",
		Username:    "bob",
		Settings:    2,
	}
	repo.addUser(alice, true)
	repo.addUser(bob, true) // Bob has showPlans = true

	// Set friendship
	repo.friends[alice.UUID] = map[string]bool{bob.UUID: true}

	svc := user_service.NewUserService(repo)

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
		ID:          1,
		UUID:        "550e8400-e29b-41d4-a716-446655440001",
		FirebaseUID: "fb-alice",
		Username:    "alice",
		Settings:    1,
	}
	bob := user_model.User{
		ID:          2,
		UUID:        "550e8400-e29b-41d4-a716-446655440002",
		FirebaseUID: "fb-bob",
		Username:    "bob",
		Settings:    2,
	}
	repo.addUser(alice, true)
	repo.addUser(bob, false) // Bob hides plans!

	repo.friends[alice.UUID] = map[string]bool{bob.UUID: true}

	svc := user_service.NewUserService(repo)

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
		ID:          1,
		UUID:        "550e8400-e29b-41d4-a716-446655440001",
		FirebaseUID: "fb-alice",
		Username:    "alice",
		Settings:    1,
	}
	bob := user_model.User{
		ID:          2,
		UUID:        "550e8400-e29b-41d4-a716-446655440002",
		FirebaseUID: "fb-bob",
		Username:    "bob",
		Settings:    2,
	}
	repo.addUser(alice, true)
	repo.addUser(bob, true) // Bob allows plans for friends, but Alice is NOT a friend

	svc := user_service.NewUserService(repo)

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
		ID:          2,
		UUID:        "550e8400-e29b-41d4-a716-446655440002",
		FirebaseUID: "fb-bob",
		Username:    "bob",
		Settings:    2,
	}
	repo.addUser(bob, true)

	svc := user_service.NewUserService(repo)

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
	svc := user_service.NewUserService(repo)

	_, err := svc.GetUserProfile(context.Background(), "nonexistent", "")
	if err != user_service.ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUpdateProfile(t *testing.T) {
	repo := newMockUserRepo()
	u := user_model.User{
		ID:          1,
		UUID:        "550e8400-e29b-41d4-a716-446655440001",
		FirebaseUID: "fb-alice",
		Username:    "alice",
		Settings:    1,
	}
	repo.addUser(u, true)

	svc := user_service.NewUserService(repo)

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

func TestListCreatedEvents(t *testing.T) {
	repo := newMockUserRepo()
	u := user_model.User{
		ID:          1,
		UUID:        "550e8400-e29b-41d4-a716-446655440001",
		FirebaseUID: "fb-alice",
		Username:    "alice",
		Settings:    1,
	}
	repo.addUser(u, true)
	repo.createdEvents[u.UUID] = []event_model.Event{
		{ID: 10, Title: "Concert de Jazz"},
	}

	svc := user_service.NewUserService(repo)

	events, err := svc.ListCreatedEvents(context.Background(), "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 || events[0].Title != "Concert de Jazz" {
		t.Errorf("unexpected created events: %+v", events)
	}
}
