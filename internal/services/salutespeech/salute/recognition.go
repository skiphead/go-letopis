package salute

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/skiphead/salutespeech/client"
	"github.com/skiphead/salutespeech/recognition/async"
	"github.com/skiphead/salutespeech/types"
	"github.com/skiphead/salutespeech/upload"
	"github.com/skiphead/salutespeech/utils"
)

// Client defines the interface for Salute Speech operations.
type Client interface {
	Upload(ctx context.Context, pathAudioFile string) (*async.Request, error)
	CreateTask(ctx context.Context, req *async.Request) (*async.Response, error)
	WaitForResult(ctx context.Context, responseResultID string) (*async.TaskResult, error)
	ExtractText(ctx context.Context, fileID string) (string, error)
}

// clientImpl implements Client using the Salute Speech API.
type clientImpl struct {
	clientAsync  *async.Client
	clientUpload upload.Client
	logger       *slog.Logger
}

// NewClient creates a new Salute Speech client instance.
func NewClient(clientID, clientSecret string, logger *slog.Logger) (Client, error) {
	// Input validation
	if clientID == "" {
		return nil, fmt.Errorf("clientID cannot be empty")
	}
	if clientSecret == "" {
		return nil, fmt.Errorf("clientSecret cannot be empty")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	basicAuth := client.GenerateBasicAuthKey(clientID, clientSecret)

	// Create OAuth client
	oauthClient, err := client.NewOAuthClient(client.Config{
		AuthKey: basicAuth,
		Scope:   types.ScopeSaluteSpeechPers,
		Timeout: 30 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth client: %w", err)
	}

	// Create token manager
	tokenMgr := client.NewTokenManager(oauthClient, client.TokenManagerConfig{})

	clientUpload, err := upload.NewClient(tokenMgr, upload.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create upload client: %w", err)
	}

	clientAsync, err := async.NewClient(tokenMgr, async.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create async client: %w", err)
	}

	return &clientImpl{
		clientUpload: clientUpload,
		clientAsync:  clientAsync,
		logger:       logger,
	}, nil
}

// CreateTask creates a new recognition task.
func (s *clientImpl) CreateTask(ctx context.Context, req *async.Request) (*async.Response, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.RequestFileID == "" {
		return nil, fmt.Errorf("request file ID cannot be empty")
	}

	resp, err := s.clientAsync.CreateTask(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}
	return resp, nil
}

// WaitForResult waits for the recognition task to complete.
func (s *clientImpl) WaitForResult(ctx context.Context, responseResultID string) (*async.TaskResult, error) {
	// Input validation
	if responseResultID == "" {
		return nil, fmt.Errorf("response result ID cannot be empty")
	}

	result, err := s.clientAsync.WaitForResult(ctx, responseResultID, 2*time.Second, 5*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for result: %w", err)
	}
	return result, nil
}

// ExtractText extracts text from the recognition result.
func (s *clientImpl) ExtractText(ctx context.Context, responseFileID string) (string, error) {
	// Input validation
	if responseFileID == "" {
		return "", fmt.Errorf("response file ID cannot be empty")
	}

	byteData, err := s.clientAsync.DownloadTaskResult(ctx, responseFileID)
	if err != nil {
		return "", fmt.Errorf("failed to download task result: %w", err)
	}

	text, err := ExtractTextFromResults(byteData)
	if err != nil {
		return "", fmt.Errorf("failed to extract text from results: %w", err)
	}
	return text, nil
}

// Upload uploads an audio file for recognition.
func (s *clientImpl) Upload(ctx context.Context, pathAudioFile string) (*async.Request, error) {
	// Input validation
	if pathAudioFile == "" {
		return nil, fmt.Errorf("audio file path cannot be empty")
	}

	audioType, detectErr := utils.DetectAudioContentType(pathAudioFile)
	if detectErr != nil {
		s.logger.Error("Failed to detect audio type",
			slog.String("error", detectErr.Error()),
			slog.String("path", pathAudioFile))

		audioType = "audio/ogg"
		s.logger.Warn("Using default audio type", slog.String("audio_type", string(audioType)))
	}

	uploadResp, err := s.clientUpload.UploadFromFile(ctx, pathAudioFile, audioType)
	if err != nil {
		return nil, fmt.Errorf("failed to upload audio file %s: %w", pathAudioFile, err)
	}

	if uploadResp == nil || uploadResp.Result.RequestFileID == "" {
		return nil, fmt.Errorf("upload response is invalid: missing request file ID")
	}

	return &async.Request{
		RequestFileID: uploadResp.Result.RequestFileID,
		Options: &async.Options{
			AudioEncoding: async.EncodingOGG_OPUS,
			SampleRate:    16000,
			Model:         async.ModelGeneral,
			Language:      "ru-RU",
		},
	}, nil
}
