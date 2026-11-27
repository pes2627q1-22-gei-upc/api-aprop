package auth_service

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/auth_repository"
)

const firebaseUID = "Vw3kU1pZ8rQ2hXyT6mN9aB4cD7e1"

type fakeRepository struct {
	users     map[string]user_model.User // by Firebase UID
	createErr error
	// createdBySomeoneElse simulates a concurrent request that created the user right before Create
	createdBySomeoneElse *user_model.User

	createCalls      int
	updateEmailCalls int
	gotUsername      string
}

func newFakeRepository(users ...user_model.User) *fakeRepository {
	f := &fakeRepository{users: map[string]user_model.User{}}
	for _, u := range users {
		f.users[u.FirebaseUID] = u
	}
	return f
}

func (f *fakeRepository) GetByFirebaseUID(_ context.Context, uid string) (user_model.User, error) {
	user, ok := f.users[uid]
	if !ok {
		return user_model.User{}, auth_repository.ErrUserNotFound
	}
	return user, nil
}

func (f *fakeRepository) Create(_ context.Context, uid, username, email string) (user_model.User, error) {
	f.createCalls++
	f.gotUsername = username
	if f.createdBySomeoneElse != nil {
		f.users[uid] = *f.createdBySomeoneElse
		return user_model.User{}, auth_repository.ErrUserAlreadyExists
	}
	if f.createErr != nil {
		return user_model.User{}, f.createErr
	}
	user := user_model.User{ID: testID(byte(len(f.users) + 1)), FirebaseUID: uid, Username: username, Email: email}
	f.users[uid] = user
	return user, nil
}

func (f *fakeRepository) UpdateEmail(_ context.Context, id pgtype.UUID, email string) (user_model.User, error) {
	f.updateEmailCalls++
	for uid, u := range f.users {
		if u.ID == id {
			u.Email = email
			f.users[uid] = u
			return u, nil
		}
	}
	return user_model.User{}, auth_repository.ErrUserNotFound
}

func TestSync_CreatesNewUser(t *testing.T) {
	repo := newFakeRepository()

	result, err := NewAuthService(repo).Sync(context.Background(), firebaseUID, "anna@example.com", "  anna  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Created {
		t.Error("expected Created to be true")
	}
	if repo.gotUsername != "anna" {
		t.Errorf("username = %q, want it trimmed to %q", repo.gotUsername, "anna")
	}
	if result.User.Email != "anna@example.com" {
		t.Errorf("email = %q", result.User.Email)
	}
}

func TestSync_ExistingUserIsNotCreated(t *testing.T) {
	existing := user_model.User{ID: testID(7), FirebaseUID: firebaseUID, Username: "anna", Email: "anna@example.com"}
	repo := newFakeRepository(existing)

	// The username is ignored for existing users, even if it is empty
	result, err := NewAuthService(repo).Sync(context.Background(), firebaseUID, "anna@example.com", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Created || repo.createCalls != 0 {
		t.Errorf("Created = %v, createCalls = %d; want an existing user", result.Created, repo.createCalls)
	}
	if repo.updateEmailCalls != 0 {
		t.Errorf("updateEmailCalls = %d, want 0 when the email did not change", repo.updateEmailCalls)
	}
	if result.User.ID != testID(7) {
		t.Errorf("user id = %v, want 7", result.User.ID)
	}
}

func TestSync_UpdatesEmailChangedInFirebase(t *testing.T) {
	repo := newFakeRepository(user_model.User{ID: testID(7), FirebaseUID: firebaseUID, Username: "anna", Email: "old@example.com"})

	result, err := NewAuthService(repo).Sync(context.Background(), firebaseUID, "new@example.com", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updateEmailCalls != 1 || result.User.Email != "new@example.com" {
		t.Errorf("updateEmailCalls = %d, email = %q; want the new email stored", repo.updateEmailCalls, result.User.Email)
	}
}

func TestSync_InvalidUsername(t *testing.T) {
	for _, username := range []string{"", "  ", "ab", "this-username-is-way-too-long-for-us"} {
		repo := newFakeRepository()
		_, err := NewAuthService(repo).Sync(context.Background(), firebaseUID, "", username)
		if !errors.Is(err, ErrInvalidUsername) {
			t.Errorf("username %q: err = %v, want ErrInvalidUsername", username, err)
		}
		if repo.createCalls != 0 {
			t.Errorf("username %q: user should not be created", username)
		}
	}
}

func TestSync_UsernameTaken(t *testing.T) {
	repo := newFakeRepository()
	repo.createErr = auth_repository.ErrUsernameTaken

	_, err := NewAuthService(repo).Sync(context.Background(), firebaseUID, "", "anna")
	if !errors.Is(err, ErrUsernameTaken) {
		t.Errorf("err = %v, want ErrUsernameTaken", err)
	}
}

func TestSync_ConcurrentFirstSignIn(t *testing.T) {
	repo := newFakeRepository()
	repo.createdBySomeoneElse = &user_model.User{ID: testID(3), FirebaseUID: firebaseUID, Username: "anna"}

	result, err := NewAuthService(repo).Sync(context.Background(), firebaseUID, "", "anna")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Created || result.User.ID != testID(3) {
		t.Errorf("Created = %v, id = %v; want the user created by the other request", result.Created, result.User.ID)
	}
}

// testID builds a deterministic user UUID for the fakes.
func testID(n byte) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte{15: n}, Valid: true}
}

func TestResolveUserID(t *testing.T) {
	registered := user_model.User{ID: testID(7), FirebaseUID: firebaseUID}
	const userID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"

	tests := []struct {
		name       string
		identifier string
		want       string
		wantErr    error
	}{
		{name: "uuid is used as it is", identifier: "3F2504E0-4F89-11D3-9A0C-0305E82C3301", want: userID},
		{name: "firebase uid is resolved to the user id", identifier: firebaseUID, want: registered.ID.String()},
		{name: "unknown firebase uid", identifier: "aBcDeFgHiJkLmNoPqRsTuVwXyZ12", wantErr: ErrUserNotRegistered},
		{name: "missing identifier", wantErr: ErrUserNotRegistered},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewAuthService(newFakeRepository(registered)).ResolveUserID(context.Background(), tt.identifier)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("user id = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveUserID_RepositoryError(t *testing.T) {
	repoErr := errors.New("db down")
	_, err := NewAuthService(&failingRepository{err: repoErr}).ResolveUserID(context.Background(), firebaseUID)
	if !errors.Is(err, repoErr) || errors.Is(err, ErrUserNotRegistered) {
		t.Fatalf("error = %v, want the repository error", err)
	}
}

// failingRepository fails every lookup, like a database that is down.
type failingRepository struct {
	fakeRepository
	err error
}

func (f *failingRepository) GetByFirebaseUID(_ context.Context, _ string) (user_model.User, error) {
	return user_model.User{}, f.err
}
