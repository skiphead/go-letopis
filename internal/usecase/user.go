package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/skiphead/go-letopis/internal/domain/entity"
	"github.com/skiphead/go-letopis/internal/domain/repository"
)

// ErrUserAlreadyExists is returned when trying to create a user that already exists.
var ErrUserAlreadyExists = errors.New("user already exists")

// UserUseCase defines the interface for user-related operations.
type UserUseCase interface {
	Start(ctx context.Context, user *entity.User) (*entity.User, error)
	Validate(ctx context.Context, userID int64) bool
}

// userUseCase implements UserUseCase.
type userUseCase struct {
	userRepo repository.UserRepository
	logger   *slog.Logger
}

// NewUserUseCase creates a new UserUseCase instance.
func NewUserUseCase(userRepo repository.UserRepository, logger *slog.Logger) UserUseCase {
	return &userUseCase{
		userRepo: userRepo,
		logger:   logger,
	}
}

// Start initializes a new user session.
// If the user already exists, it returns ErrUserAlreadyExists.
func (uc *userUseCase) Start(ctx context.Context, user *entity.User) (*entity.User, error) {
	// Validate input
	if user == nil {
		return nil, fmt.Errorf("user cannot be nil")
	}
	if user.TelegramID == 0 {
		return nil, fmt.Errorf("telegram_id is required")
	}

	// Check if user already exists
	result, err := uc.userRepo.Get(ctx, user.TelegramID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if result != nil {
		uc.logger.Warn("User already exists",
			slog.Int64("telegram_id", user.TelegramID),
			slog.String("username", user.UserName),
		)
		return nil, ErrUserAlreadyExists
	}

	// Create new user
	createResult, err := uc.userRepo.Create(ctx, &entity.User{
		TelegramID: user.TelegramID,
		UserName:   user.UserName,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	uc.logger.Info("User created successfully",
		slog.Int64("user_id", createResult.ID),
		slog.Int64("telegram_id", createResult.TelegramID),
		slog.String("username", createResult.UserName),
	)

	return createResult, nil
}

// Validate checks if a user exists and is valid.
func (uc *userUseCase) Validate(ctx context.Context, userID int64) bool {
	if userID == 0 {
		uc.logger.Warn("Validate called with zero user ID")
		return false
	}

	result, err := uc.userRepo.Get(ctx, userID)
	if err != nil {
		uc.logger.Error("Failed to validate user",
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		return false
	}

	// Simplified check: user exists and telegram_id matches
	if result != nil {
		uc.logger.Debug("User validated successfully",
			slog.Int64("user_id", userID),
			slog.Int64("telegram_id", result.TelegramID),
		)
		return true
	}

	uc.logger.Debug("User not found",
		slog.Int64("user_id", userID),
	)
	return false
}
