# Design

## Context

See proposal.md for the motivation and the requirements in `specs/`. Current state:

- `GET /reports/{id}` (`getReport`) already exists and returns the full report with signed photo links (`report-detail`). The `reports` module has its service, `PgRepository`, sqlc queries in `backend/internal/reports/repository/queries/reports.sql` and the `PhotoStore.URL` signer (presigning is local, no network round trip).
- `reports` stores `created_at`, `location geography(Point,4326)` (GiST index) and photos in `report_photos (report_id, position)`. The only `created_at` index is `(author_id, created_at desc)`, for the rate limit.
- The request logger records only `r.URL.Path`, not the query string.
- Frontend: `pages/index.vue` shows the session id; `pages/reports/[id].vue` is a `ComingSoon` placeholder that Reportar already opens after publishing (with the «Reporte publicado» toast). `layouts/default.vue` hides both mobile bars when `route.meta.focusedOnMobile`. `ReportLocationMap.vue` is the draggable Leaflet map of step 2. Hey API generates TanStack helpers from the contract into `app/api/`.
- Figma (file `sC2o7u0tojnA5XjhlrE9Ts`, mobile frames «01 · Inicio» `5508:1539` and «04 · Detalle del reporte» `5509:1714`; PC frames `5515:1539` and `5517:1814`, «Mismo contenido y orden que en celular … centrado (máx. 560 px)»). Card: badges for category and status, relative time at the right, photo, title, and a line with pin icon «120 m · Jr. Ejemplo cdra. 3». Detail: own top bar (back, «Reporte», «⋯»), photo with «1 / 2» badge, badges, title, meta line «Vecino anónimo · hace 2 h · Jr. Ejemplo cdra. 3», details, then blocks of later stories.

## Goals / Non-Goals

**Goals:**
- A stable, cursor-based `GET /reports` that later filters (category, status, search) can extend with query parameters.
- Reuse the existing signing and report mapping code; no second way to build photo links.
- One set of formatters (status label, relative date, distance) shared by the card and the detail.

**Non-Goals:**
- Zones as data, moderation-hidden reports, filters, ordering choices, counters and every detail block of later stories (see proposal.md).
- Caching photos or links beyond what TanStack Query already does.

## Decisions

### Contract: `GET /reports` (`listReports`)

In `openapi/modules/reports.yaml`, under the existing `/reports` path (next to `post`), tag `reports`, bearer auth inherited:

- Query parameters: `cursor` (string, optional), `limit` (integer, 1–50, default 20), `latitude` (number, -90..90, optional), `longitude` (number, -180..180, optional).
- `200` → `ReportPage { items: ReportSummary[], nextCursor: string | null }` (both required; `nextCursor` nullable).
- `ReportSummary { id (uuid), title, category (common `Category`), status (`ReportStatus`), reference (nullable), createdAt (date-time), coverPhoto (`ReportPhoto`), distanceMeters (integer, nullable) }`, all required so the generated types have no optional fields. `distanceMeters` is null when no location was sent.
- `400` (text) for a bad cursor, a `limit` out of range, only one coordinate, or a coordinate out of range; `401` from common.
- New schemas stay in `reports.yaml` (single-module). Root `openapi.yaml` already references `/reports`; only the tag description changes to «Publishing and reading neighbor reports.»

Alternatives: offset pagination (`page`/`offset`) was rejected because new reports shift the offsets while a neighbor scrolls, which repeats items; the spec forbids that. Returning full `Report` items was rejected: it signs up to 3 links per item and exposes `authorName`/`details` that the card does not need.

### Keyset pagination on `(created_at, id)`

Order `created_at desc, id desc`; the cursor is base64url of `"<created_at RFC3339Nano>|<id>"`, decoded and validated by the service (bad format → `ErrInvalidCursor` → 400). The query fetches `limit + 1` rows; if it gets the extra one, `nextCursor` is built from the last returned item. A migration adds `create index reports_created_at_id_idx on public.reports (created_at desc, id desc)`.

```sql
-- name: ListReports :many
-- Never selects author_id nor author_name.
SELECT r.id, r.title, r.category_slug, c.label AS category_label, r.status,
       r.reference, r.created_at, p.storage_key AS cover_key,
       CASE WHEN sqlc.narg('lat')::float8 IS NULL THEN NULL
            ELSE round(ST_Distance(r.location,
                   ST_SetSRID(ST_MakePoint(sqlc.narg('lng')::float8, sqlc.narg('lat')::float8), 4326)::geography))::int
       END AS distance_meters
FROM reports r
JOIN categories c ON c.slug = r.category_slug
JOIN report_photos p ON p.report_id = r.id AND p.position = 1
WHERE sqlc.narg('after_created_at')::timestamptz IS NULL
   OR (r.created_at, r.id) < (sqlc.narg('after_created_at')::timestamptz, sqlc.narg('after_id')::uuid)
ORDER BY r.created_at DESC, r.id DESC
LIMIT @row_limit;
```

The inner join on position 1 is safe: every report has at least one photo (`report-publishing`). The exact SQL may be adjusted to what sqlc infers (for example the nullable `distance_meters`), without changing behavior. Postgres timestamps have microsecond precision; the cursor keeps the full value so ties are broken by `id`.

Alternative: cursor as a report id only (look up its `created_at`) — one more query per page and breaks if the report is ever removed; rejected.

### Backend module shape

- `domain.go`: `ReportSummary` (entity returned by the repository: id, title, category slug/label, status, reference, created at, cover storage key, distance `*int`), `ErrInvalidCursor`.
- `dto.go`: `ListReportsInput { Cursor string; Limit int; Latitude, Longitude *float64 }`, `ReportSummaryDTO` (with `CoverPhoto PhotoDTO`, `DistanceMeters *int`), `ReportPageDTO { Items; NextCursor string }`.
- `service.go`: `Repository` gains `List(ctx, ListParams) ([]ReportSummary, error)` (`ListParams` in `domain.go`: after-key, limit, optional point). `Service.List` validates limit and coordinates (both or neither, in range → `invalid(...)`, reusing `ErrInvalidReport` for the 400 mapping), decodes the cursor, asks for `limit+1`, signs each cover with `photoLinkTTL`, builds the next cursor.
- `repository.go`: `PgRepository.List` maps sqlc rows to `ReportSummary`.
- `handler.go`: `ListReports` maps params to the input and DTOs to `api.ReportPage`; `ErrInvalidReport`/`ErrInvalidCursor` → 400 text. The generated binding already rejects non-numeric `limit`/coordinates with 400.

### Location privacy

The neighbor's coordinates travel only as query parameters of this request; they are not stored and the request logger records only the path. The frontend rounds them to 4 decimals (about 11 m) before sending, enough for «120 m» and less precise than the device position. The order does not depend on the location, so the cursor never carries it.

### Frontend: Inicio feed

- `pages/index.vue` uses `useInfiniteQuery` with the generated `listReportsInfiniteOptions` when Hey API produces it for the `cursor` parameter; otherwise `useInfiniteQuery` with `queryFn` calling `listReports` and `getNextPageParam: page => page.nextCursor ?? undefined`. The query key includes the rounded location, so sharing it reloads the list from the first page.
- `staleTime` of 5 minutes and `gcTime` below one hour, so cached pages never outlive their signed links. An `<img>` that fails (expired link or storage down) shows the placeholder (`@error`).
- Continuous loading: a sentinel element after the last card watched with `useIntersectionObserver` (VueUse, already a Nuxt UI dependency) calls `fetchNextPage()` when visible, `hasNextPage` and not already fetching. A failed next page shows an inline «Reintentar» under the cards.
- States: `USkeleton` cards while the first page loads; empty state «Aún no hay reportes» with a «Reportar» button to `/report`; error «No pudimos cargar los reportes.» with «Reintentar» (`refetch`).
- Header row: «Más recientes» on the left and «Usar mi ubicación» (`UButton`, ghost, `i-lucide-locate-fixed`) on the right, in the place of the out-of-scope «Ordenar». When distances are on, the text reads «Cerca de ti · más recientes» as in Figma and the button is hidden.
- Location: a small `useNeighborLocation` composable calls `navigator.geolocation.getCurrentPosition` only from the button, stores `locationShared = true` in `localStorage` (try/catch) on success, and on later visits, if that flag is set and `navigator.permissions.query({ name: 'geolocation' })` says `granted`, gets the position without a prompt. Decline/failure → toast «No pudimos obtener tu ubicación. Los reportes se muestran sin distancia.» and clears the flag. The position itself is kept only in memory.
- `ReportCard.vue`: `UCard` as a `NuxtLink` to `/reports/{id}`; top row `UBadge`s (category neutral, status colored) and relative time; photo (`aspect-[7/3]`, `object-cover`, placeholder with `i-lucide-camera`); title (`line-clamp-2`); bottom line with `i-lucide-map-pin` and `«distance · reference»`, each part only when present, the line hidden when both are missing.

### Frontend: shared formatters

`app/utils/reportFormat.ts` (auto-imported):
- `statusLabel(status)`: `publicado` «Publicado», `derivado` «Derivado», `recibido` «Recibido por la entidad», `en_atencion` «En atención», `atendido` «Atendido según la entidad», `cerrado` «Cerrado»; plus `statusColor` (neutral for Publicado, info for Derivado/Recibido, warning for En atención, success for Atendido, neutral for Cerrado).
- `publishedAgo(date, now)`: under 1 min «hace un momento»; minutes «hace 5 min»; hours «hace 2 h»; calendar yesterday «ayer»; under 7 days «hace 3 días»; else `Intl.DateTimeFormat('es-PE', { day: 'numeric', month: 'short', year: 'numeric' })` («12 nov 2026»). Hand-written thresholds instead of `Intl.RelativeTimeFormat` alone, because its output for the short style varies by browser («hace 2 h.»).
- `formatDistance(m)`: under 1000 m, rounded to tens with a minimum of 10 («120 m»); otherwise kilometers with one decimal and a comma («1,2 km»).

### Frontend: report page

- `pages/reports/[id].vue` uses `useQuery(getReportOptions({ path: { id } }))` with the same stale/gc times. 404 (from the response status) → «No encontramos este reporte.» with a button «Ir a Inicio»; other errors → «No pudimos cargar el reporte.» with «Reintentar»; loading → `USkeleton` blocks.
- Order as in Figma: photos, badges, title, meta line («author · publishedAgo · reference», author = `authorName ?? 'Vecino anónimo'`), details (`whitespace-pre-line`), then a read-only map of the place (below the details, because the Figma frame has no map but the story asks for the place).
- Photos: `UCarousel` with one item per photo and a `UBadge` «n / total» at the top right only when there are 2 or 3; fixed height (`h-[200px]` mobile, taller on desktop) with `object-cover`; image error → placeholder.
- Map: `ReportLocationMap.vue` gains a `readonly` prop: no dragging, no click to move, no scroll-wheel zoom, centered on the report at `PLACED_ZOOM`. Reusing it keeps one Leaflet setup (pin icon, tiles, attribution).
- Mobile top bar: the page sets `definePageMeta({ ownTopBarOnMobile: true })`; `layouts/default.vue` hides `AppTopBar` when that meta is true but keeps `AppBottomBar` (already `fixed`) and its bottom padding. The page renders its own bar (sticky at top on mobile; on desktop it is the header of the centered panel, as in the PC frame `5517:1814`): `UButton` `i-lucide-arrow-left` with `aria-label="Volver"` and «Reporte». Back: `router.back()` when `window.history.state?.back` exists (set by Vue Router for in-app navigation), else `navigateTo('/')`. After publishing, Reportar opens the page with `navigateTo`, so back returns to Reportar's previous screen; acceptable since the draft is cleared.
- Desktop: the default layout already keeps the header; the page content is `max-w-[560px] mx-auto`, like the Reportar card.

## Risks / Trade-offs

- [Signing 20 links per page] → presigning is a local HMAC with no I/O, so it is expected to be negligible next to the query.
- [Links expire after one hour while a tab stays open] → `gcTime`/`staleTime` below one hour and an image fallback; returning to the tab refetches stale pages.
- [`ST_Distance` on geography for every row of the page] → only for the ≤51 rows of the page after ordering by the index; no full-table distance sort.
- [Permissions API not in every browser (older Safari)] → without it, later visits simply do not restore distances until the neighbor taps the button again.
- [Back after publishing goes to the screen before Reportar, not Reportar] → intended; Reportar's draft was already cleared.
- [Figma blocks of later stories are left out] → the page leaves room under the details; later stories add their blocks without changing this one's order.

## Migration Plan

Apply the new index migration (`make db-reset` locally, `supabase db push` on deploy) before the API; it is additive and does not lock writes for long on the current small table. Deploy the API, then the frontend. Rollback: drop the index in a follow-up migration and redeploy the previous API and frontend; `GET /reports/{id}` and publishing are not affected.
