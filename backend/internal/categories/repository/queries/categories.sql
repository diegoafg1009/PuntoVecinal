-- name: ListCategories :many
SELECT slug, label
FROM categories
ORDER BY position;
