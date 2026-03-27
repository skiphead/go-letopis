package bot

// Message constants for bot responses.
const (
	// MessageStart is sent when user starts the bot.
	MessageStart = "Привет, %s! 👋\n\nЯ бот для обработки аудио и голосовых сообщений. Отправь мне аудиофайл или голосовое сообщение, и я его обработаю."

	// MessageHelp provides information about available commands and bot capabilities.
	MessageHelp = "📌 *Доступные команды:*\n" +
		"/start - начать работу\n" +
		"/help - показать эту справку\n\n" +
		"🎵 *Что я умею:*\n" +
		"- Принимаю аудиофайлы до 20 MB\n" +
		"- Принимаю голосовые сообщения\n" +
		"- Преобразую речь в текст\n" +
		"- Сохраняю обработанные файлы"

	// MessageUnknownCommand is sent when user sends an unrecognized command.
	MessageUnknownCommand = "❓ Неизвестная команда. Используй /help для списка команд."

	// MessageFileTooBig is sent when uploaded file exceeds size limit.
	MessageFileTooBig = "❌ Файл слишком большой (%s). Максимальный размер: 20 MB"

	// MessageVoiceTooLong is sent when voice message exceeds duration limit.
	MessageVoiceTooLong = "❌ Голосовое сообщение слишком длинное. Максимальная длина: 10 минут"

	// MessageServerBusy is sent when the server is overloaded.
	MessageServerBusy = "⚠️ Сервер временно перегружен. Пожалуйста, попробуй позже."

	// MessageInternalError is sent when an unexpected error occurs.
	MessageInternalError = "🔧 Произошла внутренняя ошибка. Пожалуйста, попробуй позже."

	// MessageAudioReceiving confirms audio file reception and processing start.
	MessageAudioReceiving = "🎵 Получил аудиофайл *%s*! Начинаю обработку..."

	// MessageVoiceReceiving confirms voice message reception and processing start.
	MessageVoiceReceiving = "🎙️ Получено голосовое сообщение! Начинаю обработку..."

	// MessageAudioDownloadFailed is sent when audio download fails.
	MessageAudioDownloadFailed = "❌ Не удалось скачать аудиофайл *%s*. Проверь соединение и попробуй снова."

	// MessageVoiceDownloadFailed is sent when voice message download fails.
	MessageVoiceDownloadFailed = "❌ Не удалось скачать голосовое сообщение. Проверь соединение и попробуй снова."

	// MessageAudioSaveFailed is sent when saving processed audio fails.
	MessageAudioSaveFailed = "❌ Не удалось сохранить обработанный аудиофайл."

	// MessageVoiceSaveFailed is sent when saving processed voice message fails.
	MessageVoiceSaveFailed = "❌ Не удалось сохранить обработанное голосовое сообщение."

	// MessageAudioSaved confirms successful audio processing.
	MessageAudioSaved = "✅ Аудиофайл *%s* успешно обработан!\n\n📊 Длительность: %d сек\n💾 Размер: %s"

	// MessageVoiceSaved confirms successful voice message processing.
	MessageVoiceSaved = "✅ Голосовое сообщение успешно обработано!\n\n📊 Длительность: %d сек"
	// MessageInvalidMeetingID is sent when meeting ID is invalid.
	MessageInvalidMeetingID = "❌ Неверный ID встречи. Пожалуйста, укажите корректный числовой ID."

	// MessageMeetingNotFound is sent when meeting is not found.
	MessageMeetingNotFound = "❌ Встреча с указанным ID не найдена."

	// MessageNoTranscription is sent when meeting has no transcription.
	MessageNoTranscription = "📝 Транскрипция для этой встречи отсутствует."

	// MessageWelcomeBack is sent when user already exists.
	MessageWelcomeBack = "С возвращением, %s! 👋\n\nЯ готов обработать ваши аудиофайлы и голосовые сообщения."

	// MessageSearchQueryRequired is sent when search query is empty.
	MessageSearchQueryRequired = "🔍 Пожалуйста, укажите ключевые слова для поиска.\n\nПример: `/find бюджет сайт`"
)
