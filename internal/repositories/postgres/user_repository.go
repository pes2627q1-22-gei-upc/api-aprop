package postgres

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/postgres/sqlc"
	"api-aprop/internal/repositories/user_repository"
)

type UserRepository struct {
	queries *sqlc.Queries
	pool    *pgxpool.Pool
}

var _ user_repository.Repository = (*UserRepository)(nil)

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{queries: sqlc.New(pool), pool: pool}
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (user_model.User, error) {
	row, err := r.queries.GetUserByUsername(ctx, username)
	if errors.Is(err, pgx.ErrNoRows) {
		return user_model.User{}, user_repository.ErrUserNotFound
	}
	if err != nil {
		return user_model.User{}, fmt.Errorf("get user by username %q: %w", username, err)
	}
	u := mapGetUserRowToUser(sqlc.GetUserByUUIDRow(row))
	return u, nil
}

func (r *UserRepository) GetByFirebaseUID(ctx context.Context, firebaseUID string) (user_model.User, error) {
	row, err := r.queries.GetUserByFirebaseUIDWithSettings(ctx, firebaseUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return user_model.User{}, user_repository.ErrUserNotFound
	}
	if err != nil {
		return user_model.User{}, fmt.Errorf("get user by firebase uid %q: %w", firebaseUID, err)
	}
	u := mapGetUserRowToUser(sqlc.GetUserByUUIDRow(row))
	return u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, uid pgtype.UUID) (user_model.User, error) {
	row, err := r.queries.GetUserByUUID(ctx, uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return user_model.User{}, user_repository.ErrUserNotFound
	}
	if err != nil {
		return user_model.User{}, fmt.Errorf("get user by uuid %q: %w", uid, err)
	}
	u := mapGetUserRowToUser(row)
	return u, nil
}

func (r *UserRepository) CountFriends(ctx context.Context, uid pgtype.UUID) (int, error) {
	count, err := r.queries.CountFriends(ctx, uid)
	if err != nil {
		return 0, fmt.Errorf("count friends for %s: %w", uid, err)
	}
	return int(count), nil
}

func (r *UserRepository) AreFriends(ctx context.Context, u1 pgtype.UUID, u2 pgtype.UUID) (bool, error) {
	if !u1.Valid || !u2.Valid {
		return false, nil
	}
	if u1 == u2 {
		return false, user_repository.ErrCannotFriendSelf
	}
	areFriends, err := r.queries.AreFriends(ctx, sqlc.AreFriendsParams{
		User1: u1,
		User2: u2,
	})
	if err != nil {
		return false, fmt.Errorf("check are friends: %w", err)
	}
	return areFriends, nil
}

func (r *UserRepository) Unfriend(ctx context.Context, userid pgtype.UUID, targetuuid pgtype.UUID) error {
	err := r.queries.RemoveFriendship(ctx, sqlc.RemoveFriendshipParams{User1: userid, User2: targetuuid})
	return err
}

func (r *UserRepository) CountPlans(ctx context.Context, uid pgtype.UUID) (int, error) {
	count, err := r.queries.CountUserPlans(ctx, uid)
	if err != nil {
		return 0, fmt.Errorf("count plans for %s: %w", uid, err)
	}
	return int(count), nil
}

func (r *UserRepository) CountCompletedPlans(ctx context.Context, uid pgtype.UUID) (int, error) {
	count, err := r.queries.CountCompletedUserPlans(ctx, uid)
	if err != nil {
		return 0, fmt.Errorf("count completed plans for %s: %w", uid, err)
	}
	return int(count), nil
}

func (r *UserRepository) ListReceivedRatings(ctx context.Context, uid pgtype.UUID) ([]user_model.ReceivedRating, error) {
	rows, err := r.queries.ListUserReceivedRatings(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("list received ratings for %s: %w", uid, err)
	}
	out := make([]user_model.ReceivedRating, 0, len(rows))
	for _, row := range rows {
		out = append(out, user_model.ReceivedRating{
			EventID:    int(row.EventID),
			EventTitle: row.EventTitle,
			Rating:     int(row.Rating),
			RatingDate: row.Ratingdate.Time,
		})
	}
	return out, nil
}

func (r *UserRepository) ListCreatedEvents(ctx context.Context, uid pgtype.UUID) ([]event_model.Event, error) {
	rows, err := r.queries.ListUserCreatedEvents(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("list created events for %s: %w", uid, err)
	}
	events := make([]event_model.Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, createdEventToModel(row))
	}
	return events, nil
}

func (r *UserRepository) UpdateProfile(ctx context.Context, id pgtype.UUID, name, username, biography, profilePicture *string) (user_model.User, error) {
	params := sqlc.UpdateUserProfileParams{
		ID: id,
	}
	if name != nil {
		params.Name = pgtype.Text{String: *name, Valid: true}
	}
	if username != nil {
		params.Username = pgtype.Text{String: *username, Valid: true}
	}
	if biography != nil {
		params.Biography = pgtype.Text{String: *biography, Valid: true}
	}
	if profilePicture != nil {
		params.ProfilePicture = pgtype.Text{String: *profilePicture, Valid: true}
	}

	row, err := r.queries.UpdateUserProfile(ctx, params)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		if pgErr.ConstraintName == usersUsernameKey {
			return user_model.User{}, user_repository.ErrUsernameTaken
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return user_model.User{}, user_repository.ErrUserNotFound
	}
	if err != nil {
		return user_model.User{}, fmt.Errorf("update profile of user %s: %w", id.String(), err)
	}

	u := user_model.User{
		ID:             row.ID,
		FirebaseUID:    row.Firebaseuid,
		Username:       row.Username,
		Email:          row.Email.String,
		ProfilePicture: row.Profilepicture.String,
		Biography:      row.Biography.String,
		DateRegistered: row.Dateregistered.Time,
		AccountState:   int(row.Accountstate),
		Experience:     int(row.Experience),
		Settings:       int(row.Settings),
	}
	if row.Name.Valid {
		u.Name = row.Name.String
	}
	if row.Level.Valid {
		lvl := int(row.Level.Int32)
		u.Level = &lvl
	}
	if row.Rating.Valid {
		r := row.Rating.Float32
		u.Rating = &r
	}
	return u, nil
}

func isInvalidInterval(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "22007" ||
			pgErr.Code == "22008" ||
			pgErr.Code == "22015"
	}
	return false
}

func (r *UserRepository) UpdateSettings(ctx context.Context, settings user_model.Settings) error {
	nottypes, err := r.queries.GetNotificationTypes(ctx)
	if err != nil {
		return err
	}
	for _, v := range settings.AllowedNotifTypes {
		if !slices.ContainsFunc(nottypes, func(i1 sqlc.Notificationtype) bool { return i1.ID == int32(v) }) {
			return user_repository.ErrInvalidNotificationType
		}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qtx := r.queries.WithTx(tx)

	// TODO(DP-043): with a language, a hand-written query updates everything in one statement (the generated Setting model does not have Language).
	if settings.Language != "" {
		err = qtx.UpdateSettings(ctx, sqlc.UpdateSettingsParams{
			ID:                 int32(settings.ID),
			ShowPlans:          settings.ShowPlans,
			NotificationBefore: settings.NotificationBefore,
			Language:           settings.Language,
		})
	} else {
		_, err = qtx.UpdateUserSettings(ctx, sqlc.UpdateUserSettingsParams{
			ID:                 int32(settings.ID),
			ShowPlans:          settings.ShowPlans,
			NotificationBefore: settings.NotificationBefore,
		})
	}
	if isInvalidInterval(err) {
		return user_repository.ErrInvalidInterval
	} else if err != nil {
		return fmt.Errorf("update settings %d: %w", settings.ID, err)
	}

	int32types := []int32{}
	for _, v := range settings.AllowedNotifTypes {
		int32types = append(int32types, int32(v))
	}

	err = qtx.DeleteAllowedNotificationTypes(ctx, int32(settings.ID))
	if err != nil {
		return fmt.Errorf("update settings %d: %w", settings.ID, err)
	}

	err = qtx.UpdateAllowedNotificationTypes(ctx, sqlc.UpdateAllowedNotificationTypesParams{SettingsID: int32(settings.ID), NotificationTypes: int32types})
	if err != nil {
		return fmt.Errorf("update settings %d: %w", settings.ID, err)
	}

	return tx.Commit(ctx)
}

func (r *UserRepository) GetUserFriends(ctx context.Context, userid pgtype.UUID) ([]string, error) {
	rows, err := r.queries.GetUserFriends(ctx, userid)
	if errors.Is(err, pgx.ErrNoRows) {
		return []string{}, nil
	} else if err != nil {
		return []string{}, err
	}
	friends := []string{}
	for _, val := range rows {
		friends = append(friends, val.String())
	}

	return friends, nil
}

func (r *UserRepository) GetFriendRequests(ctx context.Context, userid pgtype.UUID) ([]user_model.FriendRequest, error) {
	rows, err := r.queries.GetFriendRequests(ctx, userid)
	if errors.Is(err, pgx.ErrNoRows) {
		return []user_model.FriendRequest{}, nil
	} else if err != nil {
		return []user_model.FriendRequest{}, err
	}
	friendreqs := []user_model.FriendRequest{}
	for _, val := range rows {
		friendreqs = append(friendreqs, toFriendRequest(val))
	}

	return friendreqs, nil
}

func mapGetUserRowToUser(row sqlc.GetUserByUUIDRow) user_model.User {
	u := user_model.User{
		ID:             row.ID,
		FirebaseUID:    row.Firebaseuid,
		Username:       row.Username,
		Email:          row.Email.String,
		ProfilePicture: row.Profilepicture.String,
		Biography:      row.Biography.String,
		DateRegistered: row.Dateregistered.Time,
		AccountState:   int(row.Accountstate),
		Experience:     int(row.Experience),
		Settings:       int(row.Settings),
	}
	if row.Name.Valid {
		u.Name = row.Name.String
	}
	if row.Level.Valid {
		l := int(row.Level.Int32)
		u.Level = &l
	}
	if row.Rating.Valid {
		r := row.Rating.Float32
		u.Rating = &r
	}
	return u
}

func createdEventToModel(row sqlc.ListUserCreatedEventsRow) event_model.Event {
	return toModel(sqlc.Event{
		ID:           row.ID,
		Title:        row.Title,
		Subtitle:     row.Subtitle,
		Description:  row.Description,
		Schedule:     row.Schedule,
		Url:          row.Url,
		Tickets:      row.Tickets,
		Ticketsurl:   row.Ticketsurl,
		Free:         row.Free,
		Permanent:    row.Permanent,
		Modality:     toText(row.Modality.String),
		Relatedurls:  row.Relatedurls,
		Urlnames:     row.Urlnames,
		Images:       row.Images,
		Videos:       row.Videos,
		Email:        row.Email,
		Place:        row.Place,
		Latitude:     row.Latitude,
		Longitude:    row.Longitude,
		PostalCode:   row.PostalCode,
		Municipality: row.Municipality,
		Region:       row.Region,
		Address:      row.Address,
		Phone:        row.Phone,
		Activityurl:  row.Activityurl,
		Startdate:    row.Startdate,
		Enddate:      row.Enddate,
	}, 0)
}

func toFriendRequest(row sqlc.Friendrequest) user_model.FriendRequest {
	fr := user_model.FriendRequest{
		Sender:       row.Sender,
		Receiver:     row.Receiver,
		DateSent:     row.Datesent.Time,
		State:        string(row.State),
		DateResolved: row.Dateresolved.Time,
	}
	return fr
}

func (r *UserRepository) PendingRequestExists(ctx context.Context, userid pgtype.UUID, target pgtype.UUID) (bool, error) {
	exists, err := r.queries.PendingRequestExists(ctx, sqlc.PendingRequestExistsParams{Sender: userid, Receiver: target})
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (r *UserRepository) IsBlocked(ctx context.Context, userid pgtype.UUID, target pgtype.UUID) (bool, error) {
	blocked, err := r.queries.IsBlocked(ctx, sqlc.IsBlockedParams{Blockeduser: userid, Blockinguser: target})
	if err != nil {
		return false, err
	}
	return blocked, nil
}

func (r *UserRepository) SendFriendRequest(ctx context.Context, userid pgtype.UUID, target pgtype.UUID) (user_model.FriendRequest, error) {
	row, err := r.queries.CreateFriendRequest(ctx, sqlc.CreateFriendRequestParams{Sender: userid, Receiver: target})
	if err != nil {
		return user_model.FriendRequest{}, err
	}
	return toFriendRequest(row), nil
}

func (r *UserRepository) AcceptFriendRequest(ctx context.Context, userid pgtype.UUID, targetid pgtype.UUID, date time.Time) (user_model.FriendRequest, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return user_model.FriendRequest{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qtx := r.queries.WithTx(tx)

	friendreq, err := qtx.AcceptFriendRequest(ctx, sqlc.AcceptFriendRequestParams{Sender: targetid, Receiver: userid, Datesent: toTimestamp(date)})
	if err != nil {
		return user_model.FriendRequest{}, err
	}

	err = qtx.AddFriendship(ctx, sqlc.AddFriendshipParams{User1: userid, User2: targetid})
	if err != nil {
		return user_model.FriendRequest{}, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return user_model.FriendRequest{}, err
	}

	return toFriendRequest(friendreq), nil
}

func (r *UserRepository) DenyFriendRequest(ctx context.Context, userid pgtype.UUID, targetid pgtype.UUID, date time.Time) (user_model.FriendRequest, error) {
	row, err := r.queries.DenyFriendRequest(ctx, sqlc.DenyFriendRequestParams{Sender: targetid, Receiver: userid, Datesent: toTimestamp(date)})
	if err != nil {
		return user_model.FriendRequest{}, err
	}

	return toFriendRequest(row), nil
}

func (r *UserRepository) GetSettings(ctx context.Context, id int) (user_model.Settings, error) {
	rows, err := r.queries.GetSettings(ctx, int32(id))
	if err != nil {
		return user_model.Settings{}, err
	}

	allowed := []int{}
	if rows[0].Notificationtype != -1 {
		for _, val := range rows {
			allowed = append(allowed, int(val.Notificationtype))
		}
	}

	language, err := r.queries.GetSettingsLanguage(ctx, int32(id))
	if err != nil {
		return user_model.Settings{}, err
	}

	return user_model.Settings{ID: int(rows[0].ID), NotificationBefore: intervalToString(rows[0].Notificationbefore), ShowPlans: rows[0].Showplans, AllowedNotifTypes: allowed, Language: language}, nil
}

func toTimestamp(t time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{Time: t, Valid: true}
}

// DeleteUser erases the user and everything that references it (RGPD), in an order that respects the
// foreign keys. TODO(DP-031): the unofficial events of the user are deleted too, with the sign ups and
// ratings of other users, and the ratings the user received.
func (r *UserRepository) DeleteUser(ctx context.Context, id pgtype.UUID, beforeCommit func(context.Context) error) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.queries.WithTx(tx)

		eventIDs, err := q.ListCreatedUnofficialEventIDs(ctx, id)
		if err != nil {
			return fmt.Errorf("list events created by user %s: %w", id, err)
		}
		for _, eventID := range eventIDs {
			if _, err := deleteEventTx(ctx, q, eventID); err != nil {
				return fmt.Errorf("delete events created by user %s: %w", id, err)
			}
		}

		dependents := []struct {
			name   string
			delete func(context.Context, pgtype.UUID) error
		}{
			// Notifications reference friend requests, so they go first
			{"notifications", q.DeleteUserNotifications},
			{"friend requests", q.DeleteUserFriendRequests},
			{"friendships", q.DeleteUserFriendships},
			{"blocks", q.DeleteUserBlocks},
			{"sign-ups", q.DeleteUserSignUps},
			{"event ratings", q.DeleteUserEventRatings},
			{"user ratings", q.DeleteUserUserRatings},
			{"messages", q.DeleteUserMessages},
			{"preference test choices", q.DeleteUserPreferenceTestChoices},
			{"preference tests", q.DeleteUserPreferenceTests},
			{"preferences", q.DeleteUserPreferences},
			{"level", q.DeleteUserLevel},
			{"state", q.DeleteUserState},
			{"chatbot conversations", q.DeleteUserChatbotConversations},
		}
		for _, d := range dependents {
			if err := d.delete(ctx, id); err != nil {
				return fmt.Errorf("delete %s of user %s: %w", d.name, id, err)
			}
		}

		settingsID, err := q.DeleteUserRow(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("delete user %s: %w", id, user_repository.ErrUserNotFound)
		}
		if err != nil {
			return fmt.Errorf("delete user %s: %w", id, err)
		}
		if err := q.DeleteSettings(ctx, settingsID); err != nil {
			return fmt.Errorf("delete settings of user %s: %w", id, err)
		}

		if beforeCommit != nil {
			if err := beforeCommit(ctx); err != nil {
				return err
			}
		}
		return nil
	})
}

func intervalToString(interval pgtype.Interval) string {
	return fmt.Sprintf("%d mons %d days %d usecs",
		interval.Months,
		interval.Days,
		interval.Microseconds,
	)
}
