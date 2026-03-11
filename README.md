# 🧠 MeetMind - AI Meeting Assistant

<p align="center">
  <img src="docs/logo.png" alt="MeetMind Logo" width="200"/>
</p>

<p align="center">
  <b>🇬🇧 English</b> | <b>🇷🇺 Русский</b>
</p>

---

<!-- ENGLISH VERSION -->
<div lang="en">

# 🇬🇧 MeetMind

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
[![Telegram Bot](https://img.shields.io/badge/Telegram-Bot-26A5E4?style=for-the-badge&logo=telegram)](https://core.telegram.org/bots)
[![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)](LICENSE)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16+-316192?style=for-the-badge&logo=postgresql)](https://www.postgresql.org/)

## 📋 About

**MeetMind** is a Telegram bot that automatically transcribes meeting audio, creates concise summaries, and enables searching through conversation content. No more lost agreements!

### 🎯 Problem
- We forget 80% of what we hear within 24 hours
- Meeting notes get lost in messengers
- New employees struggle to catch up with context

### ✅ Solution
- Automatic audio transcription (SaluteSpeech)
- Concise meeting summaries (GigaChat)
- Full-text search across all conversations
- Convenient access via Telegram bot

## ✨ Features

| Command | Description |
|---------|-------------|
| `/start` | Register in the system |
| `/list` | List all meetings |
| `/get <id>` | Full meeting transcription |
| `/find <text>` | Search by keywords |
| `/chat <question>` | Ask questions about meetings |
| `/stats` | Usage statistics |
| `/help` | Command help |

### 🎤 Supported Formats
- Voice messages (OGG)
- Audio files (MP3, WAV, M4A)
- Up to 50 MB per file

## 🏗 Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  Telegram   │────▶│   MeetMind   │────▶│  PostgreSQL │
│    Bot      │◀────│    Worker    │◀────│             │
└─────────────┘     └──────┬───────┘     └─────────────┘
                           │
                    ┌──────▼───────┐
                    │    Queue     │
                    │  (Channels)  │
                    └──────┬───────┘
                           │
              ┌────────────┴────────────┐
              │                         │
       ┌──────▼──────┐           ┌──────▼──────┐
       │ SaluteSpeech │           │  GigaChat   │
       │     API      │           │     API     │
       └──────────────┘           └─────────────┘
```

## 🛠 Tech Stack

- **Backend**: Go 1.21+
- **Bot**: [telebot v3](https://gopkg.in/telebot.v3)
- **Database**: PostgreSQL 16+ (full-text search)
- **Queue**: In-memory channels (scalable)
- **AI/ML**:
    - [SaluteSpeech](https://developers.sber.ru/docs/en/salutespeech/overview) — speech recognition
    - [GigaChat](https://developers.sber.ru/gigachat) — summaries and Q&A
- **Infrastructure**: Docker, Docker Compose

## 📦 Installation

### Prerequisites
- Go 1.21+
- PostgreSQL 16+
- Docker (optional)
- SaluteSpeech and GigaChat API keys

### Quick Start

1. **Clone repository**
```bash
git clone https://github.com/yourusername/meetmind.git
cd meetmind
```

2. **Configure environment**
```bash
cp .env.example .env
# Edit .env, add your API keys
```

3. **Run PostgreSQL via Docker**
```bash
docker-compose up -d postgres
```

4. **Apply migrations**
```bash
make migrate-up
```

5. **Build and run**
```bash
make build
./bin/meetmind
```

### 🐳 Docker Setup
```bash
# Build image
docker build -t meetmind .

# Run all services
docker-compose up -d
```

## 📁 Project Structure

```
meetmind/
├── cmd/
│   └── bot/                 # Entry point
│       └── main.go
├── internal/
│   ├── bot/                 # Telegram bot
│   │   ├── handler/         # Command handlers
│   │   ├── middleware/      # Middleware (rate limit, logs)
│   │   └── keyboard/        # Keyboards
│   ├── services/            # External services
│   │   ├── salutespeech/    # SaluteSpeech client
│   │   ├── gigachat/        # GigaChat client
│   │   └── downloader/      # File downloader
│   ├── repository/          # Database layer
│   │   ├── postgres/        # PostgreSQL implementations
│   │   └── models/          # Data models
│   ├── queue/               # Task queue
│   └── config/              # Configuration
├── pkg/
│   └── utils/               # Utilities
├── migrations/              # SQL migrations
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── README.md
```

## 🔧 Configuration

Environment variables (`.env`):

```env
# Telegram
TELEGRAM_BOT_TOKEN=your_bot_token

# SaluteSpeech
SALUTE_SPEECH_CLIENT_ID=your_client_id
SALUTE_SPEECH_CLIENT_SECRET=your_client_secret

# GigaChat
GIGACHAT_CLIENT_ID=your_client_id
GIGACHAT_CLIENT_SECRET=your_client_secret
GIGACHAT_AUTH_KEY=your_auth_key

# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/meetmind

# App
WORKER_COUNT=5
MAX_FILE_SIZE=50
LOG_LEVEL=info
```

## 🚀 Usage Examples

```
User:  /find server deployment
Bot:   Found 3 meetings:
       
       1. 2024-05-12 - Infrastructure discussion
          ...discussed purchasing new **server** for production...
       
       2. 2024-05-10 - Development roadmap
          ...migrate databases to dedicated **server**...

User:  /get 1
Bot:   📝 Meeting from 2024-05-12
       
       Transcription:
       [full meeting text...]
```

## 🧪 Testing

```bash
# Run all tests
make test

# With coverage report
make test-coverage

# Linter
make lint
```

## 📊 Monitoring

- **Metrics**: Prometheus endpoint at `/metrics`
- **Logs**: Structured JSON logging
- **Tracing**: OpenTelemetry support (optional)

## 🤝 Contributing

1. Fork the repository
2. Create branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

### Code Requirements
- Pass `make lint`
- Write tests for new functionality
- Update documentation

## 📈 Roadmap

- [x] Basic audio transcription
- [x] GigaChat integration
- [x] Full-text search
- [ ] Group chat support
- [ ] Web interface for meetings
- [ ] PDF/Markdown export
- [ ] Multi-language support
- [ ] Calendar integration

## 📄 License

MIT License. See [LICENSE](LICENSE) for details.

## 📞 Contact

- Telegram: [@meetmind_bot](https://t.me/meetmind_bot) (demo bot)
- GitHub Issues: [create issue](https://github.com/yourusername/meetmind/issues)
- Email: support@meetmind.ai

</div>

---

<!-- RUSSIAN VERSION -->
<div lang="ru">

# 🇷🇺 MeetMind

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
[![Telegram Bot](https://img.shields.io/badge/Telegram-Бот-26A5E4?style=for-the-badge&logo=telegram)](https://core.telegram.org/bots)
[![License](https://img.shields.io/badge/Лицензия-MIT-green?style=for-the-badge)](LICENSE)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16+-316192?style=for-the-badge&logo=postgresql)](https://www.postgresql.org/)

## 📋 О проекте

**MeetMind** — это Telegram-бот, который автоматически транскрибирует аудиозаписи встреч, создаёт краткие выжимки и позволяет искать по содержимому разговоров. Больше никаких потерянных договорённостей!

### 🎯 Проблема
- Мы забываем 80% услышанного через 24 часа
- Записи встреч теряются в мессенджерах
- Новым сотрудникам сложно вникнуть в контекст

### ✅ Решение
- Автоматическая транскрипция аудио (SaluteSpeech)
- Краткие саммари встреч (GigaChat)
- Полнотекстовый поиск по всем разговорам
- Доступ через удобный Telegram-бот

## ✨ Возможности

| Команда | Описание |
|---------|----------|
| `/start` | Регистрация в системе |
| `/list` | Список всех встреч |
| `/get <id>` | Полная транскрипция встречи |
| `/find <текст>` | Поиск по ключевым словам |
| `/chat <вопрос>` | Задать вопрос по встречам |
| `/stats` | Статистика использования |
| `/help` | Справка по командам |

### 🎤 Поддержка форматов
- Голосовые сообщения (OGG)
- Аудиофайлы (MP3, WAV, M4A)
- До 50 МБ на файл

## 🏗 Архитектура

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  Telegram   │────▶│   MeetMind   │────▶│  PostgreSQL │
│    Bot      │◀────│    Worker    │◀────│             │
└─────────────┘     └──────┬───────┘     └─────────────┘
                           │
                    ┌──────▼───────┐
                    │    Queue     │
                    │  (Channels)  │
                    └──────┬───────┘
                           │
              ┌────────────┴────────────┐
              │                         │
       ┌──────▼──────┐           ┌──────▼──────┐
       │ SaluteSpeech │           │  GigaChat   │
       │     API      │           │     API     │
       └──────────────┘           └─────────────┘
```

## 🛠 Технологический стек

- **Backend**: Go 1.21+
- **Бот**: [telebot v3](https://gopkg.in/telebot.v3)
- **База данных**: PostgreSQL 16+ (полнотекстовый поиск)
- **Очереди**: In-memory channels (с возможностью масштабирования)
- **AI/ML**:
    - [SaluteSpeech](https://developers.sber.ru/docs/ru/salutespeech/overview) — распознавание речи
    - [GigaChat](https://developers.sber.ru/gigachat) — генерация саммари и ответы на вопросы
- **Инфраструктура**: Docker, Docker Compose

## 📦 Установка

### Предварительные требования
- Go 1.21+
- PostgreSQL 16+
- Docker (опционально)
- API ключи SaluteSpeech и GigaChat

### Быстрый старт

1. **Клонируйте репозиторий**
```bash
git clone https://github.com/yourusername/meetmind.git
cd meetmind
```

2. **Настройте переменные окружения**
```bash
cp .env.example .env
# Отредактируйте .env, добавьте свои API ключи
```

3. **Запустите PostgreSQL через Docker**
```bash
docker-compose up -d postgres
```

4. **Примените миграции**
```bash
make migrate-up
```

5. **Соберите и запустите**
```bash
make build
./bin/meetmind
```

### 🐳 Запуск через Docker
```bash
# Сборка образа
docker build -t meetmind .

# Запуск всех сервисов
docker-compose up -d
```

## 📁 Структура проекта

```
meetmind/
├── cmd/
│   └── bot/                 # Точка входа
│       └── main.go
├── internal/
│   ├── bot/                 # Telegram бот
│   │   ├── handler/         # Обработчики команд
│   │   ├── middleware/      # Middleware (rate limit, логи)
│   │   └── keyboard/        # Клавиатуры
│   ├── services/            # Внешние сервисы
│   │   ├── salutespeech/    # Клиент для SaluteSpeech
│   │   ├── gigachat/        # Клиент для GigaChat
│   │   └── downloader/      # Загрузка файлов
│   ├── repository/          # Работа с БД
│   │   ├── postgres/        # PostgreSQL реализации
│   │   └── models/          # Модели данных
│   ├── queue/               # Очередь задач
│   └── config/              # Конфигурация
├── pkg/
│   └── utils/               # Утилиты
├── migrations/              # SQL миграции
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── README.md
```

## 🔧 Конфигурация

Переменные окружения (`.env`):

```env
# Telegram
TELEGRAM_BOT_TOKEN=your_bot_token

# SaluteSpeech
SALUTE_SPEECH_CLIENT_ID=your_client_id
SALUTE_SPEECH_CLIENT_SECRET=your_client_secret

# GigaChat
GIGACHAT_CLIENT_ID=your_client_id
GIGACHAT_CLIENT_SECRET=your_client_secret
GIGACHAT_AUTH_KEY=your_auth_key

# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/meetmind

# App
WORKER_COUNT=5
MAX_FILE_SIZE=50
LOG_LEVEL=info
```

## 🚀 Примеры использования

```
User:  /find сервер
Bot:   Найдено 3 встречи:
       
       1. 12.05.2024 - Обсуждение инфраструктуры
          ...обсудили покупку нового **сервера** для продакшена...
       
       2. 10.05.2024 - План развития
          ...перенести базы данных на выделенный **сервер**...

User:  /get 1
Bot:   📝 Встреча от 12.05.2024
       
       Транскрипция:
       [полный текст встречи...]
```

## 🧪 Тестирование

```bash
# Запуск всех тестов
make test

# С coverage отчётом
make test-coverage

# Линтер
make lint
```

## 📊 Мониторинг

- **Метрики**: Prometheus endpoint на `/metrics`
- **Логи**: Структурированные логи в JSON
- **Трейсинг**: Поддержка OpenTelemetry (опционально)

## 🤝 Вклад в проект

1. Форкните репозиторий
2. Создайте ветку (`git checkout -b feature/amazing-feature`)
3. Зафиксируйте изменения (`git commit -m 'Add amazing feature'`)
4. Запушьте ветку (`git push origin feature/amazing-feature`)
5. Откройте Pull Request

### Требования к коду
- Проходите `make lint`
- Пишите тесты на новую функциональность
- Обновляйте документацию

## 📈 План развития

- [x] Базовая транскрипция аудио
- [x] Интеграция с GigaChat
- [x] Полнотекстовый поиск
- [ ] Поддержка групповых чатов
- [ ] Веб-интерфейс для просмотра встреч
- [ ] Экспорт в PDF/Markdown
- [ ] Мультиязычная поддержка
- [ ] Интеграция с календарём

## 📄 Лицензия

MIT License. Смотрите [LICENSE](LICENSE) для деталей.

## 📞 Контакты

- Telegram: [@meetmind_bot](https://t.me/meetmind_bot) (демо-бот)
- GitHub Issues: [создать issue](https://github.com/yourusername/meetmind/issues)
- Email: support@meetmind.ai

</div>

---

<p align="center">
  <b>🇬🇧 English</b> | <b>🇷🇺 Русский</b>
</p>

<p align="center">
  Made with ❤️ for effective meetings
</p>
