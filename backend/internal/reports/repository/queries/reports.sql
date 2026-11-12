-- name: InsertReport :exec
INSERT INTO reports (id, author_id, category_slug, title, details, reference, location, author_name)
VALUES (
  @id,
  @author_id,
  @category_slug,
  @title,
  sqlc.narg('details'),
  sqlc.narg('reference'),
  ST_SetSRID(ST_MakePoint(@longitude::float8, @latitude::float8), 4326)::geography,
  sqlc.narg('author_name')
);

-- name: InsertReportPhoto :exec
INSERT INTO report_photos (report_id, position, storage_key)
VALUES (@report_id, @position, @storage_key);

-- name: CountRecentReports :one
-- Reports the author published since @since, and when the oldest of them was
-- created (null when there are none).
SELECT count(*)::int AS total, min(created_at)::timestamptz AS oldest
FROM reports
WHERE author_id = @author_id
  AND created_at > @since;

-- name: CategoryExists :one
SELECT EXISTS (SELECT 1 FROM categories WHERE slug = @slug);

-- name: GetReport :one
-- Never selects author_id: the detail is public to any session.
SELECT
  r.id,
  r.title,
  r.details,
  r.category_slug,
  c.label AS category_label,
  r.status,
  ST_Y(r.location::geometry)::float8 AS latitude,
  ST_X(r.location::geometry)::float8 AS longitude,
  r.reference,
  r.author_name,
  r.created_at
FROM reports r
JOIN categories c ON c.slug = r.category_slug
WHERE r.id = @id;

-- name: ListReportPhotos :many
SELECT position, storage_key
FROM report_photos
WHERE report_id = @report_id
ORDER BY position;
