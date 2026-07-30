-- name: ExampleSelectAll :many
SELECT * FROM examples;

-- name: InsertExample :exec
INSERT INTO examples (text)
VALUES ({% if cookiecutter.database_choice == 'postgres' %}$1{% else %}?{% endif %});

-- name: UpdateExample :exec
UPDATE examples
SET text = {% if cookiecutter.database_choice == 'postgres' %}$1{% else %}?{% endif %}
WHERE id = {% if cookiecutter.database_choice == 'postgres' %}$2{% else %}?{% endif %};
