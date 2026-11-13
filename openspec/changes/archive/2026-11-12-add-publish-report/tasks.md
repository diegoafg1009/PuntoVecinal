# Tasks

## 1. Contract and database

- [x] 1.1 [backend] Add the `reports` module to the contract: `openapi/modules/reports.yaml` with `createReport` (multipart, 201/400/401/413/429), `getReport` (200/401/404, used by the page in HU-02), `openapi/config/reports.yaml`, the `reports` tag and the path references in `openapi/openapi.yaml`; verify with `make bundle` (from `backend/`) that the bundle builds and that no schema exposes the author's user id.
- [x] 1.2 [backend] Run `make generate-api` and verify `internal/reports/api/generated.go` exists, the other modules' generated files are unchanged, and a second run leaves `git status` unchanged.
- [x] 1.3 [db] Add the migration in `supabase/migrations/` (PostGIS, `report_status`, `reports`, `report_photos`, indexes, RLS enabled without policies); verify with `make db-reset` and `make db-lint` (from `backend/`).
- [x] 1.4 [backend] Add `internal/reports/repository/queries/reports.sql` (insert report with the point built in SQL, insert photo, count recent reports with the oldest timestamp, get by id with latitude/longitude and photo keys), the `sqlc.yaml` entry, and run `go tool sqlc generate`; verify `go build ./...` passes.

## 2. Object storage

- [x] 2.1 [backend] Rename `Spaces` to `S3` in `internal/platform/config` (`S3_ENDPOINT`, `S3_PUBLICENDPOINT`, `S3_REGION`, `S3_BUCKET`, `S3_ACCESSKEY`, `S3_SECRETKEY`, `S3_FORCEPATHSTYLE`), validate the required ones at startup, and update `.env.example` with the RustFS values as local default and production examples as comments; verify with config tests (defaults, missing required values, public endpoint falling back to the endpoint).
- [x] 2.2 [backend] Add `backend/compose.yaml` with RustFS (API 9000, console 9001, named volume) plus a one-shot service that creates the bucket `punto-vecinal`, and the `make storage-start` / `make storage-stop` targets (also in `make help`); verify `make storage-start` brings it up, the bucket exists in the console and `make storage-stop` stops it.
- [x] 2.3 [backend] Add `internal/platform/storage` (AWS SDK for Go v2): `Put`, `Delete` and signed `URL` using the public endpoint, path-style switch; verify with unit tests against a fake S3 HTTP server (put, delete, URL host and expiry) and, with RustFS running, a manual round trip (put, fetch through the signed URL, delete).
- [x] 2.4 [backend] Add `internal/platform/imaging`: decode JPEG/PNG with pixel-count bounds, re-encode as JPEG; verify with unit tests that EXIF GPS is gone after processing and that non-images and oversized images are rejected.

## 3. Backend reports module

- [x] 3.1 [backend] Create `internal/reports` (`domain.go`, `dto.go`, `service.go` with `Repository` and `PhotoStore` interfaces defined where consumed, `repository/repository.go` `PgRepository`) with `Create`: validation, author name trimming, rate limit (5 per 10 minutes, seconds to wait), photo cleaning, upload then transactional insert, cleanup of uploaded files on failure; verify with service tests using fakes for every scenario of `report-publishing`.
- [x] 3.2 [backend] Add `Get` to the service (signed photo links with a one-hour expiry, `ErrNotFound`); verify with service tests for the scenarios of `report-detail` (existing, unknown id, ordered photos, no author id in the DTO).
- [x] 3.3 [backend] Add `handler.go` for the two operations (multipart parsing with per-photo and total size limits, error mapping to 400/404/413/429 with `Retry-After`) and `Register`, mount it in `internal/platform/httpserver/routes.go`, and make `platform/auth` skip request body validation; verify with handler tests through the module router (201, every 400 variant, 401 without a token on each operation, 404 on `getReport`, 413, 429 with `Retry-After`, no author id in any response) and that existing tests still pass with `make test`.
- [x] 3.4 [backend] Update `backend/AGENTS.md` (modules list, multipart note, storage and imaging packages, S3 settings, RustFS commands) and `README.md` (Archivos: S3-compatible, RustFS locally); verify the text matches the code and the Makefile.

## 4. Frontend

- [x] 4.1 [frontend] Run `bun run generate:api` (after `make bundle`) and verify `app/api/` has `createReport` and `getReport` with their types and Vue Query helpers; check how `photos` is serialized into `FormData` and add a `bodySerializer` if it is not a repeated field; verify with `bun run typecheck`.
- [x] 4.2 [frontend] Extend `reportDraft` with `authorName` (stored with the text, trimmed, max 40); verify with `bun run typecheck` and by reloading the form that the name is restored.
- [x] 4.3 [frontend] Build `ReportStepReview.vue`: summary with «Editar» per block, rules checkbox with the rules modal, author switch and name field for anonymous sessions, «Publicar» with Spanish validation messages; render step 3 in `pages/report.vue`; verify in `bun run dev` the scenarios of «Step 3 reviews the report», «Publishing requires accepting the community rules» and «Neighbors without an account choose how they appear».
- [x] 4.4 [frontend] Wire the publish mutation: disabled while pending, on success `draft.reset()`, navigate to `/reports/{id}` and show the toast «Reporte publicado»; on 429 and other errors show the Spanish messages and keep the draft; verify the scenarios of «Publishing outcome» in the browser (success, API down, 429).
- [x] 4.5 [frontend] Add the placeholder `pages/reports/[id].vue` with `ComingSoon`; verify that after publishing the browser opens `/reports/{id}` showing «Próximamente» (the real page is HU-02).

## 5. Integration

- [x] 5.1 [all] With `make supabase-start`, `make storage-start`, the API and the app running, publish a report as an anonymous neighbor with 3 photos, once anonymous and once with a name; verify the rows in `reports`/`report_photos`, the objects in the RustFS bucket without EXIF, `GET /reports/{id}` through the API (photo links open), and that the sixth report in 10 minutes gets the wait message.
- [x] 5.2 [all] Point `S3_*` at a different S3-compatible endpoint (or RustFS with another bucket and region) without code changes and verify publish and photo display still work.
- [x] 5.3 [all] Run `openspec validate add-publish-report`, `make test` (backend) and `bun run typecheck` (frontend), and confirm every task above is done before archiving.
