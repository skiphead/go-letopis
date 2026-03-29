package salute

import (
	"encoding/json"
	"fmt"
	"strings"
)

// WordAlignment represents the alignment of a word with timestamps.
type WordAlignment struct {
	Word  string `json:"word"`
	Start string `json:"start"`
	End   string `json:"end"`
}

// Result represents a single transcription result.
type Result struct {
	Text           string          `json:"text"`
	NormalizedText string          `json:"normalized_text"`
	Start          string          `json:"start"`
	End            string          `json:"end"`
	WordAlignments []WordAlignment `json:"word_alignments"`
}

// EmotionsResult represents emotion analysis results.
type EmotionsResult struct {
	Positive float64 `json:"positive"`
	Neutral  float64 `json:"neutral"`
	Negative float64 `json:"negative"`
}

// BackendInfo contains information about the transcription backend.
type BackendInfo struct {
	ModelName     string `json:"model_name"`
	ModelVersion  string `json:"model_version"`
	ServerVersion string `json:"server_version"`
}

// SpeakerInfo contains information about the detected speaker.
type SpeakerInfo struct {
	SpeakerID             int     `json:"speaker_id"`
	MainSpeakerConfidence float64 `json:"main_speaker_confidence"`
}

// PersonIdentity represents demographic information about the speaker.
type PersonIdentity struct {
	Age         string  `json:"age"`
	Gender      string  `json:"gender"`
	AgeScore    float64 `json:"age_score"`
	GenderScore float64 `json:"gender_score"`
}

// TranscriptionData represents complete transcription data from the API.
type TranscriptionData struct {
	Results             []Result       `json:"results"`
	Eou                 bool           `json:"eou"`
	EmotionsResult      EmotionsResult `json:"emotions_result"`
	ProcessedAudioStart string         `json:"processed_audio_start"`
	ProcessedAudioEnd   string         `json:"processed_audio_end"`
	BackendInfo         BackendInfo    `json:"backend_info"`
	Channel             int            `json:"channel"`
	SpeakerInfo         SpeakerInfo    `json:"speaker_info"`
	EouReason           string         `json:"eou_reason"`
	Insight             string         `json:"insight"`
	PersonIdentity      PersonIdentity `json:"person_identity"`
}

// ExtractTextFromResults deserializes JSON data and extracts all text fields from results.
func ExtractTextFromResults(data []byte) (string, error) {
	var transcriptions []TranscriptionData

	err := json.Unmarshal(data, &transcriptions)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	var texts []string
	for _, item := range transcriptions {
		for _, result := range item.Results {
			if result.Text != "" {
				texts = append(texts, result.Text)
			}
		}
	}

	return strings.Join(texts, " "), nil
}

// ExtractTextFromTranscriptions extracts text from already deserialized transcription data.
func ExtractTextFromTranscriptions(transcriptions []TranscriptionData) string {
	var texts []string

	for _, item := range transcriptions {
		for _, result := range item.Results {
			if result.Text != "" {
				texts = append(texts, result.Text)
			}
		}
	}

	return strings.Join(texts, " ")
}
