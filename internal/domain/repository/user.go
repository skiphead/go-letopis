package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skiphead/go-letopis/internal/domain/entity"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	Create(ctx context.Context, user *entity.User) (*entity.User, error)
	Get(ctx context.Context, telegramID int64) (*entity.User, error)
}

// userRepository implements UserRepository using generic repository
type userRepository struct {
	generic     *GenericRepository[entity.User]
	pool        *pgxpool.Pool
	logger      *slog.Logger
	nullHandler *NullHandler
}

// NewUserRepository creates a new UserRepository instance
func NewUserRepository(db *pgxpool.Pool, logger *slog.Logger) UserRepository {
	generic, err := NewGenericRepository[entity.User](db, logger, "users")
	if err != nil {
		panic(fmt.Sprintf("failed to create user repository: %v", err))
	}
	return &userRepository{
		generic:     generic,
		pool:        db,
		logger:      logger,
		nullHandler: &NullHandler{},
	}
}

// scanUser scans a single user row from the database
func (r *userRepository) scanUser(row pgx.Row) (*entity.User, error) {
	var user entity.User
	var userName, firstName, lastName sql.NullString

	err := row.Scan(
		&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.TelegramID,
		&userName, &firstName, &lastName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	// Unified NULL handling using NullHandler
	user.UserName = r.nullHandler.FromNullString(userName)
	user.FirstName = r.nullHandler.FromNullString(firstName)
	user.LastName = r.nullHandler.FromNullString(lastName)

	return &user, nil
}

// Create inserts a new user record into the database
func (r *userRepository) Create(ctx context.Context, user *entity.User) (*entity.User, error) {
	if user == nil {
		return nil, errors.New("user is nil")
	}

	if user.TelegramID == 0 {
		return nil, errors.New("telegram_id is required and cannot be 0")
	}

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// NULL handling is done automatically by entityToFieldMap which uses DefaultNullableStrategy
	returningColumns := []string{"id", "created_at", "updated_at", "telegram_id", "username", "first_name", "last_name"}

	createdUser, err := r.generic.CreateWithReturning(ctx, user, returningColumns)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return createdUser, nil
}

// Get retrieves a user by Telegram ID
func (r *userRepository) Get(ctx context.Context, telegramID int64) (*entity.User, error) {
	if telegramID == 0 {
		return nil, errors.New("telegram_id is required and cannot be 0")
	}

	query := `SELECT id, created_at, updated_at, telegram_id, username, first_name, last_name 
		FROM users 
		WHERE telegram_id = $1`

	row := r.pool.QueryRow(ctx, query, telegramID)

	user, err := r.scanUser(row)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}
