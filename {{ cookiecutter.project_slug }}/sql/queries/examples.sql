{% if cookiecutter.database_choice == 'sqlite' %}
-- name: ExampleSelectAll :many
SELECT * FROM examples;

-- name: InsertExample :exec
INSERT INTO examples (text)
VALUES (?);

-- name: UpdateExample :exec
UPDATE examples
SET text = ?
WHERE id = ?;
{% endif %}
{% if cookiecutter.database_choice == 'postgres' %}
-- name: ExampleSelectAll :many
SELECT * FROM examples;

-- name: InsertExample :exec
INSERT INTO examples (text)
VALUES ($1);

-- name: UpdateExample :exec
UPDATE examples
SET text = $1
WHERE id = $2;
{% endif %}

