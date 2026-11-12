# Tasks

## 1. Contract

- [x] 1.1 [backend] Update `openapi/modules/reports.yaml` as design.md §1 describes.
  - Add the `ReportCategoryFilter` parameter (`category`, an array of strings, form/explode, `maxItems: 5`, `uniqueItems`) and the `ReportStatusFilter` parameter (`status`, `ReportStatus`) under `components/parameters`.
  - Reference both from `listReports`, and extend its description and 400 text.
  - Add `getReportMap` (`GET /reports/map`), which answers `200` with `ReportMap`, `400` (text) and the common `401`.
  - Add the `ReportMap` and `ReportPin` schemas.
  - Reference `/reports/map` in `openapi/openapi.yaml`.

  Verify that `make bundle` (from `backend/`) succeeds and that the bundle has `/reports/map` and the two parameters on `/reports`.
- [x] 1.2 [backend] Run `make generate-api` from `backend/`. Verify the following:
  - `internal/reports/api/generated.go` has `GetReportMap` in `StrictServerInterface`.
  - `ListReportsParams` has `Category *[]string` and `Status *ReportStatus`.
  - `go build ./...` fails only because `Handler` does not implement `GetReportMap` yet.

## 2. Queries

- [x] 2.1 [backend] In `internal/reports/repository/queries/reports.sql`, do the following, as design.md §3 describes:
  - Add the category and status filters to `ListReports`, keeping the cursor condition in parentheses.
  - Add `ListReportPins :many`, which never selects `author_id` or `author_name`.
  - Add `UnknownCategories :many`.

  Run `go tool sqlc generate`. Verify that the three methods are generated and that a second run leaves `git status` unchanged.

## 3. Backend filters and map

- [x] 3.1 [backend] Add the filter fields to `ListParams`, plus `MapParams` and `ReportPin`, in `domain.go`. Add the filter fields to `ListReportsInput`, plus `MapInput` and `ReportPinDTO`, in `dto.go`. Add `ListPins` and `UnknownCategories` to the `Repository` interface. Implement them in `PgRepository`, and pass the filters through `PgRepository.List`. Verify with `go vet ./internal/reports/...`.
- [x] 3.2 [backend] Implement `checkFilters` (dedupe, at most 5 slugs, unknown slug → `ErrInvalidReport`, known status values). Use it from `Service.List`, and add `Service.Map` (limit 500). Verify with unit tests in `service_test.go` covering:
  - filters passed through to the repository;
  - duplicates dropped;
  - more than 5 slugs, an unknown slug and an unknown status, each rejected;
  - `UnknownCategories` not called without categories;
  - `Map` asking for 500 and mapping pins without the author.

  The existing `List`/`Search` tests must still pass.
- [x] 3.3 [backend] Implement `Handler.GetReportMap` and pass `category`/`status` in `Handler.ListReports`. Verify with `handler_test.go` cases through the module router:
  - `GET /reports/map` → 200 with pins, and the JSON has no `authorName`, `details` or author id.
  - `/reports/map` is not taken as a report id.
  - `?category=limpieza&category=alumbrado&status=publicado` reaches the service on both operations.
  - An unknown category → 400, and an invalid status → 400.
  - 401 without a token.

  `make test` must pass.
- [x] 3.4 [backend] Check against the local stack: run `make db-reset`, insert reports in several categories and statuses (through `POST /reports` plus an SQL `update` of the status), and check by hand with `curl` that:
  - `GET /reports?category=limpieza` returns only Limpieza.
  - `GET /reports?category=limpieza&status=publicado` combines both filters.
  - Walking the pages of a filtered feed with `limit=2` repeats and skips nothing.
  - `GET /reports/map?status=en_atencion` returns only those pins, with their coordinates.

  Record the commands in the PR description.
- [x] 3.5 [backend] Update the module list in `backend/AGENTS.md` (`reports` now also serves `GET /reports/map` and the feed filters). Verify that the text matches the contract.

## 4. Frontend client and shared pieces

- [x] 4.1 [frontend] Run `bun run generate:api` (after `make bundle`). Verify that `app/api/` exports `getReportMap` and `getReportMapOptions`, that `listReports` accepts `category` and `status`, and that `bun run typecheck` passes.
- [x] 4.2 [frontend] Move the pilot center and zoom from `ReportLocationMap.vue` to `utils/pilotZone.ts`, and import them there. Add `statusPinColor` to `utils/reportFormat.ts` (design.md §8). Verify with `bun run typecheck`, and check that the Reportar map still opens at the same place.
- [x] 4.3 [frontend] Add `stores/reportFilters.ts` (design.md §5). Verify with `bun run typecheck`. In the browser devtools, check that `queryParams` leaves out empty values and that `activeCount` counts categories plus the status.
- [x] 4.4 [frontend] Read the Figma frames for «Mapa» (mobile `5509:1594`, desktop `5517:1670`) and the filter chips (Inicio `5508:1549`), and follow them as design.md §6 and §8 describe.

## 5. Frontend filters on Inicio

- [x] 5.1 [frontend] Add `toggleCategory` and `setStatus` to `stores/reportFilters.ts` (design.md §5). Add `components/ReportFilterChips.vue` (design.md §6): «Todo», the «Estado ▾» menu with «Todos» first, and one toggle chip per category, applied at once, in a horizontally scrollable row. Verify in the browser that tapping chips changes the active styles, that «Todo» clears everything, and that the «Estado» chip names the chosen status.
- [x] 5.2 [frontend] Add the chips to `pages/index.vue`, pass `filters.queryParams` to the infinite query, and add the filters empty state with «Quitar filtros» (design.md §7). Verify in the browser against the local API:
  - Choosing Limpieza and Alumbrado shows only those cards.
  - Choosing a status shows only that status.
  - Scrolling loads more filtered pages.
  - A filter with no match shows the message, and «Quitar filtros» brings every card back.
  - Going back from a report keeps the filter, and a reload clears it.

## 6. Frontend Mapa

- [x] 6.1 [frontend] Add `components/ReportPinsMap.vue`: Leaflet with OSM tiles, one outline map-pin marker per pin colored by `statusPinColor`, a marker-click event that emits the id, a map-click event that closes the preview, and centering on a location prop or on the pilot zone. Verify in the browser that the points appear at the reports' locations, with their status colors.
- [x] 6.2 [frontend] Replace `pages/map.vue` (design.md §8) with:
  - the `getReportMap` query keyed by the filters;
  - the legend;
  - the bottom preview card, as a link to the report (photo thumbnail from `getReport` with a placeholder on error, title, category and status from the pin, chevron, no distance);
  - the round locate button («Usar mi ubicación» as its label) with `restore()` on mount;
  - the filter chips floating over the map;
  - the loading, error («Reintentar»), no-reports («Reportar») and no-match («Quitar filtros») overlays;
  - the mobile full-height and desktop panel layouts.

  Verify in the browser against the local API, on mobile and desktop widths:
  - Opening Mapa asks for nothing and centers on the pilot zone.
  - The locate button centers on the device, and declining shows the message.
  - Tapping a pin shows the preview, tapping the map closes it, and tapping the card opens the report.
  - The filters applied on Inicio are in effect on Mapa, and the reverse.
  - Stopping the API shows «Reintentar», which recovers when the API is back.

## 7. Frontend navigation

- [x] 7.1 [frontend] Turn Mapa into a normal link to `/map` in `AppBottomBar.vue` and `AppHeader.vue` (design.md §9), removing `showMapSoon`, «Pronto» and `aria-disabled`. Verify in the browser:
  - Tapping Mapa opens it and marks it as current on mobile and desktop.
  - Typing `/map` opens Mapa.
  - No «Próximamente» toast appears for Mapa.

  `bun run typecheck` must pass.

## 8. Integration

- [x] 8.1 [all] With the local stack, the API and `bun run dev` running, walk through every scenario in `specs/report-map`, `specs/report-filters`, `specs/report-feed` and `specs/app-navigation` with an anonymous session. Note any scenario that does not hold and fix it before marking this task done. Run `openspec validate add-report-map-and-filters --strict`.

## Workflow follow-up

- Archive the change with `/opsx:archive` after review and merge, syncing the specs (this removes the two Mapa «not available» requirements from `app-navigation`).
