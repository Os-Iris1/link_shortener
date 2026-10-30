BEGIN;

-- Сохраняем совместимую таблицу, созданную ранней реализацией без миграций.
CREATE TABLE IF NOT EXISTS links (
    code TEXT PRIMARY KEY,
    original_url TEXT NOT NULL
);

COMMIT;
