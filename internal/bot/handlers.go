package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/skiphead/go-letopis/internal/domain/entity"
	"gopkg.in/telebot.v3"
)

// registerHandlers registers all command and event handlers.
func (b *Bot) registerHandlers() {
	b.Handle("/start", b.onStart)
	b.Handle("/help", b.onHelp)
	b.Handle("/list", b.listByUserID)
	b.Handle("/get", b.get)
	b.Handle("/find", b.find)
	b.Handle("/chat", b.chat)
	b.Handle(telebot.OnText, b.onTextMessage)
	b.Handle(telebot.OnAudio, b.onAudio)
	b.Handle(telebot.OnVoice, b.onVoice)
	b.Handle(telebot.OnCallback, b.onCallback)
	b.Use(b.loggingMiddleware)
}

// chat handles the /chat command.
func (b *Bot) chat(c telebot.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user := c.Sender()
	if user == nil {
		return nil
	}

	b.logger.Info("Start command",
		slog.String("username", resolveUsername(user)),
		slog.Int64("user_id", user.ID),
	)

	if !b.userUseCase.Validate(ctx, user.ID) {
		return c.Send(MessageHelp, telebot.ModeHTML)
	}

	receiveMsg, err := b.aiUseCase.Chat(context.Background(), c.Message().Payload)
	if err != nil {
		return err
	}

	return c.Send(receiveMsg, telebot.ModeHTML)
}

// listByUserID handles the /list command.
func (b *Bot) listByUserID(c telebot.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user := c.Sender()
	if user == nil {
		return nil
	}

	b.logger.Info("Start command",
		slog.String("username", resolveUsername(user)),
		slog.Int64("user_id", user.ID),
	)

	if !b.userUseCase.Validate(context.Background(), user.ID) {
		return c.Send(MessageHelp, telebot.ModeHTML)
	}

	list, err := b.meetingUseCase.List(ctx, user.ID)
	if err != nil {
		return err
	}

	msg := FormatMeetingsList(list)

	return c.Send(msg, telebot.ModeHTML)
}

// get handles the /get command.
func (b *Bot) get(c telebot.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user := c.Sender()
	if user == nil {
		return nil
	}

	b.logger.Info("Start command",
		slog.String("username", resolveUsername(user)),
		slog.Int64("user_id", user.ID),
	)

	id, err := strconv.Atoi(c.Message().Payload)
	if err != nil {
		return err
	}

	if !b.userUseCase.Validate(ctx, user.ID) {
		return c.Send(MessageHelp, telebot.ModeHTML)
	}

	meeting, err := b.meetingUseCase.Get(context.Background(), int64(id), user.ID)
	if err != nil {
		return err
	}

	msg := meeting.Transcription

	return c.Send(msg, telebot.ModeHTML)
}

// find handles the /find command.
func (b *Bot) find(c telebot.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user := c.Sender()
	if user == nil {
		return nil
	}

	b.logger.Info("Start command",
		slog.String("username", resolveUsername(user)),
		slog.Int64("user_id", user.ID),
	)

	if !b.userUseCase.Validate(ctx, user.ID) {
		return c.Send(MessageHelp, telebot.ModeHTML)
	}

	prepKeywords := strings.Split(c.Message().Payload, " ")
	var keywords []string
	for _, keyword := range prepKeywords {
		keywords = append(keywords, strings.TrimSpace(keyword))
	}

	req := entity.SearchRequest{
		UserID:   user.ID,
		Keywords: keywords,
	}

	result, err := b.meetingUseCase.SearchByKeywords(context.Background(), req)
	if err != nil {
		return err
	}

	msg := FormatSearchResult(result)
	fmt.Println(msg, err)

	return c.Send(msg, telebot.ModeHTML)
}

// onStart handles the /start command.
func (b *Bot) onStart(c telebot.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user := c.Sender()
	if user == nil {
		return nil
	}

	b.logger.Info("Start command",
		slog.String("username", resolveUsername(user)),
		slog.Int64("user_id", user.ID),
	)

	u, err := b.userUseCase.Start(ctx, &entity.User{
		TelegramID: user.ID,
		UserName:   user.Username,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
	})
	if err != nil {
		return err
	}

	msg := fmt.Sprintf(MessageStart, escapeHTML(u.FirstName))
	return c.Send(msg, telebot.ModeHTML)
}

// onHelp handles the /help command.
func (b *Bot) onHelp(c telebot.Context) error {
	user := c.Sender()
	b.logger.Info("Help command",
		slog.String("username", resolveUsername(user)),
		slog.Int64("user_id", user.ID),
	)
	return c.Send(MessageHelp, telebot.ModeHTML)
}

// onTextMessage handles regular text messages.
func (b *Bot) onTextMessage(c telebot.Context) error {
	text := c.Text()
	if strings.HasPrefix(text, "/") {
		b.logger.Info("Unknown command",
			slog.String("command", text),
			slog.Int64("user_id", c.Sender().ID),
		)
		return c.Reply(MessageUnknownCommand)
	}
	return nil
}

// onCallback handles callback requests from inline buttons.
func (b *Bot) onCallback(c telebot.Context) error {
	_ = c.Respond()
	return nil
}

// onAudio handles incoming audio files.
func (b *Bot) onAudio(c telebot.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msg := c.Message()
	if msg == nil || msg.Audio == nil {
		return nil
	}

	audio := msg.Audio
	user := c.Sender()
	if user == nil {
		b.logger.Warn("Audio from anonymous user")
		return nil
	}

	if !b.userUseCase.Validate(ctx, user.ID) {
		return c.Send(MessageHelp, telebot.ModeHTML)
	}

	if audio.FileSize > MaxAudioSize {
		return c.Reply(fmt.Sprintf(MessageFileTooBig, formatFileSize(audio.FileSize)))
	}

	username := resolveUsername(user)
	b.logAudioReceived(username, user.ID, audio)
	b.sendSafe(c, fmt.Sprintf(MessageAudioReceiving, escapeHTML(audio.FileName)), telebot.ModeHTML)
	b.enqueueAudioJob(c, user, audio, msg.Caption)

	return nil
}

// onVoice handles incoming voice messages.
func (b *Bot) onVoice(c telebot.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msg := c.Message()
	if msg == nil || msg.Voice == nil {
		return nil
	}

	voice := msg.Voice
	user := c.Sender()

	if !b.userUseCase.Validate(ctx, user.ID) {
		return c.Send(MessageHelp, telebot.ModeHTML)
	}

	if voice.FileSize > MaxAudioSize {
		return c.Reply(MessageVoiceTooLong)
	}

	username := resolveUsername(user)
	b.logVoiceReceived(username, user.ID, voice)

	b.sendSafe(c, MessageVoiceReceiving, telebot.ModeHTML)
	b.enqueueVoiceJob(c, user, voice)

	return nil
}

// logAudioReceived logs audio file reception.
func (b *Bot) logAudioReceived(username string, userID int64, audio *telebot.Audio) {
	b.logger.Info("Audio received",
		slog.String("username", username),
		slog.Int64("user_id", userID),
		slog.String("filename", audio.FileName),
		slog.String("mime_type", audio.MIME),
		slog.String("file_id", audio.FileID),
		slog.String("unique_id", audio.UniqueID),
		slog.Int64("size", audio.FileSize),
		slog.Int("duration", audio.Duration),
	)
}

// logVoiceReceived logs voice message reception.
func (b *Bot) logVoiceReceived(username string, userID int64, voice *telebot.Voice) {
	b.logger.Info("Voice received",
		slog.String("username", username),
		slog.Int64("user_id", userID),
		slog.String("file_id", voice.FileID),
		slog.String("unique_id", voice.UniqueID),
		slog.Int64("size", voice.FileSize),
		slog.Int("duration", voice.Duration),
	)
}

// enqueueAudioJob creates and queues an audio processing job.
func (b *Bot) enqueueAudioJob(c telebot.Context, user *telebot.User, audio *telebot.Audio, caption string) {
	jobFile := copyTelebotFile(audio.File)
	jobCtx, jobCancel := context.WithTimeout(context.Background(), JobTimeout)

	job := &processJob{
		ctx:      jobCtx,
		cancel:   jobCancel,
		chatID:   c.Chat().ID,
		userID:   user.ID,
		file:     jobFile,
		fileName: audio.FileName,
		mimeType: audio.MIME,
		fileSize: audio.FileSize,
		duration: audio.Duration,
		caption:  caption,
		fileType: "audio",
	}

	if !b.tryEnqueueJob(job) {
		jobCancel()
		b.logger.Warn("Job queue is full, sending file too busy message")
		b.sendSafe(c, MessageServerBusy, telebot.ModeHTML)
	} else {
		b.logger.Debug("Audio job queued",
			slog.String("filename", audio.FileName),
			slog.String("file_id", audio.FileID),
		)
	}
}

// enqueueVoiceJob creates and queues a voice message processing job.
func (b *Bot) enqueueVoiceJob(c telebot.Context, user *telebot.User, voice *telebot.Voice) {
	jobFile := copyTelebotFile(voice.File)
	fileName := fmt.Sprintf("voice_%d.ogg", time.Now().UnixNano())
	jobCtx, jobCancel := context.WithTimeout(context.Background(), JobTimeout)

	job := &processJob{
		ctx:      jobCtx,
		cancel:   jobCancel,
		chatID:   c.Chat().ID,
		userID:   user.ID,
		file:     jobFile,
		fileName: fileName,
		mimeType: voice.MIME,
		fileSize: voice.FileSize,
		duration: voice.Duration,
		fileType: "voice",
	}

	if !b.tryEnqueueJob(job) {
		jobCancel()
		b.logger.Warn("Job queue is full, sending voice too busy message")
		b.sendSafeToChat(c.Chat().ID, MessageServerBusy, telebot.ModeHTML)
	} else {
		b.logger.Debug("Voice job queued", slog.String("file_id", voice.FileID))
	}
}
