

-- name: ExampleSelectAll :many
SELECT * FROM examples;

-- name: InsertExample :exec
INSERT INTO examples (text)
VALUES ($1);

-- name: UpdateExample :exec
UPDATE examples
SET text = $1
WHERE id = $2;


