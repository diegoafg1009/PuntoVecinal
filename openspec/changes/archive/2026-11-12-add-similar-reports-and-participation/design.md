# Design

## Context

See proposal.md for why. The current state that shapes the approach:

- `internal/reports` serves `POST /reports` (multipart, read by `readForm` under `limitBody`), `GET /reports`, `/reports/map`, `/reports/search` and `/reports/{id}`. `Service.Create` cleans photos with `imaging.Clean`, uploads them, inserts the report, and removes the uploads if the insert fails. `checkRate` limits publishing to 5 reports per 10 minutes, and `encodeCursor`/`decodeCursor` handle keyset cursors over `(created_at, id)`.
- `reports` has no embedding column, and there is no support or contribution table. `GetReport` never selects `author_id`.
- `config.Voyage{APIKey}` is loaded from `VOYAGE_APIKEY` but nothing uses it. The backend docs already name Voyage AI and pgvector for text similarity.
- The auth middleware puts `Claims` (user id, anonymous flag) in the context. Every session is anonymous today, since accounts do not exist yet.
- Frontend: `ReportStepPhotoAndPlace.next()` sets `draft.step = 3`. `ReportStepReview.vue` publishes with `createReport` and then calls `navigateTo('/reports/{id}')` and `toast.add('Reporte publicado')`. `utils/photo.ts#processPhoto` cleans and reduces a photo, and the draft keeps photos as `{ id, blob, url }`. `pages/reports/[id].vue` uses `getReport` through TanStack Query.
- Figma (mobile page): `07 · ¿ya lo reportaron?` (`5550:1997`), `10 · Sumarte` (`5551:2019`), `11 · Te sumaste` (`5551:2110`) and `04 · Detalle` (`5509:1714`). The PC page has the same frames: `5553:2655`, `5553:2807`, `5553:2980` and `5517:1814`. The note on frame 10 says «Tu aporte» holds the text and the photo from steps 1 and 2 and always includes them. The design shows one photo. Frame 11's toast reads «Te sumaste al reporte» / «Ahora a 15 vecinos les afecta. Tu aporte ya aparece abajo.». The product owner confirmed that the «Aportar» button of frame 04 opens frame 10, «Sumarte a este reporte», so aportar and sumarse are one action.

## Goals / Non-Goals

**Goals:**
- Similar reports are found by meaning, among nearby open reports, with one Voyage call per check.
- Publishing never fails or slows down noticeably because of Voyage.
- Joining is a single atomic request, so support and aporte cannot be half-saved.
- Support and aportes never reveal who the neighbor is.

**Non-Goals:**
- Comments, «¿Cómo está ahora?», case tracking, the «Con cuenta» variants, notifications.
- A vector index (HNSW/IVFFlat): the radius filter on the GiST index leaves few rows at pilot scale.
- Showing the affected count on Mapa, since its Figma frame does not show it. Inicio and Buscar do show it (§10).
- Re-embedding a report when its text changes, since reports cannot be edited.

## Decisions

### 1. Contract (`openapi/modules/reports.yaml`, all tag `reports`)

**`findSimilarReports` — `POST /reports/similar`.** The body is JSON, because the text can be up to 1100 characters and a query string is a bad place for it.
- `SimilarReportsRequest` `{ title (string, maxLength 100), details? (string, maxLength 1000), latitude, longitude }`, all required except `details`.
- `200` → `SimilarReports` `{ items: SimilarReport[] }`, with `maxItems: 5`.
- `SimilarReport` `{ id, title, category (Category), status (ReportStatus), createdAt, coverPhoto (string, uri), distanceMeters (integer), affectedCount (integer, ≥1) }`.
- `400` (text), common `401`, and `503` (text) when the comparison service is unavailable.

`POST /reports/similar` does not clash with `GET /reports/{id}`, the same way `/reports/search` and `/reports/map` don't.

**`supportReport` — `PUT /reports/{id}/support`** and **`unsupportReport` — `DELETE /reports/{id}/support`.** Neither takes a body.
- `200` → `ReportSupport` `{ affectedCount (integer), support (SupportState) }`.
- Common `401` and `404`, and `409` (text) when the caller is the author.
- `SupportState` is a string enum `[author, supporter, none]`.
- `id` is a plain string, as in `getReport`, so a malformed id answers 404.

**`listReportContributions` — `GET /reports/{id}/contributions`.**
- Parameters `cursor` and `limit` (1–50, default 20).
- `200` → `ContributionPage` `{ items: Contribution[], nextCursor (nullable string) }`.
- `400`, common `401` and `404`.
- `Contribution` `{ id, text, authorName (nullable), createdAt, photoUrl (nullable, uri) }`.

**`createReportContribution` — `POST /reports/{id}/contributions`.** The body is `multipart/form-data` `CreateContributionRequest` `{ text (required, maxLength 1200), authorName? (maxLength 40), photo? (binary) }`. Every aporte also marks the caller's support, so there is no flag.
- `201` → `CreatedContribution` `{ contribution: Contribution, affectedCount, support: SupportState }`.
- `400`, common `401` and `404`, `413`, and `429` with `Retry-After`.

**`Report`** gains the required fields `affectedCount` (integer), `contributionCount` (integer) and `support` (`SupportState`).

**`ReportSummary`** (the items of `listReports` and `searchReports`) gains the required field `affectedCount` (integer, ≥1). `ReportPin` (Mapa) does not change.

Root `openapi/openapi.yaml` references the three new paths: `/reports/similar`, `/reports/{id}/support` and `/reports/{id}/contributions`.

Why there is no separate `/join` endpoint and no `support` flag: in the product, aportar and sumarse are the same action. «Aportar» on the report page opens «Sumarte a este reporte», so every aporte is "aporte + support". One operation keeps one validation path, one rate limit and one transaction. The 1200-character text limit fits a join aporte, which is title + blank line + details: 100 + 2 + 1000.

### 2. Database (new migration `..._add_similarity_and_participation.sql`)

```sql
create extension if not exists vector with schema extensions;

-- Meaning of title + details (Voyage, 1024 dims). Null until embedded.
alter table public.reports add column embedding extensions.vector(1024);

create table public.report_supports (
  report_id uuid not null references public.reports (id) on delete cascade,
  user_id uuid not null references auth.users (id),
  created_at timestamptz not null default now(),
  primary key (report_id, user_id)
);

create table public.report_contributions (
  id uuid primary key default gen_random_uuid(),
  report_id uuid not null references public.reports (id) on delete cascade,
  author_id uuid not null references auth.users (id),   -- never shown
  author_name text check (length(author_name) between 1 and 40),
  text text not null check (length(text) between 1 and 1200),
  photo_key text,
  created_at timestamptz not null default now()
);
create index report_contributions_report_idx on public.report_contributions (report_id, created_at, id);
create index report_contributions_author_idx on public.report_contributions (author_id, created_at desc);
create index reports_embedding_missing_idx on public.reports (created_at) where embedding is null;

alter table public.report_supports enable row level security;       -- no policies: only the API role
alter table public.report_contributions enable row level security;
```

The affected count is computed rather than stored: `1 + count(report_supports where user_id <> reports.author_id)`. The service never inserts the author as a supporter, and the query also excludes them as a second barrier. Pilot volumes make the subquery cheap. A stored counter would need triggers and could drift.

### 3. Queries (`reports.sql`) and sqlc

- `ListReports`, `SearchReportsRecent` and `SearchReportsNearest` gain `affected_count`, using the same subquery as `GetReport`. The feed and search queries do not change their order or cursors.
- `GetReport` gains a `@caller` argument and returns:
  - `affected_count`;
  - `contribution_count`;
  - `(r.author_id = @caller) AS is_author`;
  - `EXISTS (SELECT 1 FROM report_supports s WHERE s.report_id = r.id AND s.user_id = @caller) AS supported`.

  It still never selects `author_id`.
- `GetReportAuthor :one`: returns `author_id`, used only internally for 404/409 decisions.
- `InsertReport` gains `sqlc.narg('embedding')`.
- `ListReportsMissingEmbedding :many` (limit) and `SetReportEmbedding :exec`.
- `FindSimilarReports :many`. Within `ST_DWithin(location, point, 300)`, it keeps rows where:
  - `embedding IS NOT NULL`;
  - `status NOT IN ('atendido','cerrado')`;
  - `embedding <=> @embedding <= @max_distance`.

  It orders by `embedding <=> @embedding`, then distance, then id, with `LIMIT 5`, and returns the card fields, `distance_meters`, `affected_count` and the cover key.
- `InsertSupport :exec` (`ON CONFLICT DO NOTHING`), `DeleteSupport :exec`, and `CountAffected :one`.
- `InsertContribution :exec`, `CountRecentContributions :one` (same shape as `CountRecentReports`), `ListContributions :many` (keyset ascending on `(created_at, id)`, never `author_id`), and `GetContribution :one` (used for the response).
- `sqlc.yaml` gets an override for `vector` (both `extensions.vector` and `vector`) to `github.com/pgvector/pgvector-go.Vector`. `database.New` registers the pgvector types in `AfterConnect` (`pgxvec.RegisterTypes`).

### 4. Embeddings (`internal/platform/embedding`)

- `Voyage` client: `POST https://api.voyageai.com/v1/embeddings` with `model: voyage-4` (multilingual, 1024 dims by default, 200 M free tokens), `input: [text]`, and `input_type: "document"` for both stored reports and checks. Duplicate detection is symmetric, since a draft and a report are the same kind of text. The client has an `http.Client` timeout and returns `ErrUnavailable` on transport errors, non-2xx answers, or an empty key.
- The text is always `title + "\n" + details` (just the title when there are no details). It is built in one function, `reports.comparableText`, so nothing else can reach Voyage.
- `reports` defines its consumer interface in `service.go`: `Embedder { Embed(ctx, text string) (pgvector.Vector, error) }`. `Register` and `NewService` take it.
- Cosine-distance threshold: `maxSimilarDistance` is a constant in the service, `0.3`. Task 3.6 calibrated it with `voyage-4` on 15 invented Spanish pairs (same problem 0.10–0.26, different problem 0.23–0.52). It is small, so retune it with real reports; the scale depends on the model. The reasoning is in the code comment.

Alternatives considered: `pg_trgm` or full-text search, which fail the "other words" criterion, and storing embeddings in a separate table, which adds a join for no benefit at this scale.

### 5. Publishing and backfill

- `Service.Create` calls `Embed` with its own 3 s timeout after validation, before the insert. On any error it logs at warn level (without the text) and inserts `embedding = NULL`. Publishing behaves as before otherwise.
- `reports.RunEmbeddingBackfill(ctx, repo, embedder, logger)`: a goroutine started from `cmd/punto-vecinal-api/main.go` with the server context. Every 5 minutes it embeds up to 20 reports where `embedding IS NULL`, oldest first, and stops the round on the first `ErrUnavailable`. It does nothing when the key is empty.

### 6. Similar check (`Service.Similar`)

1. Validate the input with the same title, details and coordinate rules as `validate`, using `invalid(...)` → 400.
2. `Embed` with a 5 s timeout. `ErrUnavailable` → `ErrComparisonUnavailable` → 503.
3. `FindSimilarReports`, then sign the cover photo links (`cards`-style helper) → DTOs.

### 7. Support (`Service.Support` / `Service.Unsupport`)

`GetReportAuthor` → `ErrNotFound` (404), or `ErrIsAuthor` (409) when it equals the caller. Then `InsertSupport`/`DeleteSupport` → `CountAffected` → `ReportSupportDTO{AffectedCount, State}`. Both operations are idempotent. They do not use a transaction, because the count is read after the write and a concurrent change only shifts it by one.

### 8. Contributions (`Service.Contribute` / `Service.Contributions`)

`Contribute(ctx, in ContributionInput{ReportID, AuthorID, Text, AuthorName, Photo []byte})`:
1. Validate: trimmed text 1–1200 and a name of at most 40 characters. The handler enforces at most 1 photo and the size limit.
2. `GetReportAuthor` → 404.
3. Rate: `CountRecentContributions` over 10 minutes, at most 10 → `TooManyContributionsError{RetryAfter}`, which becomes 429 with `Retry-After`. This mirrors `checkRate`.
4. If there is a photo: `imaging.Clean` → `contributions/<id>.jpg` → `Put`.
5. The repository `CreateContribution` runs one transaction: `InsertContribution`, and, unless the caller is the author, `InsertSupport ON CONFLICT DO NOTHING`. If it fails, the uploaded photo is deleted.
6. Return the contribution (with its signed photo URL), `CountAffected` and the state.

The handler reads the multipart body with a dedicated `readContributionForm`, structured like `readForm`. It rejects a second `photo` part with 400, applies `MaxPhotoBytes` per photo (413), and runs under `limitBody(MaxPhotoBytes + 64 KiB)`.

`Contributions(ctx, reportID, cursor, limit)`: `GetReportAuthor` for 404, then a keyset page ascending, with the same cursor encoding as the feed and signed photo links.

`Get` now takes the caller id and maps `is_author`/`supported` to `SupportState`.

### 9. Frontend: Reportar

- `stores/reportDraft.ts` gains in-memory state that is not persisted: `similar: SimilarReport[]`, `similarOpen: boolean` and `joinTarget: SimilarReport | null`. It also gains a `contributionText` getter (title, plus `\n\n` and details when there are any). `reset()` clears all of these.
- `ReportStepPhotoAndPlace.next()` (after the form validates) runs a `findSimilarReports` mutation with the draft's title, details and location. It shows `loading` on «Seguir» and ignores taps while the mutation is pending.
  - Items → `draft.similar = items; draft.similarOpen = true`.
  - Empty list or error → `draft.step = 3`.
- New `ReportSimilarDialog.vue`: a `UModal` that is a bottom sheet on mobile and centered on desktop, following Figma `5550:1997`/`5553:2655`.
  - Each item is a `UCard` with a photo (placeholder on error), the title, a category `UBadge`, «N m», a users icon with the count, and `publishedAgo`. The list scrolls in a max-height area.
  - Tapping an item sets `draft.joinTarget`.
  - «No es ninguno, seguir» → `similarOpen = false; step = 3`. The X button or dismissing the dialog → `similarOpen = false`.
- New `ReportJoin.vue`: the «Sumarte a este reporte» screen (Figma `5551:2019`/`5553:2807`). It is used from two places (§9 and §10). It takes the report's card data (`title`, `category`, `status`, `coverPhoto`, `affectedCount`, optional `distanceMeters`) and an `editable` flag. It emits `back`, and shows:
  - A top bar with back and «Sumarte a este reporte».
  - `ReportCompactCard.vue` (new) with the badges, the title, «A N m» when there is a distance, and «a N les afecta». Below it the text «Tu apoyo se suma a este caso; no se crea uno nuevo.».
  - `ReportAuthorNameField.vue` (new, extracted from `ReportStepReview.vue` and reused there): the `USwitch` «Mostrar mi nombre o apodo» with its `UInput`. It has the same rules as step 3 and uses `canChooseName` and `AUTHOR_NAME_MAX_LENGTH`.
  - «Tu aporte». When it is not editable, it shows the given text and photo. When it is editable, it shows a `UTextarea` (maxlength 1200) and one photo through `processPhoto` (take or choose, with a thumbnail and a remove button).
  - `UButton` «Sumarme a este reporte» and a link-style «No es el mismo, volver».
- `composables/useJoinReport.ts` sends the screen in both places. It calls `createReportContribution` with `text`, `authorName` and an optional `photo`, then:
  1. invalidates `getReport` and `listReportContributions` for that id;
  2. calls the caller's `onJoined` (Reportar uses it to run `draft.reset()`);
  3. calls `navigateTo('/reports/{id}', { replace: true })`;
  4. calls `toast.add({ title: 'Te sumaste al reporte', description: 'Ahora a ${affectedCount} vecinos les afecta. Tu aporte ya aparece abajo.', color: 'success' })`. This is the same pattern as «Reporte publicado», and `affectedCount` comes from `CreatedContribution`.

  On error it shows a Spanish message (429 → wait message) and keeps the screen.
- From Reportar, `pages/report.vue` renders `ReportJoin` (not editable) in place of the stepper while `draft.joinTarget` is set. This keeps the focused mobile layout and needs no nested route under `report.vue`. «Tu aporte» shows `draft.contributionText` and the first draft photo, which is also the photo that is sent. Back and «No es el mismo» run `joinTarget = null; similarOpen = true`. A reload leaves the page on step 2, because the join target is not persisted.

### 10. Frontend: report page

- `ReportCard.vue` (Inicio and Buscar) shows `affectedCount` next to an `i-lucide-users` icon, in the same row as the distance and age, as in Figma frames 01 (`5508:1539`) and 02 (`5522:1935`). It reuses the icon and count of `ReportCompactCard.vue`. After support changes, the feed and search caches are left to refresh on their own, without explicit invalidation.
- `ReportSupportButton.vue` shows «También me afecta · N».
  - Its variant is `soft` when supported or author, and `outline` otherwise.
  - It is disabled for `author`.
  - On tap it calls `supportReport`/`unsupportReport` and patches the `getReport` query data optimistically. It rolls back with a toast on error, and ignores taps while a request is pending, so the final state is the API's.
- «Aportar» (`UButton` outline) navigates to the new page `pages/reports/[id]/join.vue`. This needs `pages/reports/[id].vue` to move to `pages/reports/[id]/index.vue`, with no behavior change.
  - The page has `ownTopBarOnMobile` and renders `ReportJoin` with `editable`. The card data comes from the cached `getReport` query, so it is fetched if the cache is empty.
  - The distance is shown only when `useNeighborLocation.restore()` already has a position. The page never asks for the location.
  - Back and «No es el mismo, volver» return to the report: `router.back()` when the report is the previous entry, otherwise `navigateTo('/reports/{id}')`.
  - Submitting uses `useJoinReport`, so the result is the same as joining from Reportar: the toast «Te sumaste al reporte», the support marked and the aporte in the list.
  - The text and photo are held in local state only. Leaving the page drops them, which is acceptable for a short text.
- `ReportContributions.vue` shows «Aportes y comentarios · {report.contributionCount}». It uses `useInfiniteQuery` on `listReportContributions` with a «Ver más» button, and shows empty, error (with «Reintentar») and loading states. Each item has a `UAvatar`, «{authorName ?? 'Vecino anónimo'} · {publishedAgo}», the text and the photo (placeholder on error).
- Order on the page, as in Figma: header data → support + Aportar row → map → aportes. The Figma cards «¿Cómo está ahora?» and «Seguimiento del caso», the comment input and the «…» menu are not built.

## Risks / Trade-offs

- **Threshold quality**: a fixed cosine threshold can miss real duplicates or show unrelated reports. Mitigation: calibrate with sample pairs (task 3.6). Because the threshold is a single constant, retuning is a one-line change.
- **Voyage dependency**: the check is unavailable without a key or network, and the app silently skips it. Reports published meanwhile are embedded by the backfill, which can take up to about 5 minutes.
- **Cost and privacy**: one Voyage call per publish and one per check. Only the title and details are sent, and that is enforced by a single text builder.
- **Abuse of support**: anyone can create new anonymous sessions to inflate counts. That is accepted for the pilot, the same as for publishing.
- **Optimistic toggle**: a quick double tap is ignored while pending, rather than queued. That is simpler, and the final state is always the server's.
- **Text limit of 1200** on aportes is a little unusual. It is chosen so a joined draft always fits without truncation.
