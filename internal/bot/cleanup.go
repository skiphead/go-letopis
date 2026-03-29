package bot

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// CleanupOldTempFiles удаляет устаревшие временные файлы, пропуская активные.
func (b *Bot) CleanupOldTempFiles(ctx context.Context, maxAge time.Duration) error {
	logger := b.logger.With(slog.String("operation", "cleanup"))

	entries, err := os.ReadDir(b.tempDir)
	if err != nil {
		return fmt.Errorf("failed to read temp dir: %w", err)
	}

	stats := b.processCleanupEntries(ctx, entries, maxAge, logger)
	logger.Info("Cleanup completed",
		slog.Int("deleted", stats.deleted),
		slog.Int("skipped_active", stats.skippedActive),
		slog.Int("failed", stats.failed),
	)
	return nil
}

// cleanupStats хранит статистику операции очистки.
type cleanupStats struct {
	deleted       int
	skippedActive int
	failed        int
}

// processCleanupEntries обрабатывает список файлов и возвращает статистику.
func (b *Bot) processCleanupEntries(ctx context.Context, entries []os.DirEntry, maxAge time.Duration, logger *slog.Logger) cleanupStats {
	stats := cleanupStats{}
	now := time.Now()

	for _, entry := range entries {
		// Проверяем контекст на отмену
		select {
		case <-ctx.Done():
			logger.Warn("Cleanup interrupted by context",
				slog.String("reason", ctx.Err().Error()),
				slog.Int("deleted", stats.deleted),
				slog.Int("skipped_active", stats.skippedActive),
				slog.Int("failed", stats.failed),
			)
			return stats
		default:
		}

		if entry.IsDir() {
			continue
		}
		b.processSingleFile(ctx, filepath.Join(b.tempDir, entry.Name()), now, maxAge, logger, &stats)
	}
	return stats
}

// processSingleFile обрабатывает один файл: проверяет активность, возраст и удаляет при необходимости.
func (b *Bot) processSingleFile(ctx context.Context, filePath string, now time.Time, maxAge time.Duration, logger *slog.Logger, stats *cleanupStats) {
	if b.isFileActive(filePath) {
		logger.Debug("Skipping active file", slog.String("path", filePath))
		stats.skippedActive++
		return
	}

	info, err := os.Stat(filePath)
	if err != nil {
		logger.Warn("Failed to get file info",
			slog.String("path", filePath),
			slog.String("error", err.Error()),
		)
		stats.failed++
		return
	}

	if now.Sub(info.ModTime()) > maxAge {
		b.tryRemoveOldFile(ctx, filePath, info, now, logger, stats)
	}
}

// tryRemoveOldFile пытается удалить устаревший файл и обновляет статистику.
func (b *Bot) tryRemoveOldFile(ctx context.Context, filePath string, info os.FileInfo, now time.Time, logger *slog.Logger, stats *cleanupStats) {
	if err := os.Remove(filePath); err != nil {
		if !os.IsNotExist(err) {
			logger.Warn("Failed to remove old file",
				slog.String("path", filePath),
				slog.String("error", err.Error()),
			)
			stats.failed++
		}
		// Если файл не существует, просто игнорируем (не увеличиваем статистику ошибок)
	} else {
		logger.Info("Removed old temp file",
			slog.String("path", filePath),
			slog.Duration("age", now.Sub(info.ModTime())),
		)
		stats.deleted++
	}
}
