-- name: InsertReport :exec
INSERT INTO reports (id, author_id, category_slug, title, details, reference, location, author_name, embedding)
VALUES (
  @id,
  @author_id,
  @category_slug,
  @title,
  sqlc.narg('details'),
  sqlc.narg('reference'),
  ST_SetSRID(ST_MakePoint(@longitude::float8, @latitude::float8), 4326)::geography,
  sqlc.narg('author_name'),
  sqlc.narg('embedding')
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
-- Never selects author_id: the detail is public to any session. The caller's
-- relation to the report comes as two flags instead.
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
  r.created_at,
  (1 + (SELECT count(*) FROM report_supports s WHERE s.report_id = r.id AND s.user_id <> r.author_id))::int AS affected_count,
  (SELECT count(*) FROM report_contributions k WHERE k.report_id = r.id)::int AS contribution_count,
  (r.author_id = @caller)::bool AS is_author,
  EXISTS (
    SELECT 1 FROM report_supports s WHERE s.report_id = r.id AND s.user_id = @caller
  )::bool AS supported
FROM reports r
JOIN categories c ON c.slug = r.category_slug
WHERE r.id = @id;

-- name: ListReportPhotos :many
SELECT position, storage_key
FROM report_photos
WHERE report_id = @report_id
ORDER BY position;

-- name: ListReports :many
-- One page of the feed, newest first, continuing after the (created_at, id)
-- key when one is given. The distance is in whole meters and only computed
-- when a location is given: -1 means no location (sqlc cannot type a nullable
-- expression). affected_count is the author plus the supporters. Never selects
-- author_id nor author_name.
SELECT
  r.id,
  r.title,
  r.category_slug,
  c.label AS category_label,
  r.status,
  r.reference,
  r.created_at,
  p.storage_key AS cover_key,
  CASE
    WHEN sqlc.narg('lat')::float8 IS NULL THEN -1
    ELSE round(ST_Distance(
      r.location,
      ST_SetSRID(ST_MakePoint(sqlc.narg('lng')::float8, sqlc.narg('lat')::float8), 4326)::geography
    ))::int
  END AS distance_meters,
  (1 + (SELECT count(*) FROM report_supports s WHERE s.report_id = r.id AND s.user_id <> r.author_id))::int AS affected_count
FROM reports r
JOIN categories c ON c.slug = r.category_slug
JOIN report_photos p ON p.report_id = r.id AND p.position = 1
WHERE (
    sqlc.narg('after_created_at')::timestamptz IS NULL
    OR (r.created_at, r.id) < (sqlc.narg('after_created_at')::timestamptz, sqlc.narg('after_id')::uuid)
  )
  AND (cardinality(@categories::text[]) = 0 OR r.category_slug = ANY(@categories::text[]))
  AND (sqlc.narg('status')::report_status IS NULL OR r.status = sqlc.narg('status')::report_status)
ORDER BY r.created_at DESC, r.id DESC
LIMIT @row_limit;

-- name: ListReportPins :many
-- The newest reports as map points, with the same category and status filters
-- as ListReports (an empty @categories array matches every category). Never
-- selects author_id nor author_name, and needs no photo.
SELECT
  r.id,
  r.title,
  r.category_slug,
  c.label AS category_label,
  r.status,
  ST_Y(r.location::geometry)::float8 AS latitude,
  ST_X(r.location::geometry)::float8 AS longitude
FROM reports r
JOIN categories c ON c.slug = r.category_slug
WHERE (cardinality(@categories::text[]) = 0 OR r.category_slug = ANY(@categories::text[]))
  AND (sqlc.narg('status')::report_status IS NULL OR r.status = sqlc.narg('status')::report_status)
ORDER BY r.created_at DESC, r.id DESC
LIMIT @row_limit;

-- name: UnknownCategories :many
-- The slugs in @slugs that are not one of the fixed categories.
SELECT s::text AS slug
FROM unnest(@slugs::text[]) AS s
WHERE NOT EXISTS (SELECT 1 FROM categories WHERE slug = s);

-- name: SearchReportsRecent :many
-- One page of the search, newest first, continuing after the (created_at, id)
-- key when one is given. A report matches when every word in @words (already
-- split and escaped for ILIKE; an empty array matches everything) appears in
-- its title or details, ignoring case and accents. The distance is as in
-- ListReports: -1 means no location. Never selects author_id nor author_name.
SELECT
  r.id,
  r.title,
  r.category_slug,
  c.label AS category_label,
  r.status,
  r.reference,
  r.created_at,
  p.storage_key AS cover_key,
  CASE
    WHEN sqlc.narg('lat')::float8 IS NULL THEN -1
    ELSE round(ST_Distance(
      r.location,
      ST_SetSRID(ST_MakePoint(sqlc.narg('lng')::float8, sqlc.narg('lat')::float8), 4326)::geography
    ))::int
  END AS distance_meters,
  (1 + (SELECT count(*) FROM report_supports s WHERE s.report_id = r.id AND s.user_id <> r.author_id))::int AS affected_count
FROM reports r
JOIN categories c ON c.slug = r.category_slug
JOIN report_photos p ON p.report_id = r.id AND p.position = 1
WHERE NOT EXISTS (
    SELECT 1 FROM unnest(@words::text[]) AS w
    WHERE extensions.unaccent(r.title || ' ' || coalesce(r.details, ''))
          NOT ILIKE '%' || extensions.unaccent(w) || '%'
  )
  AND (
    sqlc.narg('after_created_at')::timestamptz IS NULL
    OR (r.created_at, r.id) < (sqlc.narg('after_created_at')::timestamptz, sqlc.narg('after_id')::uuid)
  )
ORDER BY r.created_at DESC, r.id DESC
LIMIT @row_limit;

-- name: SearchReportsNearest :many
-- One page of the search, nearest first to (@lat, @lng), ties going to the
-- newest. The keyset is (distance, created_at, id): a report comes after the
-- cursor when it is farther, or at the same exact distance and older. The
-- exact distance is returned for the cursor, and distance_meters is the same
-- distance rounded to whole meters.
-- Never selects author_id nor author_name.
SELECT found.*, round(found.distance)::int AS distance_meters
FROM (
  SELECT
    r.id,
    r.title,
    r.category_slug,
    c.label AS category_label,
    r.status,
    r.reference,
    r.created_at,
    p.storage_key AS cover_key,
    ST_Distance(
      r.location,
      ST_SetSRID(ST_MakePoint(@lng::float8, @lat::float8), 4326)::geography
    )::float8 AS distance,
    (1 + (SELECT count(*) FROM report_supports s WHERE s.report_id = r.id AND s.user_id <> r.author_id))::int AS affected_count
  FROM reports r
  JOIN categories c ON c.slug = r.category_slug
  JOIN report_photos p ON p.report_id = r.id AND p.position = 1
  WHERE NOT EXISTS (
    SELECT 1 FROM unnest(@words::text[]) AS w
    WHERE extensions.unaccent(r.title || ' ' || coalesce(r.details, ''))
          NOT ILIKE '%' || extensions.unaccent(w) || '%'
  )
) AS found
WHERE sqlc.narg('after_distance')::float8 IS NULL
   OR found.distance > sqlc.narg('after_distance')::float8
   OR (
     found.distance = sqlc.narg('after_distance')::float8
     AND (found.created_at, found.id) < (sqlc.narg('after_created_at')::timestamptz, sqlc.narg('after_id')::uuid)
   )
ORDER BY found.distance ASC, found.created_at DESC, found.id DESC
LIMIT @row_limit;

-- name: GetReportAuthor :one
-- Only for decisions inside the service (404, 409, aporte support); the value
-- never reaches a response.
SELECT author_id FROM reports WHERE id = @id;

-- name: ListReportsMissingEmbedding :many
-- Reports to embed in the backfill job, oldest first.
SELECT id, title, details
FROM reports
WHERE embedding IS NULL
ORDER BY created_at, id
LIMIT @row_limit;

-- name: SetReportEmbedding :exec
UPDATE reports SET embedding = @embedding::vector WHERE id = @id AND embedding IS NULL;

-- name: FindSimilarReports :many
-- Open reports within 300 m of the point whose embedding is within
-- @max_distance (cosine distance) of @embedding, best match first. Never
-- selects author_id nor author_name nor details.
SELECT
  r.id,
  r.title,
  r.category_slug,
  c.label AS category_label,
  r.status,
  r.created_at,
  p.storage_key AS cover_key,
  round(ST_Distance(
    r.location,
    ST_SetSRID(ST_MakePoint(@lng::float8, @lat::float8), 4326)::geography
  ))::int AS distance_meters,
  (1 + (SELECT count(*) FROM report_supports s WHERE s.report_id = r.id AND s.user_id <> r.author_id))::int AS affected_count
FROM reports r
JOIN categories c ON c.slug = r.category_slug
JOIN report_photos p ON p.report_id = r.id AND p.position = 1
WHERE ST_DWithin(
    r.location,
    ST_SetSRID(ST_MakePoint(@lng::float8, @lat::float8), 4326)::geography,
    300
  )
  AND r.embedding IS NOT NULL
  AND r.status NOT IN ('atendido', 'cerrado')
  AND (r.embedding <=> @embedding::vector)::float8 <= @max_distance::float8
ORDER BY (r.embedding <=> @embedding::vector) ASC, 8 ASC, r.id ASC
LIMIT 5;

-- name: InsertSupport :exec
INSERT INTO report_supports (report_id, user_id)
VALUES (@report_id, @user_id)
ON CONFLICT DO NOTHING;

-- name: DeleteSupport :exec
DELETE FROM report_supports WHERE report_id = @report_id AND user_id = @user_id;

-- name: CountAffected :one
-- The author plus every other neighbor who supports the report.
SELECT (1 + count(*))::int AS total
FROM report_supports s
JOIN reports r ON r.id = s.report_id
WHERE s.report_id = @report_id AND s.user_id <> r.author_id;

-- name: IsSupporter :one
SELECT EXISTS (
  SELECT 1 FROM report_supports WHERE report_id = @report_id AND user_id = @user_id
);

-- name: InsertContribution :exec
INSERT INTO report_contributions (id, report_id, author_id, author_name, text, photo_key)
VALUES (
  @id,
  @report_id,
  @author_id,
  sqlc.narg('author_name'),
  @text,
  sqlc.narg('photo_key')
);

-- name: CountRecentContributions :one
-- Aportes the author added since @since, and when the oldest of them was
-- created (null when there are none).
SELECT count(*)::int AS total, min(created_at)::timestamptz AS oldest
FROM report_contributions
WHERE author_id = @author_id
  AND created_at > @since;

-- name: GetContribution :one
-- Never selects author_id.
SELECT id, text, author_name, photo_key, created_at
FROM report_contributions
WHERE id = @id;

-- name: ListContributions :many
-- One page of the aportes of a report, oldest first, continuing after the
-- (created_at, id) key when one is given. Never selects author_id.
SELECT id, text, author_name, photo_key, created_at
FROM report_contributions
WHERE report_id = @report_id
  AND (
    sqlc.narg('after_created_at')::timestamptz IS NULL
    OR (created_at, id) > (sqlc.narg('after_created_at')::timestamptz, sqlc.narg('after_id')::uuid)
  )
ORDER BY created_at ASC, id ASC
LIMIT @row_limit;
