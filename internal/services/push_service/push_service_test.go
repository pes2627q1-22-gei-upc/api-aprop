package push_service

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"api-aprop/internal/clients/firebase_client"
	"api-aprop/internal/models/push_model"
	"api-aprop/internal/repositories/push_repository"
)

const (
	userA = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	userB = "3f2504e0-4f89-11d3-9a0c-0305e82c3302"
)

// fakeRepo keeps the devices and an in-memory outbox: ProcessPending claims the first pending notifications.
type fakeRepo struct {
	devices    map[string][]string // user -> tokens
	platforms  map[string]string   // token -> platform
	pending    []push_model.PendingPush
	pushed     map[int]bool
	deleted    []string
	expiredAge time.Duration
	tokensErr  error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{devices: map[string][]string{}, platforms: map[string]string{}, pushed: map[int]bool{}}
}

func (f *fakeRepo) UpsertDevice(_ context.Context, userID string, d push_model.Device) error {
	f.devices[userID] = append(f.devices[userID], d.Token)
	f.platforms[d.Token] = d.Platform
	return nil
}

func (f *fakeRepo) DeleteDevice(_ context.Context, userID, token string) (bool, error) {
	before := len(f.devices[userID])
	f.devices[userID] = without(f.devices[userID], []string{token})
	return len(f.devices[userID]) < before, nil
}

func (f *fakeRepo) DeleteDeviceTokens(_ context.Context, tokens []string) error {
	f.deleted = append(f.deleted, tokens...)
	for user := range f.devices {
		f.devices[user] = without(f.devices[user], tokens)
	}
	return nil
}

func (f *fakeRepo) DeviceTokens(_ context.Context, userIDs []string) (map[string][]string, error) {
	if f.tokensErr != nil {
		return nil, f.tokensErr
	}
	out := map[string][]string{}
	for _, id := range userIDs {
		if tokens := f.devices[id]; len(tokens) > 0 {
			out[id] = append([]string{}, tokens...)
		}
	}
	return out, nil
}

func (f *fakeRepo) ExpirePending(_ context.Context, maxAge time.Duration) (int64, error) {
	f.expiredAge = maxAge
	return 0, nil
}

func (f *fakeRepo) ProcessPending(ctx context.Context, limit int, send push_repository.SendFunc) (int, error) {
	var batch []push_model.PendingPush
	for _, p := range f.pending {
		if !f.pushed[p.ID] && len(batch) < limit {
			batch = append(batch, p)
		}
	}
	if len(batch) == 0 {
		return 0, nil
	}
	resolved, err := send(ctx, batch)
	if err != nil {
		return 0, err // rollback: nothing is marked
	}
	for _, id := range resolved {
		f.pushed[id] = true
	}
	return len(batch), nil
}

// fakeSender answers each token according to its prefix: "bad" is invalid, "down" fails transiently, else it is ok.
type fakeSender struct {
	err  error
	sent []sentPush
}

type sentPush struct {
	tokens []string
	msg    firebase_client.PushMessage
}

func (f *fakeSender) Send(_ context.Context, tokens []string, msg firebase_client.PushMessage) (firebase_client.PushResult, error) {
	if f.err != nil {
		return firebase_client.PushResult{}, f.err
	}
	f.sent = append(f.sent, sentPush{tokens: tokens, msg: msg})
	var r firebase_client.PushResult
	for _, t := range tokens {
		switch {
		case strings.HasPrefix(t, "bad"):
			r.InvalidTokens = append(r.InvalidTokens, t)
		case strings.HasPrefix(t, "down"):
			r.TransientFailures++
		default:
			r.SuccessCount++
		}
	}
	return r, nil
}

func intPtr(i int) *int           { return &i }
func strPtr(s string) *string     { return &s }
func pushedIDs(f *fakeRepo) []int { return sortedKeys(f.pushed) }

func sortedKeys(m map[int]bool) []int {
	ids := []int{}
	for id, ok := range m {
		if ok {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	return ids
}

func TestRegisterDevice(t *testing.T) {
	cases := []struct {
		name, user, token, platform string
		want                        error
	}{
		{"android", userA, "fid:token", "android", nil},
		{"platform is normalized", userA, "fid:token2", " IOS ", nil},
		{"web", userA, "fid:token3", "web", nil},
		{"invalid user", "nope", "fid:token", "android", ErrInvalidUserID},
		{"empty token", userA, "   ", "android", ErrInvalidToken},
		{"too long token", userA, strings.Repeat("x", push_model.MaxTokenLength+1), "android", ErrInvalidToken},
		{"unknown platform", userA, "fid:token", "windows", ErrInvalidPlatform},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo := newFakeRepo()
			err := NewPushService(repo, &fakeSender{}).RegisterDevice(context.Background(), c.user, c.token, c.platform)
			if !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
			if c.want == nil && repo.platforms[strings.TrimSpace(c.token)] != strings.ToLower(strings.TrimSpace(c.platform)) {
				t.Errorf("stored platform = %q", repo.platforms[c.token])
			}
		})
	}
}

func TestUnregisterDeviceIsIdempotent(t *testing.T) {
	repo := newFakeRepo()
	repo.devices[userA] = []string{"fid:token"}
	svc := NewPushService(repo, &fakeSender{})
	for i := 0; i < 2; i++ {
		if err := svc.UnregisterDevice(context.Background(), userA, "fid:token"); err != nil {
			t.Fatalf("call %d: err = %v", i, err)
		}
	}
	if len(repo.devices[userA]) != 0 {
		t.Errorf("devices = %v, want none", repo.devices[userA])
	}
	if err := svc.UnregisterDevice(context.Background(), userA, ""); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("empty token: err = %v, want ErrInvalidToken", err)
	}
}

func TestDispatchPendingWithoutSender(t *testing.T) {
	if _, err := NewPushService(newFakeRepo(), nil).DispatchPending(context.Background()); !errors.Is(err, ErrPushDisabled) {
		t.Fatalf("err = %v, want ErrPushDisabled", err)
	}
}

func TestDispatchPendingSendsAndResolves(t *testing.T) {
	repo := newFakeRepo()
	repo.devices[userA] = []string{"fid:a1", "fid:a2"}
	repo.pending = []push_model.PendingPush{
		{ID: 1, UserID: userA, Type: "Social", Text: "anna s'ha apuntat a Concert", EventID: intPtr(7), FriendID: strPtr(userB), PushEnabled: true},
		{ID: 2, UserID: userB, Type: "Recordatori", Text: "sense dispositius", PushEnabled: true},
		{ID: 3, UserID: userA, Type: "Cancelacio", Text: "desactivat", PushEnabled: false},
	}
	sender := &fakeSender{}

	delivered, err := NewPushService(repo, sender).DispatchPending(context.Background())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if delivered != 1 {
		t.Errorf("delivered = %d, want 1", delivered)
	}
	if got := pushedIDs(repo); !reflect.DeepEqual(got, []int{1, 2, 3}) {
		t.Errorf("pushed = %v, want [1 2 3] (no devices and disabled are resolved without sending)", got)
	}
	if repo.expiredAge != MaxPushAge {
		t.Errorf("expired with max age %s, want %s", repo.expiredAge, MaxPushAge)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("sent %d pushes, want 1", len(sender.sent))
	}
	got := sender.sent[0]
	if !reflect.DeepEqual(got.tokens, []string{"fid:a1", "fid:a2"}) {
		t.Errorf("tokens = %v, want both devices of the user", got.tokens)
	}
	wantMsg := firebase_client.PushMessage{
		Title: "Social",
		Body:  "anna s'ha apuntat a Concert",
		Data:  map[string]string{"notification-id": "1", "type": "Social", "event-id": "7", "friend-id": userB},
	}
	if !reflect.DeepEqual(got.msg, wantMsg) {
		t.Errorf("message = %+v, want %+v", got.msg, wantMsg)
	}
}

func TestDispatchPendingRemovesInvalidTokens(t *testing.T) {
	repo := newFakeRepo()
	repo.devices[userA] = []string{"bad:gone"}
	repo.devices[userB] = []string{"bad:old", "fid:new"}
	repo.pending = []push_model.PendingPush{
		{ID: 1, UserID: userA, Type: "Social", Text: "x", PushEnabled: true},
		{ID: 2, UserID: userB, Type: "Social", Text: "y", PushEnabled: true},
		{ID: 3, UserID: userB, Type: "Social", Text: "z", PushEnabled: true},
	}
	sender := &fakeSender{}

	if _, err := NewPushService(repo, sender).DispatchPending(context.Background()); err != nil {
		t.Fatalf("err = %v", err)
	}
	if got := pushedIDs(repo); !reflect.DeepEqual(got, []int{1, 2, 3}) {
		t.Errorf("pushed = %v, want all: invalid devices are not retried", got)
	}
	sort.Strings(repo.deleted)
	if !reflect.DeepEqual(repo.deleted, []string{"bad:gone", "bad:old"}) {
		t.Errorf("deleted = %v, want the invalid tokens", repo.deleted)
	}
	// The third notification is not sent again to the token found invalid by the second one.
	if last := sender.sent[len(sender.sent)-1]; !reflect.DeepEqual(last.tokens, []string{"fid:new"}) {
		t.Errorf("last push tokens = %v, want [fid:new]", last.tokens)
	}
}

func TestDispatchPendingRetriesTransientFailures(t *testing.T) {
	repo := newFakeRepo()
	repo.devices[userA] = []string{"down:a"}
	repo.devices[userB] = []string{"down:b", "fid:b"}
	repo.pending = []push_model.PendingPush{
		{ID: 1, UserID: userA, Type: "Social", Text: "x", PushEnabled: true},
		{ID: 2, UserID: userB, Type: "Social", Text: "y", PushEnabled: true},
	}

	if _, err := NewPushService(repo, &fakeSender{}).DispatchPending(context.Background()); err != nil {
		t.Fatalf("err = %v", err)
	}
	if got := pushedIDs(repo); !reflect.DeepEqual(got, []int{2}) {
		t.Errorf("pushed = %v, want [2]: 1 only failed transiently and is retried", got)
	}
}

func TestDispatchPendingKeepsPendingWhenFCMFails(t *testing.T) {
	repo := newFakeRepo()
	repo.devices[userA] = []string{"fid:a"}
	repo.pending = []push_model.PendingPush{{ID: 1, UserID: userA, Type: "Social", Text: "x", PushEnabled: true}}

	delivered, err := NewPushService(repo, &fakeSender{err: errors.New("bad credentials")}).DispatchPending(context.Background())
	if err != nil {
		t.Fatalf("err = %v, want nil (the notification waits for the next dispatch)", err)
	}
	if delivered != 0 || len(pushedIDs(repo)) != 0 {
		t.Errorf("delivered = %d, pushed = %v, want nothing", delivered, pushedIDs(repo))
	}
}

func TestDispatchPendingFailsWhenDevicesCannotBeRead(t *testing.T) {
	repo := newFakeRepo()
	repo.tokensErr = errors.New("db down")
	repo.pending = []push_model.PendingPush{{ID: 1, UserID: userA, Type: "Social", Text: "x", PushEnabled: true}}

	if _, err := NewPushService(repo, &fakeSender{}).DispatchPending(context.Background()); err == nil {
		t.Fatal("err = nil, want the repository error")
	}
	if len(pushedIDs(repo)) != 0 {
		t.Errorf("pushed = %v, want nothing", pushedIDs(repo))
	}
}

func TestDispatchPendingProcessesEveryBatch(t *testing.T) {
	repo := newFakeRepo()
	repo.devices[userA] = []string{"fid:a"}
	for id := 1; id <= 5; id++ {
		repo.pending = append(repo.pending, push_model.PendingPush{ID: id, UserID: userA, Type: "Social", Text: "x", PushEnabled: true})
	}
	svc := NewPushService(repo, &fakeSender{})
	svc.batchSize = 2

	delivered, err := svc.DispatchPending(context.Background())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if delivered != 5 || len(pushedIDs(repo)) != 5 {
		t.Errorf("delivered = %d, pushed = %v, want the 5 notifications in 3 batches", delivered, pushedIDs(repo))
	}
}

func TestBuildMessageTitlesAndBody(t *testing.T) {
	cases := map[string]string{
		"Social":      "Social",
		"Recordatori": "Recordatori",
		"Recomanacio": "Recomanació",
		"Cancelacio":  "Esdeveniment cancel·lat",
		"Nivell":      DefaultTitle,
	}
	for typ, want := range cases {
		if got := buildMessage(push_model.PendingPush{ID: 1, Type: typ}).Title; got != want {
			t.Errorf("title of %s = %q, want %q", typ, got, want)
		}
	}

	msg := buildMessage(push_model.PendingPush{ID: 9, Type: "Social", Text: strings.Repeat("à", MaxBodyLength+10)})
	if n := len([]rune(msg.Body)); n != MaxBodyLength || !strings.HasSuffix(msg.Body, "…") {
		t.Errorf("body has %d characters, want %d ending with an ellipsis", n, MaxBodyLength)
	}
	if _, ok := msg.Data["event-id"]; ok {
		t.Error("data has event-id without an event")
	}
}
