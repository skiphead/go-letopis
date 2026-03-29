package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gopkg.in/telebot.v3"
)

// updateMetadata contains extracted update data for logging.
type updateMetadata struct {
	updateID string
	chatID   int64
	userID   int64
	username string
	handler  string
}

// loggingMiddleware logs handler execution: timing, user, and result.
func (b *Bot) loggingMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		start := time.Now()
		meta := extractUpdateMetadata(c)

		err := next(c)
		b.logHandlerResult(meta, time.Since(start), err)

		return err
	}
}

// extractUpdateMetadata extracts metadata from the update context.
func extractUpdateMetadata(c telebot.Context) updateMetadata {
	meta := updateMetadata{
		updateID: fmt.Sprintf("%d", c.Update().ID),
		username: "anonymous",
		handler:  "unknown",
	}

	if msg := c.Message(); msg != nil {
		meta.handler = determineMessageHandler(msg)
	} else if cb := c.Callback(); cb != nil {
		handler := "callback"
		if cb.Data != "" {
			// Sanitize callback data: log only the first part if it's a structured callback
			if parts := strings.Fields(cb.Data); len(parts) > 0 && strings.HasPrefix(parts[0], "/") {
				handler = "cb:" + parts[0]
			} else {
				handler = "callback:data"
			}
		}
		meta.handler = handler
	}

	if sender := c.Sender(); sender != nil {
		meta.userID = sender.ID
		meta.username = resolveUsername(sender)
	}

	if chat := c.Chat(); chat != nil {
		meta.chatID = chat.ID
	}

	return meta
}

// determineMessageHandler determines the handler type based on message content.
func determineMessageHandler(msg *telebot.Message) string {
	switch {
	case msg.Text != "":
		if strings.HasPrefix(strings.TrimSpace(msg.Text), "/") {
			parts := strings.Fields(msg.Text)
			if len(parts) > 0 {
				// Only log the command name, not arguments
				return "cmd:" + parts[0]
			}
		}
		return "text"
	case msg.Audio != nil:
		return "audio"
	case msg.Voice != nil:
		return "voice"
	case msg.Photo != nil:
		return "photo"
	case msg.Document != nil:
		return "document"
	case msg.Contact != nil:
		return "contact"
	case msg.Location != nil:
		return "location"
	default:
		return "unknown"
	}
}

// resolveUsername returns the username or a generated fallback.
func resolveUsername(user *telebot.User) string {
	if user == nil {
		return "anonymous"
	}
	if user.Username != "" {
		return "@" + user.Username
	}
	if user.FirstName != "" {
		return user.FirstName
	}
	return fmt.Sprintf("user_%d", user.ID)
}

// logHandlerResult logs the handler execution result.
func (b *Bot) logHandlerResult(meta updateMetadata, duration time.Duration, err error) {
	attrs := []slog.Attr{
		slog.String("update_id", meta.updateID),
		slog.Int64("chat_id", meta.chatID),
		slog.Int64("user_id", meta.userID),
		slog.String("username", meta.username),
		slog.String("handler", meta.handler),
		slog.Duration("duration", duration),
	}

	level := slog.LevelInfo
	message := "Handler completed"
	if err != nil {
		level = slog.LevelError
		message = "Handler error"
		attrs = append(attrs, slog.String("error", err.Error()))
	}

	b.logger.LogAttrs(context.Background(), level, message, attrs...)
}
