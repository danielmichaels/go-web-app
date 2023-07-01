-- name: ExampleSelectAll :many
SELECT * FROM examples;

-- name: InsertExample :exec
INSERT INTO examples (text)
VALUES (?);

-- name: UpdateExample :exec
UPDATE examples
SET text = ?
WHERE id = ?;

