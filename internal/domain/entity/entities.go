package entity

import "time"

// User represents a user of the bot.
type User struct {
	ID         int64     // Unique identifier
	CreatedAt  time.Time // Creation timestamp
	UpdatedAt  time.Time // Last update timestamp
	TelegramID int64     // Telegram user ID
	UserName   string    // Telegram username
	FirstName  string    // User's first name
	LastName   string    // User's last name
}

// Meeting represents a recorded meeting or audio session.
type Meeting struct {
	ID              int64     // Unique identifier
	CreatedAt       time.Time // Creation timestamp
	UpdatedAt       time.Time // Last update timestamp
	UserID          int64     // Reference to the user who owns this meeting
	Title           string    // Meeting title
	Transcription   string    // Transcribed text from the audio
	Summary         string    // Summary of the meeting
	AudioFileID     string    // Reference to the stored audio file
	DurationSeconds int       // Meeting duration in seconds
}

// TranscriptionRecord represents a transcription record for search results.
type TranscriptionRecord struct {
	ID            int64  `db:"id"`            // Unique identifier
	UserID        int64  `db:"user_id"`       // Reference to the user
	Transcription string `db:"transcription"` // Transcribed text content
	// Add other fields according to your table schema
}

// SearchRequest represents a search query for meetings.
type SearchRequest struct {
	UserID   int64    // User requesting the search
	Keywords []string // Keywords to search for
	Limit    int
}

// Message represents a chat message to be stored.
type Message struct {
	ID       int64  `json:"id"`
	ChatID   int64  `json:"chat_id"`
	UserID   int64  `json:"user_id"`
	Username string `json:"username,omitempty"`
	Text     string `json:"text"`
	Type     string `json:"type"`
}

// Media represents a media file to be stored.
type Media struct {
	ID           int64  `json:"id"`
	ChatID       int64  `json:"chat_id"`
	UserID       int64  `json:"user_id"`
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id,omitempty"`
	FileName     string `json:"file_name,omitempty"`
	MimeType     string `json:"mime_type,omitempty"`
	FileSize     int64  `json:"file_size,omitempty"`
	Duration     int    `json:"duration,omitempty"`
	Bitrate      int    `json:"bitrate,omitempty"`
	Type         string `json:"type"`
	Caption      string `json:"caption,omitempty"`
	FilePath     string `json:"file_path,omitempty"`
}
