package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/auth_repository"
	"api-aprop/internal/repositories/postgres/sqlc"
)

const (
	uniqueViolation = "23505"

	// Names Postgres gives to the UNIQUE constraints of the Users table
	usersFirebaseUIDKey = "users_firebaseuid_key"
	usersUsernameKey    = "users_username_key"
)

// AuthRepository stores users in Postgres using the sqlc generated queries.
type AuthRepository struct {
	queries *sqlc.Queries
}

var _ auth_repository.Repository = (*AuthRepository)(nil)

func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{queries: sqlc.New(pool)}
}

func (r *AuthRepository) GetByFirebaseUID(ctx context.Context, firebaseUID string) (user_model.User, error) {
	row, err := r.queries.GetUserByFirebaseUID(ctx, firebaseUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return user_model.User{}, fmt.Errorf("get user %s: %w", firebaseUID, auth_repository.ErrUserNotFound)
	}
	if err != nil {
		return user_model.User{}, fmt.Errorf("get user %s: %w", firebaseUID, err)
	}
	return toUserModel(row), nil
}

func (r *AuthRepository) Create(ctx context.Context, firebaseUID, username, email string) (user_model.User, error) {
	row, err := r.queries.CreateUser(ctx, sqlc.CreateUserParams{
		FirebaseUid: firebaseUID,
		Username:    username,
		Email:       toText(email),
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		switch pgErr.ConstraintName {
		case usersUsernameKey:
			return user_model.User{}, fmt.Errorf("create user %s: %w", firebaseUID, auth_repository.ErrUsernameTaken)
		case usersFirebaseUIDKey:
			return user_model.User{}, fmt.Errorf("create user %s: %w", firebaseUID, auth_repository.ErrUserAlreadyExists)
		}
	}
	if err != nil {
		return user_model.User{}, fmt.Errorf("create user %s: %w", firebaseUID, err)
	}
	return toUserModel(row), nil
}

func (r *AuthRepository) UpdateEmail(ctx context.Context, id pgtype.UUID, email string) (user_model.User, error) {
	row, err := r.queries.UpdateUserEmail(ctx, sqlc.UpdateUserEmailParams{ID: id, Email: toText(email)})
	if errors.Is(err, pgx.ErrNoRows) {
		return user_model.User{}, fmt.Errorf("update email of user %s: %w", uuidToString(id), auth_repository.ErrUserNotFound)
	}
	if err != nil {
		return user_model.User{}, fmt.Errorf("update email of user %s: %w", uuidToString(id), err)
	}
	return toUserModel(row), nil
}

// toUserModel converts a Users row into the model. NULL optional columns become nil or empty strings.
func toUserModel(row sqlc.User) user_model.User {
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
	if row.Level.Valid {
		level := int(row.Level.Int32)
		u.Level = &level
	}
	if row.Rating.Valid {
		rating := row.Rating.Float32
		u.Rating = &rating
	}
	return u
}
