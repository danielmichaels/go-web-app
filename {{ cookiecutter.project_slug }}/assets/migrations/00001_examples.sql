{% if cookiecutter.database_choice == 'sqlite' %}
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
{% endif %}
{% if cookiecutter.database_choice == 'postgres' %}
-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS examples
(
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    text TEXT NOT NULL
);

CREATE OR REPLACE FUNCTION update_updated_at_column()
    RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_examples_updated_at
    BEFORE UPDATE ON examples
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS examples;
-- +goose StatementEnd
{% endif %}
