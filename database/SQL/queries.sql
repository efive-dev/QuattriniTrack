-- name: InsertTransaction :exec
INSERT INTO transactions(name, cost, date)
VALUES (?, ?, ?);

-- name: GetAllTransactions :many
SELECT * FROM transactions;

-- name: GetTransactionByID :one
SELECT *
FROM transactions
WHERE id = ?;

-- name: GetTransactionByName :many
SELECT *
FROM transactions
WHERE name = ?;

-- name: DeleteTransaction :exec
DELETE
FROM transactions
WHERE id = ?;
