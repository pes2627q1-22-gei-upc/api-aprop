package password_reset_service

import (
	"context"
	"errors"
	"testing"
	"time"

	"api-aprop/internal/clients/firebase_client"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/user_repository"
)

type fakeUsers struct {
	users   map[string]user_model.User
	err     error
	lookups int
}

func (f *fakeUsers) GetByUsername(_ context.Context, username string) (user_model.User, error) {
	f.lookups++
	if f.err != nil {
		return user_model.User{}, f.err
	}
	u, ok := f.users[username]
	if !ok {
		return user_model.User{}, user_repository.ErrUserNotFound
	}
	return u, nil
}

type fakeSender struct {
	err    error
	sent   []string
	locale string
}

func (f *fakeSender) SendPasswordResetEmail(_ context.Context, email, locale string) error {
	f.sent = append(f.sent, email)
	f.locale = locale
	return f.err
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }
func newClock() *clock                   { return &clock{t: time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)} }
func laia() map[string]user_model.User {
	return map[string]user_model.User{"laiasoler": {Username: "laiasoler", Email: "Laia@Aprop.cat"}}
}
func newService(u *fakeUsers, s *fakeSender, c *clock) *PasswordResetService {
	return NewPasswordResetService(u, s, c.now)
}

func TestRequestResetWithEmail(t *testing.T) {
	users, sender := &fakeUsers{}, &fakeSender{}
	svc := newService(users, sender, newClock())

	if err := svc.RequestReset(context.Background(), "  Laia@Aprop.CAT "); err != nil {
		t.Fatalf("RequestReset: %v", err)
	}
	if len(sender.sent) != 1 || sender.sent[0] != "laia@aprop.cat" {
		t.Errorf("sent = %v, want [laia@aprop.cat]", sender.sent)
	}
	if sender.locale != "ca" {
		t.Errorf("locale = %q, want ca", sender.locale)
	}
	if users.lookups != 0 {
		t.Errorf("an email must not look for the username (lookups = %d)", users.lookups)
	}
}

func TestRequestResetWithUsername(t *testing.T) {
	sender := &fakeSender{}
	svc := newService(&fakeUsers{users: laia()}, sender, newClock())

	if err := svc.RequestReset(context.Background(), "LaiaSoler"); err != nil {
		t.Fatalf("RequestReset: %v", err)
	}
	if len(sender.sent) != 1 || sender.sent[0] != "laia@aprop.cat" {
		t.Errorf("sent = %v, want [laia@aprop.cat]", sender.sent)
	}
}

func TestRequestResetUnknownAccountsAreNotReported(t *testing.T) {
	t.Run("unknown username", func(t *testing.T) {
		sender := &fakeSender{}
		svc := newService(&fakeUsers{users: laia()}, sender, newClock())
		if err := svc.RequestReset(context.Background(), "nobody"); err != nil {
			t.Errorf("err = %v, want nil", err)
		}
		if len(sender.sent) != 0 {
			t.Errorf("sent = %v, want none", sender.sent)
		}
	})
	t.Run("unknown email", func(t *testing.T) {
		svc := newService(&fakeUsers{}, &fakeSender{err: firebase_client.ErrEmailNotFound}, newClock())
		if err := svc.RequestReset(context.Background(), "nobody@aprop.cat"); err != nil {
			t.Errorf("err = %v, want nil", err)
		}
	})
	t.Run("username without email", func(t *testing.T) {
		sender := &fakeSender{}
		users := &fakeUsers{users: map[string]user_model.User{"pol": {Username: "pol"}}}
		if err := newService(users, sender, newClock()).RequestReset(context.Background(), "pol"); err != nil {
			t.Errorf("err = %v, want nil", err)
		}
		if len(sender.sent) != 0 {
			t.Errorf("sent = %v, want none", sender.sent)
		}
	})
	t.Run("Firebase limit of a username", func(t *testing.T) {
		svc := newService(&fakeUsers{users: laia()}, &fakeSender{err: firebase_client.ErrTooManyAttempts}, newClock())
		if err := svc.RequestReset(context.Background(), "laiasoler"); err != nil {
			t.Errorf("err = %v, want nil", err)
		}
	})
}

func TestRequestResetInvalidIdentifier(t *testing.T) {
	long := make([]byte, maxIdentifierLength+1)
	for i := range long {
		long[i] = 'a'
	}
	for _, identifier := range []string{"", "   ", "laia@", "@aprop.cat", "laia@aprop", "la ia@aprop.cat", string(long)} {
		sender := &fakeSender{}
		svc := newService(&fakeUsers{}, sender, newClock())
		if err := svc.RequestReset(context.Background(), identifier); !errors.Is(err, ErrInvalidIdentifier) {
			t.Errorf("RequestReset(%q) = %v, want ErrInvalidIdentifier", identifier, err)
		}
		if len(sender.sent) != 0 {
			t.Errorf("RequestReset(%q) sent an email", identifier)
		}
	}
}

func TestRequestResetFirebaseInvalidEmail(t *testing.T) {
	svc := newService(&fakeUsers{}, &fakeSender{err: firebase_client.ErrInvalidEmail}, newClock())
	if err := svc.RequestReset(context.Background(), "laia@aprop.cat"); !errors.Is(err, ErrInvalidIdentifier) {
		t.Errorf("err = %v, want ErrInvalidIdentifier", err)
	}
}

func TestRequestResetRateLimit(t *testing.T) {
	c := newClock()
	sender := &fakeSender{}
	svc := newService(&fakeUsers{}, sender, c)
	ctx := context.Background()

	if err := svc.RequestReset(ctx, "laia@aprop.cat"); err != nil {
		t.Fatalf("first request: %v", err)
	}

	c.advance(20 * time.Second)
	err := svc.RequestReset(ctx, "LAIA@aprop.cat")
	var limited *RateLimitedError
	if !errors.As(err, &limited) || !errors.Is(err, ErrRateLimited) {
		t.Fatalf("second request = %v, want ErrRateLimited", err)
	}
	if limited.RetryAfter != 40*time.Second {
		t.Errorf("RetryAfter = %s, want 40s", limited.RetryAfter)
	}

	c.advance(40 * time.Second)
	if err := svc.RequestReset(ctx, "laia@aprop.cat"); err != nil {
		t.Errorf("request after the window: %v", err)
	}
	if len(sender.sent) != 2 {
		t.Errorf("sent %d emails, want 2", len(sender.sent))
	}
}

func TestRequestResetRateLimitIsTheSameForUnknownAccounts(t *testing.T) {
	users := &fakeUsers{users: laia()}
	svc := newService(users, &fakeSender{}, newClock())
	ctx := context.Background()

	_ = svc.RequestReset(ctx, "nobody")
	if err := svc.RequestReset(ctx, "nobody"); !errors.Is(err, ErrRateLimited) {
		t.Errorf("err = %v, want ErrRateLimited", err)
	}
	if users.lookups != 1 {
		t.Errorf("lookups = %d, want 1: the limit is checked before looking for the user", users.lookups)
	}
}

func TestRequestResetUsernameAndEmailShareTheEmailLimit(t *testing.T) {
	sender := &fakeSender{}
	svc := newService(&fakeUsers{users: laia()}, sender, newClock())
	ctx := context.Background()

	if err := svc.RequestReset(ctx, "laia@aprop.cat"); err != nil {
		t.Fatal(err)
	}
	// Reporting the limit here would tell that the username exists
	if err := svc.RequestReset(ctx, "laiasoler"); err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if len(sender.sent) != 1 {
		t.Errorf("sent %d emails, want 1", len(sender.sent))
	}
}

func TestRequestResetFirebaseLimit(t *testing.T) {
	svc := newService(&fakeUsers{}, &fakeSender{err: firebase_client.ErrTooManyAttempts}, newClock())
	if err := svc.RequestReset(context.Background(), "laia@aprop.cat"); !errors.Is(err, ErrRateLimited) {
		t.Errorf("err = %v, want ErrRateLimited", err)
	}
}

func TestRequestResetProviderFailureAllowsRetrying(t *testing.T) {
	sender := &fakeSender{err: errors.New("status 500")}
	svc := newService(&fakeUsers{}, sender, newClock())
	ctx := context.Background()

	if err := svc.RequestReset(ctx, "laia@aprop.cat"); !errors.Is(err, ErrProvider) {
		t.Fatalf("err = %v, want ErrProvider", err)
	}
	sender.err = nil
	if err := svc.RequestReset(ctx, "laia@aprop.cat"); err != nil {
		t.Errorf("retry after a provider failure = %v, want nil", err)
	}
}

func TestRequestResetUserLookupFailure(t *testing.T) {
	svc := newService(&fakeUsers{err: errors.New("db down")}, &fakeSender{}, newClock())
	err := svc.RequestReset(context.Background(), "laiasoler")
	if err == nil || errors.Is(err, ErrRateLimited) {
		t.Errorf("err = %v, want an internal error", err)
	}
}

func TestRequestResetNotConfigured(t *testing.T) {
	svc := NewPasswordResetService(&fakeUsers{}, nil, newClock().now)
	if err := svc.RequestReset(context.Background(), "laia@aprop.cat"); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("err = %v, want ErrNotConfigured", err)
	}
}

func TestRateLimiterForgetsOldRequests(t *testing.T) {
	c := newClock()
	svc := newService(&fakeUsers{}, &fakeSender{}, c)
	for i := range pruneThreshold {
		svc.allow(string(rune('a'+i%26)) + time.Duration(i).String())
	}
	c.advance(Window)
	svc.allow("new")
	if len(svc.requests) != 1 {
		t.Errorf("remembered requests = %d, want 1", len(svc.requests))
	}
}
