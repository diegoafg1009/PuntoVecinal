# Proposal

## Why

Step 3 of the Reportar form («Revisar») is still a «Próximamente» placeholder, so a neighbor can fill steps 1 and 2 but can never publish: no report is ever created, stored or seen. HU-10 (PV-19) adds the review and publish step, and HU-11 (PV-20) lets a neighbor without an account choose whether their name appears. HU-10 also says that after publishing the neighbor lands on the report detail, so this change includes the API to read one report (the backend side of HU-02, PV-11) needed for that. The report page itself is left to HU-02, and the Inicio feed (HU-03, PV-12) is left to HU-03.

## What Changes

- Replace the step 3 placeholder with «Revisar»: a summary of category, title, details, photos and place, each with a way back to the step that edits it.
- Add the «Acepto las reglas de convivencia» checkbox. Tapping «Publicar» without accepting does not publish and points at the checkbox.
- For a neighbor without an account, add the «Mostrar mi nombre o apodo» switch with a name field. Off (the default) publishes as «Vecino anónimo». The session that published is always recorded internally, never shown.
- Add `POST /reports` (multipart: report data plus 1 to 3 photos). The backend validates the data, cleans and re-encodes the photos, stores them in an S3-compatible bucket, creates the report with status Publicado tied to the caller's session, and answers with its id.
- Limit how many reports one neighbor can publish in a short time; above the limit the API answers «wait» and the app asks the neighbor to wait.
- Add `GET /reports/{id}` (API only): photos, category, status, place, reference, publication date and author name. On success the app opens `/reports/{id}` with the toast «Reporte publicado»; that page is a «Próximamente» placeholder until HU-02 builds it.
- Photos are served through temporary signed links, never public URLs.
- Object storage is any S3-compatible server: **RustFS locally** (started with a new `docker compose` file and `make` targets) and any S3 server in production, chosen only by configuration. The `SPACES_*` settings, unused so far, become generic `S3_*` settings.
- New database tables for reports and their photos (migration).
- Out of scope: the Inicio feed and `GET /reports` (all HU-03; a published report does not show in Inicio until then), the report page UI (photos carousel, map, author label «Vecino anónimo», 404 and error states; all HU-02), the similar-reports dialog, «También me afecta», contributions, the pilot-zone filter (dropped by the team), accounts and login, editing or deleting a report, notifications, the map and search.

## Capabilities

### New Capabilities

- `report-publishing`: what the API does when a report is published: validation, photo cleaning and storage, anonymous or named authorship, ownership by session, publication rate limit.
- `report-detail`: reading one report through the API (the page is HU-02).

### Modified Capabilities

- `report-creation`: adds the step 3 «Revisar» requirements: summary, community rules, author name choice, and the publish outcomes (draft cleared on success, kept on failure).

## Impact

- `openapi/`: new `reports` module (`POST /reports` multipart, `GET /reports/{id}`), config for oapi-codegen, root file reference and tag.
- `supabase/migrations/`: new migration for `reports` and `report_photos` (PostGIS point, status enum, author session id).
- `backend/`: new `internal/reports` module (handler, service, DTOs, domain, repository with sqlc), an S3 storage client and an image cleaner in `internal/platform`, a sqlc entry, config `Spaces` renamed to `S3` (with an optional public endpoint and path-style switch), `.env.example`, `compose.yaml` with RustFS, new `make storage-start`/`storage-stop`, and the request validation middleware no longer reading multipart bodies. New Go dependency for S3 (AWS SDK v2).
- `frontend/`: new `ReportStepReview` component, `pages/report.vue` (step 3, Publicar), new placeholder `pages/reports/[id].vue`, `reportDraft` store (author name, clear after publish), regenerated API client in `app/api/`.
- Docs: `README.md` and `backend/AGENTS.md` mention generic S3 / RustFS instead of DigitalOcean Spaces.
