# Proposal

## Why

The MVP left Mapa marked as not available, and Inicio can only show every report from newest to oldest. Two stories from Entrega 2 (epic EP-01, «Entrar y consultar reportes») close that gap:

- HU-06 (PV-15, «Ver los reportes en el mapa»): see the reports as points on a map, to quickly find what happens near home.
- HU-07 (PV-16, «Filtrar por categoría y estado»): filter the reports by category and status, in Inicio and in Mapa, to see only what matters to the neighbor.

## What Changes

- New `GET /reports/map` operation. It returns the published reports as map points: id, title, category, status and the location of the problem. It is not paginated and returns at most the 500 newest reports. It never returns the author or the session that published the report.
- `GET /reports` (feed) and `GET /reports/map` accept two optional filters:
  - `category`, which can be repeated, with one or more category slugs.
  - `status`, with one status.

  With no filter they behave as they do today. An unknown category answers 400. `GET /reports/search` stays as it is.
- Mapa becomes a real section at `/map`:
  - It shows a map with one pin per report, colored by status, and a legend of the colors.
  - Tapping a pin opens a preview card at the bottom with the photo, the title, the category and the status, which leads to the report page.
  - The map centers on the neighbor when they share their location. The location is asked for only from a round locate button, or reused silently when the browser already allows it, as Inicio does. Otherwise the map centers on the pilot zone.
- Mapa is no longer marked as not available in the mobile bar or the desktop header. It opens the section like any other link, and `/map` no longer redirects to Inicio.
- Inicio and Mapa get a row of filter chips, as the Figma designs show (a scrollable row above the feed, floating over the map). The neighbor can:
  - tap «Todo» to clear every filter;
  - tap category chips to turn one or more categories on or off;
  - open the «Estado» chip to choose one status, or «Todos».

  Chips apply at once, with no panel and no «Aplicar». The same filters apply to both sections and are kept when moving between them during the visit. When nothing matches, the section says so and offers «Quitar filtros».
- Assumptions, to be confirmed in review:
  - Filters live only in memory for the visit. A reload clears them, and they are not kept in the address.
  - Opening Mapa never asks for the location on its own.
  - The pilot zone center is the coordinate already used by the Reportar map.
- Out of scope:
  - filters in Buscar;
  - clustering of points (the grouped «4» bubble in the Figma mockup);
  - loading points by visible map area;
  - showing the "Derivado, sin respuesta" label;
  - distances on the map (the «120 m» in the Figma preview card is left out because map points carry no distance).

## Capabilities

### New Capabilities
- `report-map`: the reports as map points, covering both the `GET /reports/map` operation and the Mapa screen (points, colors, preview, centering).
- `report-filters`: filtering reports by category and status, covering the filter chips, the filters shared by Inicio and Mapa, and the empty state when nothing matches.

### Modified Capabilities
- `report-feed`: the list operation accepts the category and status filters.
- `app-navigation`: Mapa is no longer marked as not available and opens its section. The redirect from `/map` to Inicio is removed.

## Impact

- Contract: `openapi/modules/reports.yaml` changes in two ways:
  - It gains `GET /reports/map` (`getReportMap`) with the new `ReportMap` and `ReportPin` schemas.
  - `listReports` gains the `category` and `status` query parameters.

  `openapi/openapi.yaml` references the new path. This regenerates `backend/internal/reports/api/generated.go` and `frontend/app/api/`.
- Database: no migration. The feed queries gain the filters, and a new pins query is added in `backend/internal/reports/repository/queries/reports.sql`. sqlc is regenerated.
- Backend: filter validation and the map method in the `internal/reports` repository, service and handler, with tests.
- Frontend:
  - `pages/map.vue` (the Mapa screen);
  - a new map component for the points;
  - a filters store (Pinia) and a filter chips component;
  - `pages/index.vue` (filters and empty state);
  - `AppBottomBar.vue` and `AppHeader.vue` (Mapa as a normal link);
  - a shared constant for the pilot zone center.
- No new dependencies: the map uses the Leaflet already installed.
