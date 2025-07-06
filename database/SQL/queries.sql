-- name: InsertTransaction :exec
INSERT INTO transactions(name, cost, date)
VALUES (?, ?, ?);

-- name: GetAllTransactions :many
SELECT * FROM transactions;
