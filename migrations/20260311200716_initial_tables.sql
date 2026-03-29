-- +goose Up
-- +goose StatementBegin

CREATE TABLE users (
                       id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                       telegram_id BIGINT UNIQUE NOT NULL,
                       username TEXT,
                       first_name TEXT,
                       last_name TEXT,
                       created_at TIMESTAMPTZ DEFAULT NOW(),
                       updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE meetings (
                          id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                          user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                          title TEXT,
                          transcription TEXT,
                          summary TEXT,
                          audio_file_id TEXT,
                          duration_seconds INTEGER,
                          created_at TIMESTAMPTZ DEFAULT NOW(),
                          updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Индекс для полнотекстового поиска
-- Примечание: язык 'russian' захардкожен. Для мультиязычности нужна доработка.
CREATE INDEX idx_meetings_transcription_gin ON meetings USING GIN (to_tsvector('russian', COALESCE(transcription, '')));

-- Индекс для выборки встреч пользователя по времени
CREATE INDEX idx_meetings_user_created ON meetings(user_id, created_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_meetings_user_created;
DROP INDEX IF EXISTS idx_meetings_transcription_gin;

-- Таблицы удаляем в порядке, обратном созданию (сначала зависимые)
DROP TABLE IF EXISTS meetings;
DROP TABLE IF EXISTS users;

-- +goose StatementEnd