package push_service

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"api-aprop/internal/clients/firebase_client"
	"api-aprop/internal/models/push_model"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/push_repository"
)

// TODO(DP-044): values proposed by the backend.
const (
	// MaxPushAge is how long a notification waits to be pushed; an older one only stays in the inbox.
	MaxPushAge = 24 * time.Hour
	// BatchSize is the number of notifications claimed from the outbox in each transaction.
	BatchSize = 100
	// MaxBodyLength bounds the text of a push (in characters) so the FCM payload is always valid.
	MaxBodyLength = 1000
	// DefaultTitle is the title of a push whose notification type has no title of its own.
	DefaultTitle = "AProp!"
)

// titles is the title of the push of each notification type (NotificationTypes.Name).
var titles = map[string]string{
	"Social":      "Social",
	"Recordatori": "Recordatori",
	"Recomanacio": "Recomanació",
	"Cancelacio":  "Esdeveniment cancel·lat",
}

type PushService struct {
	repo      push_repository.Repository
	sender    firebase_client.PushSender // nil without Firebase credentials
	batchSize int
}

var _ Service = (*PushService)(nil)

func NewPushService(repo push_repository.Repository, sender firebase_client.PushSender) *PushService {
	return &PushService{repo: repo, sender: sender, batchSize: BatchSize}
}

func (s *PushService) RegisterDevice(ctx context.Context, userID, token, platform string) error {
	userID, err := user_model.NormalizeID(userID)
	if err != nil {
		return err
	}
	token, err = validToken(token)
	if err != nil {
		return err
	}
	platform = strings.ToLower(strings.TrimSpace(platform))
	if !push_model.ValidPlatform(platform) {
		return ErrInvalidPlatform
	}
	return s.repo.UpsertDevice(ctx, userID, push_model.Device{Token: token, Platform: platform})
}

func (s *PushService) UnregisterDevice(ctx context.Context, userID, token string) error {
	userID, err := user_model.NormalizeID(userID)
	if err != nil {
		return err
	}
	token, err = validToken(token)
	if err != nil {
		return err
	}
	_, err = s.repo.DeleteDevice(ctx, userID, token)
	return err
}

// DispatchPending expires the old pending notifications and pushes the rest, a batch per transaction, until the
// outbox is empty or a batch leaves notifications to retry (they wait for the next dispatch).
func (s *PushService) DispatchPending(ctx context.Context) (int, error) {
	if s.sender == nil {
		return 0, ErrPushDisabled
	}
	expired, err := s.repo.ExpirePending(ctx, MaxPushAge)
	if err != nil {
		return 0, err
	}
	if expired > 0 {
		slog.Info("expired pending push notifications", "count", expired, "max-age", MaxPushAge)
	}

	delivered := 0
	for {
		allResolved := false
		claimed, err := s.repo.ProcessPending(ctx, s.batchSize, func(ctx context.Context, batch []push_model.PendingPush) ([]int, error) {
			resolved, n, err := s.sendBatch(ctx, batch)
			delivered += n
			allResolved = len(resolved) == len(batch)
			return resolved, err
		})
		if err != nil {
			return delivered, err
		}
		if claimed < s.batchSize || !allResolved {
			return delivered, nil
		}
	}
}

// sendBatch pushes each notification of the batch to the devices of its user. It returns the resolved notifications
// and how many reached a device. Only reading the devices fails the batch: it happens before sending anything, so the
// rollback does not lose sent pushes. A notification is resolved when the push is disabled, the user has no devices,
// a device received it or every failure was a device that no longer exists; otherwise it is retried.
func (s *PushService) sendBatch(ctx context.Context, batch []push_model.PendingPush) ([]int, int, error) {
	var userIDs []string
	seen := map[string]bool{}
	for _, p := range batch {
		if p.PushEnabled && !seen[p.UserID] {
			seen[p.UserID] = true
			userIDs = append(userIDs, p.UserID)
		}
	}
	tokens, err := s.repo.DeviceTokens(ctx, userIDs)
	if err != nil {
		return nil, 0, err
	}

	resolved := make([]int, 0, len(batch))
	delivered := 0
	for _, p := range batch {
		userTokens := tokens[p.UserID]
		if !p.PushEnabled || len(userTokens) == 0 {
			resolved = append(resolved, p.ID)
			continue
		}
		result, err := s.sender.Send(ctx, userTokens, buildMessage(p))
		if err != nil {
			slog.Warn("could not push notification, it will be retried", "notification", p.ID, "error", err)
			continue
		}
		if len(result.InvalidTokens) > 0 {
			tokens[p.UserID] = without(userTokens, result.InvalidTokens)
			if err := s.repo.DeleteDeviceTokens(ctx, result.InvalidTokens); err != nil {
				// The push was already sent, so the batch goes on; FCM will reject the tokens again next time.
				slog.Warn("could not delete invalid device tokens", "count", len(result.InvalidTokens), "error", err)
			}
		}
		if result.SuccessCount > 0 {
			delivered++
		}
		if result.SuccessCount > 0 || result.TransientFailures == 0 {
			resolved = append(resolved, p.ID)
		}
	}
	return resolved, delivered, nil
}

// buildMessage builds the push of a notification: the title depends on its type and the body is its text.
func buildMessage(p push_model.PendingPush) firebase_client.PushMessage {
	title, ok := titles[p.Type]
	if !ok {
		title = DefaultTitle
	}
	data := map[string]string{
		"notification-id": strconv.Itoa(p.ID),
		"type":            p.Type,
	}
	if p.EventID != nil {
		data["event-id"] = strconv.Itoa(*p.EventID)
	}
	if p.FriendID != nil {
		data["friend-id"] = *p.FriendID
	}
	return firebase_client.PushMessage{Title: title, Body: truncate(p.Text, MaxBodyLength), Data: data}
}

func validToken(token string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" || utf8.RuneCountInString(token) > push_model.MaxTokenLength {
		return "", ErrInvalidToken
	}
	return token, nil
}

// truncate cuts text to max characters, ending it with an ellipsis when it is cut.
func truncate(text string, max int) string {
	if utf8.RuneCountInString(text) <= max {
		return text
	}
	runes := []rune(text)
	return string(runes[:max-1]) + "…"
}

// without returns tokens minus the removed ones.
func without(tokens, removed []string) []string {
	drop := map[string]bool{}
	for _, t := range removed {
		drop[t] = true
	}
	kept := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if !drop[t] {
			kept = append(kept, t)
		}
	}
	return kept
}
