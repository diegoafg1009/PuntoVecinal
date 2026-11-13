# Proposal

## Why

Neighbors can publish reports (PV-19, PV-20), but nobody can see them: Inicio is still a placeholder that shows the session id, and the report page opened after publishing says «Próximamente». HU-02 (PV-11, «Ver el detalle de un reporte») and HU-03 (PV-12, «Ver los reportes en un feed») are the two MVP stories that let a neighbor, with or without an account, see what problems are already known before reporting again.

## What Changes

- New `GET /reports` operation: published reports from newest to oldest, in cursor-based pages, each with its cover photo (signed link), title, category, status, place reference and publication date. When the caller sends their location, each item also carries its distance in meters. The pilot has a single zone, so no zone filter is applied.
- Inicio becomes the feed: a list of report cards (photo, title, category, status, distance when known, and how long ago it was published), loaded page by page as the neighbor scrolls, with loading, empty and error states. A «Usar mi ubicación» button lets the neighbor share their location to see distances; the app never asks for it on its own.
- Tapping a card opens the report detail.
- The report page (`/reports/{id}`) replaces its «Próximamente» placeholder with the detail: photos (with a counter when there is more than one), category, status, title, author («Vecino anónimo» when there is none), publication date, place reference, details and a map with the place. It handles a report that does not exist and a load failure.
- On mobile, the report page shows its own top bar (back button and «Reporte») instead of the app top bar, and keeps the bottom bar fixed while scrolling.
- Out of scope, though present in the Figma frames: category and status filters, «Ordenar», support and comment counters, «También me afecta», «Aportar», «¿Cómo está ahora?», case tracking, comments and the «⋯» menu. They belong to later stories.

## Capabilities

### New Capabilities
- `report-feed`: the list of published reports, both the paginated `GET /reports` operation and the Inicio screen that shows it as cards with continuous loading and optional distance.

### Modified Capabilities
- `report-detail`: adds the report page that shows a report to the neighbor (photos, data, author, map, not-found and error states); the API requirements stay as they are.
- `app-navigation`: the mobile top bar with brand and avatar is no longer shown on every section; the report page uses its own top bar with a back button.

## Impact

- Contract: `openapi/modules/reports.yaml` gains `GET /reports` (`listReports`) and the `ReportSummary` and `ReportPage` schemas; regenerated `backend/internal/reports/api/generated.go` and `frontend/app/api/`.
- Backend: new keyset query in `backend/internal/reports/repository/queries/reports.sql` (with optional PostGIS distance), repository, service and handler methods in `internal/reports`, and tests. The `reports` table already has `created_at` and `location`; one new migration adds an index on `(created_at desc, id desc)` for the feed order.
- Frontend: `pages/index.vue` (feed), `pages/reports/[id].vue` (detail), new report card, photo carousel and read-only map components, shared formatters for status labels, relative dates and distances, and a layout change so a page can replace the mobile top bar.
- No new dependencies expected: Nuxt UI (`UCarousel`, `UBadge`, `UCard`, `USkeleton`), Leaflet and TanStack Query infinite queries are already available.
