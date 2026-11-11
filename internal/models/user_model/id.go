package user_model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

// ErrInvalidID is returned when a user ID is not a canonical UUID.
var ErrInvalidID = errors.New("invalid user id")

// NormalizeID accepts only the canonical UUID form (8-4-4-4-12 hex digits) and lowercases it.
func NormalizeID(id string) (string, error) {
	if len(id) != 36 {
		return "", fmt.Errorf("%w: %q", ErrInvalidID, id)
	}
	for i, c := range id {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if c != '-' {
				return "", fmt.Errorf("%w: %q", ErrInvalidID, id)
			}
		case !isHex(c):
			return "", fmt.Errorf("%w: %q", ErrInvalidID, id)
		}
	}
	return strings.ToLower(id), nil
}

// ParseID converts a canonical UUID into the type used by the database queries.
func ParseID(id string) (pgtype.UUID, error) {
	normalized, err := NormalizeID(id)
	if err != nil {
		return pgtype.UUID{}, err
	}
	var uid pgtype.UUID
	if err := uid.Scan(normalized); err != nil {
		return pgtype.UUID{}, fmt.Errorf("%w: %q: %v", ErrInvalidID, id, err)
	}
	return uid, nil
}

func isHex(c rune) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}
