package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/skiphead/go-letopis/internal/domain/entity"
	"github.com/skiphead/go-letopis/internal/domain/repository"
	gigachatservice "github.com/skiphead/go-letopis/internal/services/gigachat"
	"github.com/skiphead/go-letopis/internal/services/salutespeech/salute"
)

// AIUseCase defines the interface for AI-related operations.
type AIUseCase interface {
	Recognition(ctx context.Context, media *entity.Media) (string, error)
	Chat(ctx context.Context, text string) (string, error)
}

// aiUseCase implements AIUseCase using Salute Speech and GigaChat services.
type aiUseCase struct {
	userRepo           repository.UserRepository
	meetingRepo        repository.MeetingRepository
	logger             *slog.Logger
	saluteSpeechClient salute.Client
	gigaChatClient     gigachatservice.Client
}

// NewAIUseCase creates a new AIUseCase instance.
func NewAIUseCase(
	userRepo repository.UserRepository,
	meetingRepo repository.MeetingRepository,
	saluteSpeechClient salute.Client,
	gigaChatClient gigachatservice.Client,
	logger *slog.Logger,
) AIUseCase {
	return &aiUseCase{
		userRepo:           userRepo,
		meetingRepo:        meetingRepo,
		logger:             logger,
		saluteSpeechClient: saluteSpeechClient,
		gigaChatClient:     gigaChatClient,
	}
}

const (
	// defaultSystemPrompt is the default system prompt for GigaChat.
	defaultSystemPrompt = "Ты полезный ассистент для обработки встреч. Отвечай кратко и по делу."

	// maxTranscriptionLength is the maximum length of transcription to process.
	maxTranscriptionLength = 10000
)

// Recognition processes audio recognition and generates a summary.
func (uc *aiUseCase) Recognition(ctx context.Context, media *entity.Media) (string, error) {
	if err := uc.validateMedia(media); err != nil {
		return "", err
	}

	start := time.Now()
	uc.logger.Info("Starting audio recognition",
		slog.String("file_name", media.FileName),
		slog.Int64("user_id", media.UserID),
		slog.Int64("file_size", media.FileSize),
	)

	// Upload and recognize speech
	textExtract, err := uc.recognizeSpeech(ctx, media)
	if err != nil {
		return "", err
	}

	// Generate summary using AI
	summaryText, err := uc.generateSummary(ctx, textExtract)
	if err != nil {
		return "", err
	}

	// Save meeting to database
	if err := uc.saveMeeting(ctx, media, textExtract, summaryText); err != nil {
		return "", err
	}

	uc.logger.Info("Audio recognition completed",
		slog.Duration("duration", time.Since(start)),
		slog.Int("transcription_length", len(textExtract)),
		slog.Int("summary_length", len(summaryText)),
	)

	return summaryText, nil
}

// validateMedia validates the media input.
func (uc *aiUseCase) validateMedia(media *entity.Media) error {
	if media == nil {
		return fmt.Errorf("media cannot be nil")
	}
	if media.FilePath == "" {
		return fmt.Errorf("media file path cannot be empty")
	}
	if media.UserID == 0 {
		return fmt.Errorf("media user ID cannot be 0")
	}
	return nil
}

// recognizeSpeech performs speech recognition on the audio file.
func (uc *aiUseCase) recognizeSpeech(ctx context.Context, media *entity.Media) (string, error) {
	uc.logger.Debug("Uploading to Salute Speech", slog.String("path", media.FilePath))

	asyncReq, err := uc.saluteSpeechClient.Upload(ctx, media.FilePath)
	if err != nil {
		uc.logger.Error("Upload failed", slog.String("error", err.Error()))
		return "", fmt.Errorf("upload speech video: %w", err)
	}

	resp, err := uc.saluteSpeechClient.CreateTask(ctx, asyncReq)
	if err != nil {
		return "", fmt.Errorf("create task: %w", err)
	}

	waitResult, err := uc.saluteSpeechClient.WaitForResult(ctx, resp.Result.ID)
	if err != nil {
		return "", fmt.Errorf("wait result: %w", err)
	}

	textExtract, err := uc.saluteSpeechClient.ExtractText(ctx, waitResult.Result.ResponseFileId)
	if err != nil {
		return "", fmt.Errorf("extract text: %w", err)
	}

	uc.logger.Debug("Text extracted", slog.Int("length", len(textExtract)))

	// Truncate if too long to prevent API issues
	if len(textExtract) > maxTranscriptionLength {
		uc.logger.Warn("Transcription too long, truncating",
			slog.Int("original_length", len(textExtract)),
			slog.Int("max_length", maxTranscriptionLength))
		textExtract = textExtract[:maxTranscriptionLength]
	}

	return textExtract, nil
}

// generateSummary generates a summary of the transcription using GigaChat.
func (uc *aiUseCase) generateSummary(ctx context.Context, textExtract string) (string, error) {
	year, month, day := time.Now().Date()
	sysContent := fmt.Sprintf("Сегодня %d-%02d-%02d. %s", year, month, day, documentPrompt)

	summary, err := uc.gigaChatClient.Completion(ctx, sysContent, textExtract)
	if err != nil {
		return "", fmt.Errorf("completion: %w", err)
	}

	summaries := make([]string, 0, len(summary.Choices))
	for _, choice := range summary.Choices {
		if choice.Message.Content != "" {
			summaries = append(summaries, choice.Message.Content)
		}
	}

	if len(summaries) == 0 {
		return "", fmt.Errorf("no valid content in GigaChat response")
	}

	return strings.Join(summaries, " "), nil
}

// saveMeeting saves the meeting data to the database.
func (uc *aiUseCase) saveMeeting(ctx context.Context, media *entity.Media, textExtract, summaryText string) error {
	user, err := uc.userRepo.Get(ctx, media.UserID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	err = uc.meetingRepo.Create(ctx, &entity.Meeting{
		UserID:          user.ID,
		Title:           uc.getMeetingTitle(media),
		Transcription:   textExtract,
		Summary:         summaryText,
		AudioFileID:     media.FileID,
		DurationSeconds: media.Duration,
	})
	if err != nil {
		return fmt.Errorf("create meeting: %w", err)
	}

	return nil
}

// getMeetingTitle returns a title for the meeting.
func (uc *aiUseCase) getMeetingTitle(media *entity.Media) string {
	if media.FileName != "" {
		return media.FileName
	}
	return fmt.Sprintf("Meeting %s", time.Now().Format("2006-01-02 15:04:05"))
}

// Chat handles chat interactions with the AI.
func (uc *aiUseCase) Chat(ctx context.Context, text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("chat text cannot be empty")
	}

	response, err := uc.gigaChatClient.Completion(ctx, consultationPrompt, text)
	if err != nil {
		return "", fmt.Errorf("completion: %w", err)
	}

	responses := make([]string, 0, len(response.Choices))
	for _, choice := range response.Choices {
		if choice.Message.Content != "" {
			responses = append(responses, choice.Message.Content)
		}
	}

	if len(responses) == 0 {
		return "", fmt.Errorf("no valid content in GigaChat response")
	}

	return strings.Join(responses, " "), nil
}
