-- name: GetCase :one
SELECT *
FROM cases
WHERE id = :caseID;

-- name: ListCases :many
SELECT *
FROM cases;
