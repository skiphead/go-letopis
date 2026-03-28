package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/skiphead/go-letopis/internal/domain/entity"
	"github.com/skiphead/go-letopis/internal/domain/repository"
)

// MeetingUseCase defines the interface for meeting-related operations.
type MeetingUseCase interface {
	List(ctx context.Context, userID int64) ([]entity.Meeting, error)
	Get(ctx context.Context, meetingID, telegramID int64) (*entity.Meeting, error)
	SearchByKeywords(ctx context.Context, req entity.SearchRequest) ([]entity.TranscriptionRecord, error)
}

// meetingUseCase implements MeetingUseCase.
type meetingUseCase struct {
	meetingRepo repository.MeetingRepository
	logger      *slog.Logger
}

// NewMeetingUseCase creates a new MeetingUseCase instance.
func NewMeetingUseCase(meetingRepo repository.MeetingRepository, logger *slog.Logger) MeetingUseCase {
	return &meetingUseCase{
		meetingRepo: meetingRepo,
		logger:      logger,
	}
}

// List returns all meetings for a given user.
func (uc *meetingUseCase) List(ctx context.Context, userID int64) ([]entity.Meeting, error) {
	if err := uc.validateUserID(userID); err != nil {
		return nil, err
	}

	list, err := uc.meetingRepo.List(ctx, userID)
	if err != nil {
		uc.logger.Error("Failed to list meetings",
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to list meetings: %w", err)
	}

	uc.logger.Debug("Meetings listed successfully",
		slog.Int64("user_id", userID),
		slog.Int("count", len(list)),
	)

	return list, nil
}

// Get returns a specific meeting by ID and Telegram ID.
func (uc *meetingUseCase) Get(ctx context.Context, meetingID, telegramID int64) (*entity.Meeting, error) {
	if meetingID == 0 {
		return nil, fmt.Errorf("meeting_id is required")
	}
	if telegramID == 0 {
		return nil, fmt.Errorf("telegram_id is required")
	}

	meeting, err := uc.meetingRepo.Get(ctx, meetingID, telegramID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			uc.logger.Debug("Meeting not found",
				slog.Int64("meeting_id", meetingID),
				slog.Int64("telegram_id", telegramID),
			)
			return nil, nil // Return nil for meeting when not found
		}
		uc.logger.Error("Failed to get meeting",
			slog.Int64("meeting_id", meetingID),
			slog.Int64("telegram_id", telegramID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get meeting: %w", err)
	}

	uc.logger.Debug("Meeting retrieved successfully",
		slog.Int64("meeting_id", meetingID),
		slog.Int64("telegram_id", telegramID),
	)

	return meeting, nil
}

// SearchByKeywords performs a keyword search on meetings.
func (uc *meetingUseCase) SearchByKeywords(ctx context.Context, req entity.SearchRequest) ([]entity.TranscriptionRecord, error) {
	if err := uc.validateSearchRequest(req); err != nil {
		return nil, err
	}

	var records []entity.TranscriptionRecord
	for record, err := range uc.meetingRepo.SearchByKeywordsIter(ctx, req) {
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	uc.logger.Debug("Search completed successfully",
		slog.Int64("user_id", req.UserID),
		slog.Any("keywords", req.Keywords),
		slog.Int("results_count", len(records)),
	)

	return records, nil
}

// validateUserID validates the user ID parameter.
func (uc *meetingUseCase) validateUserID(userID int64) error {
	if userID == 0 {
		return fmt.Errorf("user_id is required")
	}
	return nil
}

// validateSearchRequest validates the search request parameters.
func (uc *meetingUseCase) validateSearchRequest(req entity.SearchRequest) error {
	if req.UserID == 0 {
		return fmt.Errorf("user_id is required")
	}
	if len(req.Keywords) == 0 {
		return fmt.Errorf("at least one keyword is required for search")
	}

	// Validate keywords are not empty strings
	for i, keyword := range req.Keywords {
		if strings.TrimSpace(keyword) == "" {
			return fmt.Errorf("keyword at index %d cannot be empty", i)
		}
	}

	return nil
}
