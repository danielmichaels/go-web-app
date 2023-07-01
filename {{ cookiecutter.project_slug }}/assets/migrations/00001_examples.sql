-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS examples
(
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    created_at DATE DEFAULT (datetime('now', 'utc')),
    updated_at DATE DEFAULT (datetime('now', 'utc')),
    text TEXT NOT NULL
);
CREATE TRIGGER update_created_at
    AFTER UPDATE ON examples
    FOR EACH ROW
    BEGIN
        UPDATE examples
        SET updated_at = datetime('now', 'utc')
        WHERE id = OLD.id;
    END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS examples;
-- +goose StatementEnd
