# Design

## Context

Steps 1 and 2 of Reportar exist and keep their data in the `reportDraft` Pinia store (text in localStorage, photo blobs in IndexedDB). Photos are already re-encoded as JPEG on the device (max side 1600 px). The backend has the modules `auth`, `categories` and `system`; only `categories` touches the database (sqlc, `PgRepository`). Config declares unused `Spaces` settings and there is no PostGIS table. There is no report page (HU-02) and Inicio is a debug card that HU-03 will replace with the feed. See proposal.md for scope.

## Goals / Non-Goals

**Goals:**
- One request publishes a report atomically, with its photos stored and cleaned by the server.
- Ownership by session is recorded, and the public author is optional.
- The published report can be read back through the API with its photos.
- Object storage works with RustFS locally and any S3-compatible server in production, by configuration only.

**Non-Goals:**
- The report page UI (HU-02), the feed (HU-03), filters, search or the map; similarity, reactions, contributions; editing or deleting reports.

## Decisions

### API: two operations in the `reports` module

New files: `openapi/modules/reports.yaml`, `openapi/config/reports.yaml`, a `reports` tag and the path references in `openapi/openapi.yaml`. Shared error responses reuse `common.yaml` (a `NotFound` text/plain response is added there if a second module will need it; otherwise it stays in the module).

- `createReport` — `POST /reports`, `multipart/form-data` with `category` (slug), `title`, `details`, `reference`, `latitude`, `longitude`, `authorName` (text) and `photos` (array of binary, `minItems: 1`, `maxItems: 3`). Responses: `201 {id, status}`, `400`, `401`, `413` (photo too large), `429` (header `Retry-After` in seconds), `500` via the shared handler.
- `getReport` — `GET /reports/{id}` (`id` uuid). `200 Report`: `id`, `title`, `details` (nullable), `category {slug,label}`, `status` (enum of the six statuses, slugs), `location {latitude, longitude}`, `reference` (nullable), `authorName` (nullable), `createdAt`, `photos [{url}]` ordered. `404` for an unknown id.

The author's user id never appears in any schema. Turning the status slug into its Spanish label and showing «Vecino anónimo» for a null `authorName` are part of the page in HU-02.

Presigned-upload alternative (client uploads straight to the bucket) was rejected: more round trips, orphan files, and the server could not clean photos it never sees (backend/AGENTS.md requires server-side processing).

The contract-driven middleware (`platform/auth`) validates request bodies against the contract and would try to read multipart files. It is configured to skip body validation (`openapi3filter.Options.ExcludeRequestBody`); authentication and routing still follow the contract, and the service validates every field itself. Safe because no other operation has a body.

The Hey API client builds `FormData` for multipart bodies; if it does not append each `photos` entry as a repeated field, a small `bodySerializer` fixes it (frontend task).

### Validation and limits live in the service

The service owns the rules from the specs and returns typed domain errors (`ErrInvalidReport`, `ErrTooManyReports`, `ErrNotFound`), which the handler maps to 400/429/404. The handler reads the multipart stream with a per-photo limit of 10 MB and a total cap of 35 MB (`http.MaxBytesReader`).

### Photo cleaning on the server

`internal/platform/imaging` decodes JPEG and PNG with the standard library (bounds checked with `image.DecodeConfig` before `Decode`, to avoid decompression bombs) and re-encodes as JPEG (quality 85). Decoding to pixels and encoding again drops EXIF/GPS and any other metadata. Orientation is already applied on the device; a photo that relies on an EXIF orientation tag is stored as decoded (accepted limitation). Images above 2000 px on the long side are rejected.

### Object storage: any S3-compatible server

The `reports` service declares a `PhotoStore` interface where it is consumed (`Put`, `Delete`, `URL(key, ttl)`), and `internal/platform/storage` implements it with AWS SDK for Go v2 over the S3 API. Nothing in the code names a provider; configuration decides:

| Setting | Meaning | Local (RustFS) | Production |
|---|---|---|---|
| `S3_ENDPOINT` | API endpoint the backend talks to | `http://127.0.0.1:9000` | the provider's endpoint |
| `S3_PUBLICENDPOINT` | endpoint put in signed links when browsers reach the server by another address (optional, defaults to `S3_ENDPOINT`) | `http://127.0.0.1:9000` | usually empty |
| `S3_REGION` | signing region | `us-east-1` | provider's region |
| `S3_BUCKET` | private bucket | `punto-vecinal` | the bucket |
| `S3_ACCESSKEY` / `S3_SECRETKEY` | credentials | `rustfsadmin` / `rustfsadmin` | real keys |
| `S3_FORCEPATHSTYLE` | `host/bucket/key` instead of `bucket.host/key` | `true` | `false` unless the provider needs it |

This renames `Spaces` to `S3` in `internal/platform/config` and `.env.example` (the old keys were never read by any code, so nothing breaks). Config validation requires the endpoint, bucket and keys at startup, like the Supabase settings, so a misconfigured deploy fails fast.

Keys are `reports/{reportID}/{position}.jpg`. Signed links (`GET`, 1 hour) are generated with a presign client built on `S3_PUBLICENDPOINT`, because the host is part of the signature. The bucket is private.

**Local RustFS:** `backend/compose.yaml` runs `rustfs/rustfs` (API `:9000`, console `:9001` at `/rustfs/console/`, volume `rustfs-data`) and a one-shot `amazon/aws-cli` service (`create-bucket`) that creates the bucket `punto-vecinal` if it does not exist. `make storage-start` / `make storage-stop` wrap `docker compose` (the names do not tie the targets to a vendor). `.env.example` ships the RustFS values uncommented as the local default, with the production variants as comments. Docker is already required by `make supabase-start`.

### Atomic publication

Order: validate → rate check → clean photos → generate the report id (UUID) → upload photos → insert report and `report_photos` rows in one transaction. If the upload or the insert fails, uploaded keys are deleted best-effort (a failed delete is logged; keys are namespaced by report id, so a leftover is unreachable garbage, never a visible report). Uploading before the insert is chosen because the reverse could leave a visible report without photos if the process died in between.

### Data model (migration)

```
report_status enum: publicado, derivado, recibido, en_atencion, atendido, cerrado
reports(
  id uuid pk default gen_random_uuid(),
  author_id uuid not null references auth.users(id),
  category_slug text not null references categories(slug),
  title text not null check (length(title) between 1 and 100),
  details text check (length(details) <= 1000),
  reference text check (length(reference) <= 150),
  location geography(Point,4326) not null,
  author_name text check (length(author_name) between 1 and 40),  -- null = «Vecino anónimo»
  status report_status not null default 'publicado',
  created_at timestamptz not null default now()
)
report_photos(report_id uuid references reports(id) on delete cascade, position smallint check (position between 1 and 3), storage_key text not null, primary key (report_id, position))
index reports(author_id, created_at desc)   -- rate limit and "my reports"
gist index on reports(location)             -- radius alerts later
```

The full status list exists now because the statuses are fixed and only move forward; this change only writes `publicado`. The migration enables PostGIS (`create extension if not exists postgis`). RLS is enabled with no policies for `anon`/`authenticated`: the API connects with its own role and the frontend never reads tables. The `author_id` filter rule applies to every future user-dependent query; reads for the detail are public to any session by design, and never select `author_id`.

sqlc: the insert builds the point in SQL (`ST_SetSRID(ST_MakePoint(@lng, @lat), 4326)::geography`); the detail query returns `ST_Y(location::geometry)` and `ST_X(location::geometry)` as plain floats, so no `geography` override is needed. A new `sqlc.yaml` entry points at `internal/reports/repository`.

### Rate limit: count recent rows

Before cleaning photos, the repository counts the author's reports with `created_at > now() - interval '10 minutes'`; 5 or more means `ErrTooManyReports`, and `Retry-After` is the seconds until the oldest of those 5 leaves the window. No Redis, and it survives restarts and multiple instances. Concurrent requests can overshoot slightly; acceptable for an abuse brake.

### Frontend

- `ReportStepReview.vue`: summary with an «Editar» button per block (sets `draft.step`), rules checkbox (`UCheckbox`) with the rules in a `UModal`, author switch (`USwitch` + `UInput`) only when `session.user.is_anonymous`, and «Publicar» driven by a TanStack mutation on `createReport`. The rules text is a fixed Spanish placeholder in the component, to be reviewed by the team.
- `reportDraft` gains `authorName` (persisted with the text; the switch is on when it is non-empty). Acceptance of the rules is never persisted.
- On success: `draft.reset()`, `navigateTo('/reports/{id}')`, toast «Reporte publicado». Errors: 429 → «Publicaste varios reportes seguidos. Espera unos minutos antes de publicar otro.»; anything else → «No pudimos publicar tu reporte. Inténtalo de nuevo.». Both keep the draft and the step. The pending mutation disables the button (double tap).
- `pages/reports/[id].vue`: placeholder with `ComingSoon` («Próximamente»), only so the address opened after publishing and from the cards is real. HU-02 replaces it (photos, map, author, 404 and error states, using the already available `getReport`).

## Risks / Trade-offs

- Publish carries up to three uploads in one request on mobile data → photos are already ≤1600 px JPEG; server limits are bounded. No progress bar in this change.
- Skipping body validation in the shared middleware → the service validates everything; tests check malformed fields get 400.
- Orphan files if cleanup fails → unreachable by key; a cleanup job is out of scope.
- Signed links expire in one hour; A client that keeps a response longer than that must refetch it; relevant for HU-02 and HU-03.
- Signed-link host mismatch (RustFS reached by Docker name vs `localhost`) → `S3_PUBLICENDPOINT` exists for this.
- Counting rows for the rate limit can overshoot under concurrency → acceptable.
- Community rules text is invented → flagged for team review before release.

## Migration Plan

Apply the migration (`make db-reset` locally, `supabase db push` on deploy) before deploying the API. Replace any `SPACES_*` variables in the environment with the `S3_*` ones and create a private bucket (locally `make storage-start` does it). Rollback: drop the two tables and the enum in a follow-up migration and remove the routes; no existing data depends on them.
