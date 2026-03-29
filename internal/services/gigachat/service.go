package gigachatservice

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/skiphead/go-gigachat"
	"github.com/skiphead/salutespeech/client"
	"github.com/skiphead/salutespeech/types"
)

// Client defines the interface for GigaChat operations.
type Client interface {
	Completion(ctx context.Context, systemContent, userContent string) (*gigachat.ChatResponse, error)
}

// clientImpl implements Client using the GigaChat API.
type clientImpl struct {
	clientChat *gigachat.Client
	logger     *slog.Logger
}

// NewClient creates a new GigaChat client instance.
func NewClient(clientID, clientSecret string, logger *slog.Logger) (Client, error) {
	basicAuth := client.GenerateBasicAuthKey(clientID, clientSecret)

	// Create OAuth client
	oauthClient, err := client.NewOAuthClient(client.Config{
		AuthKey: basicAuth,
		Scope:   types.ScopeGigaChatAPIPers,
		Timeout: 30 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth client: %w", err)
	}

	// Create token manager
	tokenMgr := client.NewTokenManager(oauthClient, client.TokenManagerConfig{})

	clientChat, err := gigachat.NewClient(tokenMgr, gigachat.Config{
		BaseURL:       "https://gigachat.devices.sberbank.ru/api/v1",
		AllowInsecure: false,
		Timeout:       30 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create GigaChat client: %w", err)
	}

	return &clientImpl{
		clientChat: clientChat,
		logger:     logger,
	}, nil
}

// Completion sends a chat completion request to GigaChat.
func (s *clientImpl) Completion(ctx context.Context, systemContent, userContent string) (*gigachat.ChatResponse, error) {
	// Validate user content
	if userContent == "" {
		err := fmt.Errorf("userContent cannot be empty")
		s.logger.ErrorContext(ctx, "validation failed",
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	// Log request
	s.logger.InfoContext(ctx, "sending completion request",
		slog.Int("system_content_length", len(systemContent)),
		slog.Int("user_content_length", len(userContent)),
		slog.String("model", gigachat.ModelGigaChatProPreview.String()),
	)

	chatReq := &gigachat.ChatRequest{
		Model: gigachat.ModelGigaChatProPreview.String(),
		Messages: []gigachat.Message{
			{
				Role:    gigachat.RoleSystem,
				Content: systemContent,
			},
			{
				Role:    gigachat.RoleUser,
				Content: userContent,
			},
		},
	}

	// Use provided context with timeout instead of context.Background()
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	startTime := time.Now()
	response, err := s.clientChat.Completion(ctxWithTimeout, chatReq)
	duration := time.Since(startTime)

	if err != nil {
		// Log error with context and wrap it with additional context
		s.logger.ErrorContext(ctx, "completion request failed",
			slog.String("error", err.Error()),
			slog.Int64("duration_ms", duration.Milliseconds()),
			slog.Int("system_content_length", len(systemContent)),
			slog.Int("user_content_length", len(userContent)),
		)
		// Wrap error with context
		return nil, fmt.Errorf("gigachat completion failed: %w", err)
	}

	// Log successful response
	if response != nil && len(response.Choices) > 0 {
		s.logger.InfoContext(ctx, "completion request succeeded",
			slog.Int64("duration_ms", duration.Milliseconds()),
			slog.Int("response_length", len(response.Choices[0].Message.Content)),
			slog.String("finish_reason", string(response.Choices[0].FinishReason)),
		)
	} else {
		s.logger.WarnContext(ctx, "completion request succeeded but response is empty",
			slog.Int64("duration_ms", duration.Milliseconds()),
		)
	}

	return response, nil
}
