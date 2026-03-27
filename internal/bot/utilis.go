package bot

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/skiphead/go-letopis/internal/domain/entity"
	"gopkg.in/telebot.v3"
)

var mimeToExt = map[string]string{
	"audio/mpeg": ".mp3", "audio/mp3": ".mp3", "audio/x-mpeg": ".mp3", "audio/x-mp3": ".mp3",
	"audio/mp4": ".m4a", "audio/m4a": ".m4a", "audio/x-m4a": ".m4a",
	"audio/ogg": ".ogg", "audio/opus": ".ogg", "audio/x-ogg": ".ogg",
	"audio/wav": ".wav", "audio/x-wav": ".wav", "audio/vnd.wave": ".wav",
	"audio/webm": ".webm",
}

// sendSafe sends a message safely, logging any errors.
func (b *Bot) sendSafe(c telebot.Context, text string, mode telebot.ParseMode) {
	if err := c.Send(text, mode); err != nil {
		chatID := int64(0)
		if chat := c.Chat(); chat != nil {
			chatID = chat.ID
		}
		b.logger.Warn("Failed to send message",
			slog.String("error", err.Error()),
			slog.Int64("chat_id", chatID),
			slog.String("text_preview", truncateString(text, 50)),
		)
	}
}

// sendSafeToChat sends a message to a specific chat safely, logging any errors.
func (b *Bot) sendSafeToChat(chatID int64, text string, mode telebot.ParseMode) {
	_, err := b.Bot.Send(&telebot.Chat{ID: chatID}, text, mode)
	if err != nil {
		b.logger.Warn("Failed to send message to chat",
			slog.String("error", err.Error()),
			slog.Int64("chat_id", chatID),
			slog.String("text_preview", truncateString(text, 50)),
		)
	}
}

// truncateString truncates a string to the specified length.
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// markFileActive marks a file as actively being processed.
func (b *Bot) markFileActive(path string) {
	b.activeFiles.Store(path, true)
	b.logger.Debug("Marked file as active", slog.String("path", path))
}

// markFileInactive marks a file as no longer being processed.
func (b *Bot) markFileInactive(path string) {
	b.activeFiles.Delete(path)
	b.logger.Debug("Marked file as inactive", slog.String("path", path))
}

// isFileActive checks if a file is currently active.
func (b *Bot) isFileActive(path string) bool {
	_, ok := b.activeFiles.Load(path)
	return ok
}

// downloadAudioToTemp downloads a file to a temporary location.
func (b *Bot) downloadAudioToTemp(ctx context.Context, file *telebot.File, fileName, mimeType string) (string, error) {
	ext := getFileExtension(fileName, mimeType, b.logger)
	tempFile, err := os.CreateTemp(b.tempDir, "audio_*"+ext)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	err = tempFile.Close()
	if err != nil {
		return "", err
	}

	done := make(chan error, 1)
	go func() { done <- b.Download(file, tempFile.Name()) }()

	select {
	case err := <-done:
		if err != nil {
			_ = os.Remove(tempFile.Name())
			return "", fmt.Errorf("failed to download: %w", err)
		}

		b.logger.Info("File downloaded",
			slog.String("path", tempFile.Name()),
			slog.Int64("size", file.FileSize),
		)
		return tempFile.Name(), nil
	case <-ctx.Done():
		_ = os.Remove(tempFile.Name())
		return "", fmt.Errorf("download cancelled: %w", ctx.Err())
	}
}

// getFileExtension determines the file extension from filename or MIME type.
func getFileExtension(fileName, mimeType string, logger *slog.Logger) string {
	if ext := filepath.Ext(fileName); ext != "" {
		return ext
	}

	extension := resolveExtensionByMIME(mimeType)
	if extension == "" {
		logger.Debug("Unknown MIME type", slog.String("mime", mimeType))
		return ".bin"
	}
	return extension
}

// resolveExtensionByMIME returns the file extension for a given MIME type.
func resolveExtensionByMIME(mimeType string) string {
	if ext, ok := mimeToExt[mimeType]; ok {
		return ext
	}
	return ".bin"
}

// formatFileSize formats a file size in bytes to a human-readable string.
func formatFileSize(bytes int64) string {
	const unit = 1024
	units := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}

	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	if exp >= len(units) {
		return fmt.Sprintf("%.2f EB", float64(bytes)/float64(div))
	}
	return fmt.Sprintf("%.2f %s", float64(bytes)/float64(div), units[exp])
}

// escapeHTML escapes special HTML characters.
func escapeHTML(s string) string {
	return html.EscapeString(s)
}

// copyTelebotFile creates a copy of a telebot.File.
func copyTelebotFile(f telebot.File) telebot.File {
	return telebot.File{
		FileID:   f.FileID,
		UniqueID: f.UniqueID,
		FileSize: f.FileSize,
	}
}

// tryEnqueueJob attempts to enqueue a job, returning false if the queue is full.
func (b *Bot) tryEnqueueJob(job *processJob) bool {
	select {
	case b.jobQueue <- job:
		return true
	default:
		return false
	}
}

// FormatMeetingsList formats a list of meetings into a readable text.
func FormatMeetingsList(meetings []entity.Meeting) string {
	if len(meetings) == 0 {
		return "❌ По вашему запросу ничего не найдено"
	}

	var sb strings.Builder
	sb.WriteString("📋 <b>Ваши встречи:</b>\n\n")

	for i, meeting := range meetings {
		title := getMeetingTitle(meeting)
		sb.WriteString(fmt.Sprintf("%d. <b>%s</b>\n", i+1, title))
		sb.WriteString(fmt.Sprintf("   <code>ID: %d</code>\n\n", meeting.ID))
	}

	sb.WriteString("💡 <i>Для просмотра деталей используйте ID встречи</i>")
	return sb.String()
}

// FormatSearchResult formats a list of search results into a readable text.
func FormatSearchResult(meetings []entity.TranscriptionRecord) string {
	if len(meetings) == 0 {
		return "❌ У вас пока нет сохраненных встреч"
	}

	var sb strings.Builder
	sb.WriteString("📋 <b>Результат поиска:</b>\n\n")

	for i, meeting := range meetings {
		sb.WriteString(fmt.Sprintf("%d. <b>%s</b>\n", i+1, meeting.Transcription))
		sb.WriteString(fmt.Sprintf("   <code>ID: %d</code>\n\n", meeting.ID))
	}

	sb.WriteString("💡 <i>Для просмотра деталей используйте /get ID встречи </i>")
	return sb.String()
}

// getMeetingTitle returns the meeting title or a default value.
func getMeetingTitle(meeting entity.Meeting) string {
	if meeting.Title != "" {
		return meeting.Title
	}
	return fmt.Sprintf("Встреча #%d", meeting.ID)
}
