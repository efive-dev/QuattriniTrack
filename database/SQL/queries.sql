-- name: InsertTransaction :exec
INSERT INTO transactions(name, cost, date, categories_id)
VALUES (?, ?, ?, ?);

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

-- name: InsertCategory :exec
INSERT INTO categories(name)
VALUES (?);

-- name: GetAllCategories :many
SELECT * FROM categories;

-- name: GetCategoryByID :one
SELECT *
FROM categories
WHERE id = ?;

-- name: DeleteCategory :exec
DELETE
FROM categories
WHERE id = ?;
