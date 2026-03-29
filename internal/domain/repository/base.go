package repository

import (
	"context"
	"errors"
	"regexp"
)

// ErrUserNotFound is returned when a user is not found in the database.
var ErrUserNotFound = errors.New("user not found")

// columnNameRegex validates column names to prevent SQL injection
var columnNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// tsQuerySanitizer removes dangerous characters from tsquery input
var tsQuerySanitizer = regexp.MustCompile(`[^\w\s&|!()\-]`)

type NullableStrategy int

// BaseRepository defines generic CRUD operations
type BaseRepository[T any] interface {
	Create(ctx context.Context, entity *T) error
	GetByID(ctx context.Context, id int64) (*T, error)
	List(ctx context.Context, limit, offset int) ([]T, error)
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id int64) error
}
