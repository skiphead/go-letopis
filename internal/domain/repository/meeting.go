package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skiphead/go-letopis/internal/domain/entity"
)

// MeetingRepository defines the interface for meeting data operations
type MeetingRepository interface {
	Create(ctx context.Context, meeting *entity.Meeting) error
	List(ctx context.Context, userID int64) ([]entity.Meeting, error)
	Get(ctx context.Context, id, telegramID int64) (*entity.Meeting, error)
	SearchByKeywordsIter(
		ctx context.Context,
		req entity.SearchRequest,
	) iter.Seq2[entity.TranscriptionRecord, error]
}

// meetingRepository implements MeetingRepository using generic repository
type meetingRepository struct {
	generic     *GenericRepository[entity.Meeting]
	pool        *pgxpool.Pool
	logger      *slog.Logger
	nullHandler *NullHandler
}

// NewMeetingRepository creates a new MeetingRepository instance
func NewMeetingRepository(db *pgxpool.Pool, logger *slog.Logger) MeetingRepository {
	generic, err := NewGenericRepository[entity.Meeting](db, logger, "meetings")
	if err != nil {
		panic(fmt.Sprintf("failed to create meeting repository: %v", err))
	}
	return &meetingRepository{
		generic:     generic,
		pool:        db,
		logger:      logger,
		nullHandler: &NullHandler{},
	}
}

// Create inserts a new meeting record into the database
func (r *meetingRepository) Create(ctx context.Context, meeting *entity.Meeting) error {
	now := time.Now()
	meeting.CreatedAt = now
	meeting.UpdatedAt = now

	// NULL handling is done automatically by entityToFieldMap which uses DefaultNullableStrategy
	return r.generic.Create(ctx, meeting)
}

// Get retrieves a meeting by ID and user Telegram ID
func (r *meetingRepository) Get(ctx context.Context, id, telegramID int64) (*entity.Meeting, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid id")
	}

	sqlQuery := `SELECT m.id, 
		m.created_at,
		m.updated_at,
		m.user_id, 
		m.title, 
		m.transcription,
		m.summary,
		m.audio_file_id,
		m.duration_seconds
	FROM meetings m
	INNER JOIN users u ON m.user_id = u.id
	WHERE m.id = $1 AND u.telegram_id = $2`

	var meeting entity.Meeting
	var summary sql.NullString
	var transcription sql.NullString

	err := r.pool.QueryRow(ctx, sqlQuery, id, telegramID).Scan(
		&meeting.ID,
		&meeting.CreatedAt,
		&meeting.UpdatedAt,
		&meeting.UserID,
		&meeting.Title,
		&transcription,
		&summary,
		&meeting.AudioFileID,
		&meeting.DurationSeconds)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("meeting not found")
		}
		return nil, err
	}

	// Unified NULL handling using NullHandler
	meeting.Transcription = r.nullHandler.FromNullString(transcription)
	meeting.Summary = r.nullHandler.FromNullString(summary)

	return &meeting, nil
}

// List retrieves all meetings for a given user
func (r *meetingRepository) List(ctx context.Context, userID int64) ([]entity.Meeting, error) {
	sqlQuery := `SELECT m.id,
		m.created_at,
		m.updated_at,
		m.user_id,
		m.title,
		m.transcription,
		m.summary,
		m.audio_file_id,
		m.duration_seconds
	FROM meetings m
	INNER JOIN users u ON m.user_id = u.id
	WHERE u.telegram_id = $1
	ORDER BY m.created_at, m.updated_at`

	rows, err := r.pool.Query(ctx, sqlQuery, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var meetings []entity.Meeting
	for rows.Next() {
		var meeting entity.Meeting
		var summary sql.NullString
		var transcription sql.NullString

		err = rows.Scan(
			&meeting.ID,
			&meeting.CreatedAt,
			&meeting.UpdatedAt,
			&meeting.UserID,
			&meeting.Title,
			&transcription,
			&summary,
			&meeting.AudioFileID,
			&meeting.DurationSeconds)
		if err != nil {
			return nil, err
		}

		// Unified NULL handling using NullHandler
		meeting.Transcription = r.nullHandler.FromNullString(transcription)
		meeting.Summary = r.nullHandler.FromNullString(summary)

		meetings = append(meetings, meeting)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return meetings, nil
}

// buildTsQuery safely builds a tsquery from keywords with limits
func (r *meetingRepository) buildTsQuery(keywords []string) string {
	if len(keywords) == 0 {
		return ""
	}

	const maxKeywords = 10
	if len(keywords) > maxKeywords {
		keywords = keywords[:maxKeywords]
	}

	const maxKeywordLength = 100
	sanitizedKeywords := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		if len(keyword) > maxKeywordLength {
			keyword = keyword[:maxKeywordLength]
		}
		sanitized := r.generic.sanitizeTsQuery(keyword)
		if sanitized != "" && len(sanitized) > 0 {
			sanitizedKeywords = append(sanitizedKeywords, sanitized)
		}
	}

	if len(sanitizedKeywords) == 0 {
		return ""
	}

	return strings.Join(sanitizedKeywords, " & ")
}

// SearchByKeywords performs a full-text search on meetings by keywords
func (r *meetingRepository) SearchByKeywords(ctx context.Context, req entity.SearchRequest) ([]entity.TranscriptionRecord, error) {
	if len(req.Keywords) == 0 {
		return []entity.TranscriptionRecord{}, nil
	}

	tsQuery := r.buildTsQuery(req.Keywords)
	if tsQuery == "" {
		return []entity.TranscriptionRecord{}, nil
	}

	query := `
		SELECT id, user_id, transcription
		FROM meetings
		WHERE user_id = (
			SELECT id
			FROM users
			WHERE telegram_id = $1
		)
		AND to_tsvector('russian', transcription) @@ to_tsquery('russian', $2)
		ORDER BY ts_rank(to_tsvector('russian', transcription), to_tsquery('russian', $2)) DESC
		LIMIT 100
	`

	rows, err := r.pool.Query(ctx, query, req.UserID, tsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()

	var records []entity.TranscriptionRecord
	for rows.Next() {
		var record entity.TranscriptionRecord
		var transcription sql.NullString

		err = rows.Scan(
			&record.ID,
			&record.UserID,
			&transcription,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Unified NULL handling using NullHandler
		record.Transcription = r.nullHandler.FromNullString(transcription)
		records = append(records, record)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return records, nil
}

// Альтернативная реализация с итератором
func (r *meetingRepository) SearchByKeywordsIter(
	ctx context.Context,
	req entity.SearchRequest,
) iter.Seq2[entity.TranscriptionRecord, error] {
	return func(yield func(entity.TranscriptionRecord, error) bool) {
		if len(req.Keywords) == 0 {
			return
		}

		tsQuery := r.buildTsQuery(req.Keywords)
		if tsQuery == "" {
			return
		}

		query := `SELECT id, user_id, transcription
                 FROM meetings
                 WHERE user_id = (SELECT id FROM users WHERE telegram_id = $1)
                 AND to_tsvector('russian', transcription) @@ to_tsquery('russian', $2)
                 ORDER BY ts_rank(...) DESC
                 LIMIT 100`

		rows, err := r.pool.Query(ctx, query, req.UserID, tsQuery)
		if err != nil {
			yield(entity.TranscriptionRecord{}, err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var record entity.TranscriptionRecord
			var transcription sql.NullString

			if err := rows.Scan(&record.ID, &record.UserID, &transcription); err != nil {
				if !yield(entity.TranscriptionRecord{}, err) {
					return
				}
				continue
			}

			record.Transcription = r.nullHandler.FromNullString(transcription)

			if !yield(record, nil) {
				return // consumer stopped iteration
			}
		}

		if err := rows.Err(); err != nil {
			yield(entity.TranscriptionRecord{}, err)
		}
	}
}
