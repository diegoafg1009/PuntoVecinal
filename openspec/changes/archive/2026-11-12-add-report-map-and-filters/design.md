# Design

## Context

See proposal.md for why. The current state that shapes the approach:

- `internal/reports` already serves the feed (`GET /reports`, keyset `(created_at, id)`), the search (`GET /reports/search`) and the detail (`GET /reports/{id}`). A `/reports/search` route already sits next to `/reports/{id}` without clashing, and the new `/reports/map` follows the same pattern.
- Feed items (`ReportSummary`) carry no location, so the map cannot be built from the feed.
- `reports.status` is the `public.report_status` enum. `categories` holds the five fixed slugs, and `Repository.CategoryExists` checks one slug.
- The frontend already uses Leaflet with OpenStreetMap tiles in `ReportLocationMap.vue`, which also holds the assumed pilot center `{ lat: -11.936, lng: -77.05 }`. `useNeighborLocation` asks for the device location only through `share()` and reuses it silently through `restore()`.
- `utils/reportFormat.ts` has the Spanish status labels and badge colors. `useCategories()` gives the fixed categories.
- Mapa is currently a disabled item in `AppBottomBar.vue` and `AppHeader.vue`, which shows a «Próximamente» toast. `pages/map.vue` redirects to `/`.

## Goals / Non-Goals

**Goals:**
- One request draws the whole map for the pilot, which has few reports.
- The feed and the map apply the same filters with the same validation.
- The filters live in one place on the frontend, so Inicio and Mapa cannot disagree.
- The screens follow the Figma designs.

**Non-Goals:**
- Loading points by viewport (bbox), clustering or heatmaps.
- Adding a `status` or `category` index: the feed index and the pilot volume are enough.
- Changing `GET /reports/search` or Buscar.

## Decisions

### 1. Contract

**`listReports` (`GET /reports`)** gains two query parameters:
- `category`: an array of strings, `style: form`, `explode: true` (`?category=limpieza&category=alumbrado`), with `maxItems: 5` and `uniqueItems: true`.
- `status`: `$ref: ReportStatus`.

Its 400 description adds "an unknown category or status". The operation description says the filters narrow the list and that a cursor should be sent with the same filters. The order and the cursor format do not change.

**New `getReportMap` (`GET /reports/map`)**, tag `reports`:
- It has the same `category` and `status` parameters, factored as `components/parameters` (`ReportCategoryFilter`, `ReportStatusFilter`) in `reports.yaml`, so both operations reference one definition.
- `200` → `ReportMap`. `400` (text) for an unknown category or status. The common `401`.
- New schemas in `reports.yaml`:
  - `ReportMap`: `{ items: ReportPin[] }`, required `items`.
  - `ReportPin`: required `id` (uuid), `title`, `category` (`common.yaml#/Category`), `status` (`ReportStatus`), `location` (`Location`).
- The description states the 500-item cap, newest first, and that the author is never returned.
- `openapi/openapi.yaml` references `/reports/map` next to `/reports/search`.

Alternative considered: adding `location` to `ReportSummary` and drawing the map from the paginated feed. Rejected because the map would need to walk every page and sign a cover photo per point. Pins are cheap and need no signed links. The preview photo comes from `getReport`, which is called only for the tapped point.

Alternative considered: `?bbox=` on `/reports/map`. Rejected for now because the pilot is one small zone. The cap keeps the response bounded, and a bbox can be added later without breaking clients.

### 2. Filter semantics and validation (service)

- The service validates the filters for both `List` and `Map` through one helper, `checkFilters(ctx, categories, status)`:
  - It drops duplicate slugs.
  - It rejects more than 5 slugs. An unknown slug answers 400 (`ErrInvalidReport`, "unknown category"). This uses a new `Repository.UnknownCategories(ctx, slugs) ([]string, error)` (one query, `SELECT unnest(@slugs::text[]) EXCEPT SELECT slug FROM categories`), instead of calling `CategoryExists` in a loop.
- The status is already an enum in the generated code. The service still checks it against the six known values, so it does not depend on the HTTP layer.
- `ListParams` gains `Categories []string` and `Status string` (empty = any). A new `MapParams{Categories, Status, Limit}` is added.
- The feed cursor stays `(created_at, id)` and carries no filters. A cursor used with different filters still continues after the same key, which is safe because it neither repeats nor skips rows within the new filter. So it is not rejected. The frontend always sends the same filters, because they are part of the query key.

### 3. Queries (sqlc)

In `backend/internal/reports/repository/queries/reports.sql`:
- `ListReports` gains:
  ```sql
  AND (cardinality(@categories::text[]) = 0 OR r.category_slug = ANY(@categories::text[]))
  AND (sqlc.narg('status')::report_status IS NULL OR r.status = sqlc.narg('status')::report_status)
  ```
  The current `WHERE` on the cursor gets wrapped in parentheses.
- New `ListReportPins :many`: it selects `r.id, r.title, r.category_slug, c.label, r.status, ST_Y(...) AS latitude, ST_X(...) AS longitude`, with the same two filters, `ORDER BY r.created_at DESC, r.id DESC LIMIT @row_limit`. It never selects `author_id` or `author_name`, and needs no photo join.
- New `UnknownCategories :many`, as above.

There is no migration. The project rule names `internal/platform/database/queries/`, but the module's queries actually live in `internal/reports/repository/queries/` (per-module sqlc layout), so they go there.

### 4. Backend layers

- `domain.go`: `ReportPin` (ID, Title, CategorySlug, CategoryLabel, Status, Latitude, Longitude), `MapParams`, and the filter fields on `ListParams`.
- `dto.go`: `ListReportsInput` gains `Categories []string` and `Status string`. New `MapInput{Categories, Status}` and `ReportPinDTO`.
- `Repository` interface: `ListPins(ctx, MapParams) ([]ReportPin, error)` and `UnknownCategories`.
- `Service.Map(ctx, MapInput)` validates the filters and calls `ListPins` with `Limit: 500` (`maxMapPins`).
- `Handler.GetReportMap` maps params and DTOs to `api.ReportMap`, and invalid input to 400. `Handler.ListReports` passes the two new params.

### 5. Frontend: filters store

`stores/reportFilters.ts` (Pinia, client state, not persisted) holds:
- `categories: string[]` and `status: ReportStatus | null`;
- `activeCount` (number of categories + 1 if there is a status), used to know whether any filter is on;
- `apply({ categories, status })`, `toggleCategory(slug)`, `setStatus(status)` and `clear()`;
- `queryParams`, a computed that builds `{ category?, status? }` and leaves out empty values so the query key stays stable.

Being in memory gives exactly the spec: the filters are shared by Inicio and Mapa, kept when going back from a report, and cleared on reload. They are not kept in the address because the navigation links are plain `/` and `/map` and would drop the query (proposal, assumption).

### 6. Frontend: filter chips

The Figma frames (Inicio, Buscar, Mapa, mobile and desktop) show the filters as a row of chips. `components/ReportFilterChips.vue` is one row used by Inicio and Mapa:
- Small `UButton`s (`size="xs"`) in a horizontally scrollable row (`overflow-x-auto`, no wrapping, hidden scrollbar).
- **«Todo»**: active (primary) when `activeCount` is 0; tapping it calls `clear()`. In Inicio the active style is `soft`, in Mapa it is `solid`, as in Figma.
- **«Estado ▾»**: a `UDropdownMenu` (a radio group of items) with «Todos» first and then the six labels from `statusLabel`. It sets the status at once. With a status chosen, the chip is active and shows that status label instead of «Estado».
- **One chip per category**, from `useCategories()` in the API order. A chip toggles its category (`toggleCategory`). Active is primary `soft` (Inicio) or `solid` (Mapa); inactive is `outline` neutral.
- Position of «Estado»: second in Inicio and last in Mapa, as in the frames, through a prop.
- The chips apply immediately, with no panel, no «Aplicar» and no local draft. Chips are `aria-pressed` toggles so the state is announced.

Buscar also has filter chips in Figma, but filters in Buscar stay out of scope.

### 7. Frontend: Inicio

`pages/index.vue` changes:
- `ReportFilterChips` goes in a row under the heading, above the cards; «Usar mi ubicación» stays in the header row.
- `filters.queryParams` is spread into `listReportsInfiniteOptions({ query })`, so a filter change resets to the first page through the query key.
- The empty state branches on `filters.activeCount`:
  - With filters on: «Ningún reporte coincide con los filtros.» and «Quitar filtros».
  - Without: the current «Aún no hay reportes» and «Reportar».

### 8. Frontend: Mapa

`pages/map.vue` replaces the redirect. `components/ReportPinsMap.vue` is the Leaflet map, separate from `ReportLocationMap.vue`, which is a single draggable pin.
- **Data:** `useQuery(getReportMapOptions({ query: filters.queryParams }))`, enabled with a session, `staleTime` 5 min. Pins carry no signed links, so there is no gc constraint.
- **Pins:** an `L.layerGroup` of `L.marker`s with a `divIcon` that draws the lucide map-pin outline (as `ReportLocationMap.vue` does), refreshed when the data changes. This is the Figma look: outline pins with a ring in the middle.
  - The stroke color comes from a new `statusPinColor(status)` in `utils/reportFormat.ts`, read from the Nuxt UI CSS variables (`--ui-error`, `--ui-info`, `--ui-warning`, `--ui-success`, `--ui-text-dimmed`), and the pin interior is white.
  - `publicado` uses error (red; the theme primary is the same green as success, so Publicado and Atendido would look alike), `derivado` and `recibido` info, `en_atencion` warning, `atendido` success and `cerrado` dimmed. `derivado` and `recibido` share a color as their badges do, and the legend groups them: «Derivado / Recibido por la entidad».
  - Clustering (the «4» bubble in Figma) is a non-goal, so overlapping pins simply overlap.
- **Legend:** Figma has none, but the spec needs one for the status colors, so a small `UCard` is overlaid in a top corner below the chips, collapsed by default on mobile.
- **Preview:** tapping a pin sets `selectedId`.
  - A card at the bottom of the map, as in Figma, shows a 56 px photo thumbnail, the title, a category `UBadge` and a status `UBadge` at once, and a chevron. The whole card is a link to `/reports/{id}`. There is no distance, because pins carry none.
  - The photo comes from `useQuery(getReportOptions({ path: { id } }))` with a skeleton while loading and a camera placeholder on error. The 1 h link TTL is handled by `gcTime` 30 min, as Inicio does.
  - Tapping the map closes the preview.
- **Centering:** `restore()` on mount, and a round icon-only locate button (`i-lucide-locate-fixed`, outline, `aria-label="Usar mi ubicación"`) at the right edge that calls `share()`.
  - `watch(location)` flies to it at zoom 16.
  - Without a location, the map uses `PILOT_CENTER` at zoom 15. `PILOT_CENTER` and the zoom move from `ReportLocationMap.vue` to `utils/pilotZone.ts`, and both maps import them.
  - The neighbor's own position is not drawn as a report pin. At most a distinct "you are here" dot is shown, which is never a report.
- **States:** a loading `UBadge`/spinner overlay while `isPending`. An error overlay with «Reintentar» (`refetch`). Two empty overlays: «Aún no hay reportes» with «Reportar» when there are no filters, and the filters empty state with «Quitar filtros» when there are.
- **Layout:**
  - On mobile, the map fills the space between the top bar and the bottom bar (`h-[calc(100dvh-…)]`). The chips float at the top left over the map, the locate button at the right, the preview card at the bottom.
  - On desktop, a centered panel titled «Mapa», like Inicio's but wider (max-w ~960px), with the map inside and the same floating chips, locate button and preview card.
- The Figma «Mapa» frames (mobile `5509:1594`, desktop `5517:1670`) were read at apply time.

### 9. Frontend: navigation

- `AppBottomBar.vue` and `AppHeader.vue` turn Mapa into a normal item with `to: '/map'`, and drop `unavailable`, `showMapSoon`, the «Pronto» badge and `aria-disabled`.
- `isCurrent('/map')` already works through `startsWith`.

## Risks / Trade-offs

- [Pilot coordinate is an assumption] → It is kept in one constant (`utils/pilotZone.ts`) so it can be fixed in one place once the real coordinates are confirmed.
- [The 500 cap hides older reports when there are more] → The pilot is far below it. The cap is documented in the contract, and a bbox parameter is the follow-up if it is ever reached.
- [OpenStreetMap public tiles have a usage policy] → They are already used by Reportar. The pilot volume is low, and the attribution stays visible.
- [Filters lost on reload] → This is accepted for the MVP and written into the spec. Moving them to the address later is a frontend-only change.
- [Category validation adds a query to every filtered feed request] → It is one small indexed query, run only when `category` is sent.
- [The preview photo needs a second request] → It is only for the tapped point. The title and status show at once from the pin.

## Migration Plan

There is no data migration. Deploy the backend first: the new parameters are optional and the new path is additive, so the current frontend keeps working. Then deploy the frontend. Rollback is reverting the frontend and then the backend, with no data to undo.
