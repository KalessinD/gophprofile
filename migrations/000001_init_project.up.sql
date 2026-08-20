BEGIN;

CREATE SCHEMA IF NOT EXISTS gophprofile;

CREATE TABLE IF NOT EXISTS gophprofile.avatars (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id VARCHAR(64) NOT NULL,
    original_s3_key VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'processing',
    mime_type VARCHAR(32) NOT NULL,
    file_size BIGINT NOT NULL,
    width INTEGER,
    height INTEGER,
    thumbnail_100_s3_key VARCHAR(255),
    thumbnail_300_s3_key VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS avatars_user_id_idx ON gophprofile.avatars USING BTREE(user_id);
CREATE INDEX IF NOT EXISTS avatars_status_idx ON gophprofile.avatars USING BTREE(status);

COMMENT ON TABLE gophprofile.avatars IS 'Таблица метаданных аватарок';

COMMENT ON COLUMN gophprofile.avatars.id IS 'Уникальный идентификатор аватарки';
COMMENT ON COLUMN gophprofile.avatars.user_id IS 'Идентификатор пользователя (из заголовка X-User-ID)';
COMMENT ON COLUMN gophprofile.avatars.original_s3_key IS 'Ключ оригинального файла в S3 хранилище';
COMMENT ON COLUMN gophprofile.avatars.status IS 'Статус обработки (processing, ready, error)';
COMMENT ON COLUMN gophprofile.avatars.mime_type IS 'MIME-тип изображения (image/jpeg, image/png, image/webp)';
COMMENT ON COLUMN gophprofile.avatars.file_size IS 'Размер оригинального файла в байтах';
COMMENT ON COLUMN gophprofile.avatars.width IS 'Ширина оригинального изображения в пикселях';
COMMENT ON COLUMN gophprofile.avatars.height IS 'Высота оригинального изображения в пикселях';
COMMENT ON COLUMN gophprofile.avatars.thumbnail_100_s3_key IS 'Ключ миниатюры 100x100 в S3 хранилище';
COMMENT ON COLUMN gophprofile.avatars.thumbnail_300_s3_key IS 'Ключ миниатюры 300x300 в S3 хранилище';
COMMENT ON COLUMN gophprofile.avatars.created_at IS 'Время создания записи';
COMMENT ON COLUMN gophprofile.avatars.updated_at IS 'Время последнего обновления записи';
COMMENT ON COLUMN gophprofile.avatars.deleted_at IS 'Время мягкого удаления (NULL если не удалена)';

COMMIT;