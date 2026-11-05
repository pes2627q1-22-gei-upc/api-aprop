package postgres

import (
	"context"
	"errors"
	"fmt"

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
}

var _ user_repository.Repository = (*UserRepository)(nil)

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{queries: sqlc.New(pool)}
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (user_model.User, bool, error) {
	row, err := r.queries.GetUserByUsername(ctx, username)
	if errors.Is(err, pgx.ErrNoRows) {
		return user_model.User{}, false, user_repository.ErrUserNotFound
	}
	if err != nil {
		return user_model.User{}, false, fmt.Errorf("get user by username %q: %w", username, err)
	}
	u, showPlans := mapGetUserRowToUser(sqlc.GetUserByUUIDRow(row))
	return u, showPlans, nil
}

func (r *UserRepository) GetByFirebaseUID(ctx context.Context, firebaseUID string) (user_model.User, bool, error) {
	row, err := r.queries.GetUserByFirebaseUIDWithSettings(ctx, firebaseUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return user_model.User{}, false, user_repository.ErrUserNotFound
	}
	if err != nil {
		return user_model.User{}, false, fmt.Errorf("get user by firebase uid %q: %w", firebaseUID, err)
	}
	u, showPlans := mapGetUserRowToUser(sqlc.GetUserByUUIDRow(row))
	return u, showPlans, nil
}

func (r *UserRepository) GetByID(ctx context.Context, uid pgtype.UUID) (user_model.User, bool, error) {
	row, err := r.queries.GetUserByUUID(ctx, uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return user_model.User{}, false, user_repository.ErrUserNotFound
	}
	if err != nil {
		return user_model.User{}, false, fmt.Errorf("get user by uuid %q: %w", uid, err)
	}
	u, showPlans := mapGetUserRowToUser(row)
	return u, showPlans, nil
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

func (r *UserRepository) GetReliability(ctx context.Context, uid pgtype.UUID) (*float32, error) {
	val, err := r.queries.GetUserReliability(ctx, uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get reliability for %s: %w", uid, err)
	}
	rel := val
	return &rel, nil
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
		return user_model.User{}, fmt.Errorf("update profile of user %s: %w", uuidToString(id), err)
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

func (r *UserRepository) UpdateSettings(ctx context.Context, settingsID int, showPlans bool) error {
	_, err := r.queries.UpdateUserSettings(ctx, sqlc.UpdateUserSettingsParams{
		ID:        int32(settingsID),
		ShowPlans: showPlans,
	})
	if err != nil {
		return fmt.Errorf("update settings %d: %w", settingsID, err)
	}
	return nil
}

func (r *UserRepository) GetUserFriends(ctx context.Context, userid pgtype.UUID) ([]string, error){
	rows, err := r.queries.GetUserFriends(ctx, userid)
	if errors.Is(err, pgx.ErrNoRows){
		return []string{}, nil
	} else if err!=nil{
		return []string{}, err
	}
	friends := []string{}
	for _, val := range(rows){
		friends = append(friends, uuidToString(val))
	}

	return friends, nil
}

func (r *UserRepository) GetFriendRequests(ctx context.Context, userid pgtype.UUID) ([]user_model.FriendRequest, error){
	rows, err := r.queries.GetFriendRequests(ctx, userid)
	if errors.Is(err, pgx.ErrNoRows){
		return []user_model.FriendRequest{}, nil
	} else if err!=nil{
		return []user_model.FriendRequest{}, err
	}
	friendreqs := []user_model.FriendRequest{}
	for _, val := range(rows){
		friendreqs = append(friendreqs, toFriendRequest(val))
	}

	return friendreqs, nil
}

func mapGetUserRowToUser(row sqlc.GetUserByUUIDRow) (user_model.User, bool) {
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
	return u, row.ShowPlans
}

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	src := u.Bytes
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		src[0:4], src[4:6], src[6:8], src[8:10], src[10:16])
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

func toFriendRequest(row sqlc.Friendrequest) user_model.FriendRequest{
	fr := user_model.FriendRequest{
		Sender: row.Sender,
		Receiver: row.Receiver,
		DateSent: row.Datesent.Time,
		State: string(row.State),
		DateResolved: row.Dateresolved.Time,
	}
	return fr
}

func (r *UserRepository) PendingRequestExists(ctx context.Context, userid pgtype.UUID, target pgtype.UUID) (bool, error){
	exists, err := r.queries.PendingRequestExists(ctx, sqlc.PendingRequestExistsParams{Sender:userid, Receiver:target})
	if err!=nil{
		return false, err
	}
	return exists, nil
}

func (r *UserRepository) IsBlocked(ctx context.Context, userid pgtype.UUID, target pgtype.UUID) (bool, error){
	blocked, err := r.queries.IsBlocked(ctx, sqlc.IsBlockedParams{Blockeduser: userid, Blockinguser: target})
	if err!=nil{
		return false, err
	}
	return blocked, nil
}

func (r *UserRepository) SendFriendRequest(ctx context.Context, userid pgtype.UUID, target pgtype.UUID) (user_model.FriendRequest, error){
	row, err := r.queries.CreateFriendRequest(ctx, sqlc.CreateFriendRequestParams{Sender:userid, Receiver:target})
	if err!=nil{
		return user_model.FriendRequest{}, err
	}
	return toFriendRequest(row), nil
}