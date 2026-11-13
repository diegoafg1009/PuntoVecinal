# Tasks

## 1. Contract

- [x] 1.1 [backend] Add `listReports` (`GET /reports`) to `openapi/modules/reports.yaml` with the `cursor`, `limit` (1–50, default 20), `latitude` and `longitude` query parameters, the `200` `ReportPage`, `400` (text) and common `401` responses, and the `ReportSummary` and `ReportPage` schemas as in design.md; update the `reports` tag description in `openapi/openapi.yaml`. Verify `make bundle` (from `backend/`) succeeds and the bundle lists both `get` and `post` under `/reports`.
- [x] 1.2 [backend] Run `make generate-api` from `backend/` and verify `internal/reports/api/generated.go` has `ListReports` in `StrictServerInterface` and `go build ./...` fails only because `Handler` does not implement it yet; running it twice leaves `git status` unchanged.

## 2. Database

- [x] 2.1 [db] Add a migration (`make migration-new name=add_reports_feed_index`) creating `reports_created_at_id_idx` on `public.reports (created_at desc, id desc)`; verify `make db-reset` and `make db-lint` pass.
- [x] 2.2 [backend] Add the `ListReports :many` query to `internal/reports/repository/queries/reports.sql` (keyset on `(created_at, id)`, cover photo at position 1, optional distance in whole meters, never selecting `author_id` nor `author_name`) and run `go tool sqlc generate`; verify the generated `ListReports` method and row type exist and a second run leaves `git status` unchanged.

## 3. Backend listing

- [x] 3.1 [backend] Add `ReportSummary`, `ListParams` and `ErrInvalidCursor` to `internal/reports/domain.go`, `ListReportsInput`, `ReportSummaryDTO` and `ReportPageDTO` to `dto.go`, and `List` to the `Repository` interface in `service.go`; implement `PgRepository.List` mapping the sqlc rows. Verify with `go build ./...` (handler still pending) and `go vet ./internal/reports/...`.
- [x] 3.2 [backend] Implement cursor encoding/decoding (base64url of `created_at|id`, full precision) and `Service.List`: default and range of `limit`, both-or-neither coordinates in range, fetch `limit+1`, sign each cover with `photoLinkTTL`, build `NextCursor`. Verify with unit tests in `service_test.go` (fake repository and store) covering: default size 20, `nextCursor` only when there are more rows, cursor round trip, invalid cursor and limits (0, 51) and lone/out-of-range coordinate return errors, distance passed through only with a location, a signing failure returns an error.
- [x] 3.3 [backend] Implement `Handler.ListReports` mapping params and DTOs to `api.ReportPage` (null `distanceMeters` and `reference` when absent; `nextCursor` null on the last page) and invalid input to 400. Verify with `handler_test.go` cases through the module router with the fake verifier: 200 with items in order, 400 for a bad cursor, `limit=0` and only `latitude`, 401 without a token, and that the JSON has no `authorName`, `details` nor author id; `make test` passes.
- [x] 3.4 [backend] Update the module list in `backend/AGENTS.md` (`reports` now serves `GET /reports` too) and verify the text matches the contract.

## 4. Frontend client and shared pieces

- [x] 4.1 [frontend] Run `bun run generate:api` (after `make bundle`) and verify `app/api/` exports `listReports`, the `ReportPage`/`ReportSummary` types and the TanStack helpers for it; `bun run typecheck` passes.
- [x] 4.2 [frontend] Add `app/utils/reportFormat.ts` with `statusLabel`, `statusColor`, `publishedAgo` and `formatDistance` as specified in design.md; verify with `bun run typecheck` and by checking in the browser console (or a temporary page) that 123 → «120 m», 1234 → «1,2 km», 2 h ago → «hace 2 h», yesterday → «ayer», 10 days ago → «12 nov 2026»-style date.
- [x] 4.3 [frontend] Add the `readonly` prop to `ReportLocationMap.vue` (no drag, no click to move, no wheel zoom, centered at `PLACED_ZOOM`); verify step 2 of Reportar still lets the pin be moved and a read-only instance cannot be changed.
- [x] 4.4 [frontend] Support `ownTopBarOnMobile` page meta in `layouts/default.vue` (hide `AppTopBar`, keep `AppBottomBar` and its padding); verify Inicio still shows the app top bar and Reportar still hides both bars.

## 5. Frontend Inicio feed

- [x] 5.1 [frontend] Add `useNeighborLocation` (geolocation only from a user action, 4-decimal rounding, `localStorage` flag in try/catch, silent reuse when the Permissions API reports `granted`, flag cleared on decline/failure); verify in the browser that opening Inicio never prompts, tapping «Usar mi ubicación» prompts, and a reload with permission granted restores distances without a prompt.
- [x] 5.2 [frontend] Add `ReportCard.vue` (link to `/reports/{id}`, category and status badges, `publishedAgo`, photo with placeholder on error, title, «distance · reference» line shown only with content); verify with `bun run typecheck` and against the Figma frame «01 · Inicio» (`5508:1539`) on mobile and «01 · Inicio · PC» (`5515:1539`).
- [x] 5.3 [frontend] Replace `pages/index.vue` with the feed: infinite query keyed by the rounded location, sentinel with `useIntersectionObserver`, skeletons, empty state with «Reportar», first-page error and next-page error with «Reintentar», header «Más recientes» / «Cerca de ti · más recientes» with «Usar mi ubicación»; stale time 5 min and gc time under 1 h. Verify in the browser against the local API with 25+ reports: newest first, the second page loads on scrolling without reload and stops at the end, sharing the location reloads with distances, a card opens its report, and stopping the API shows the error with a working «Reintentar».

## 6. Frontend report page

- [x] 6.1 [frontend] Replace `pages/reports/[id].vue`: `getReportOptions` query, skeleton, 404 state with «Ir a Inicio», error state with «Reintentar», `UCarousel` of photos with «n / total» only for 2–3 photos and placeholder on error, badges, title, meta line with «Vecino anónimo» fallback, details only when present, read-only map; content centered at 560 px on desktop. Verify in the browser against Figma «04 · Detalle del reporte» (`5509:1714`) and PC (`5517:1814`): anonymous and named reports, with and without details/reference, 1 and 3 photos, an unknown id, and the API stopped.
- [x] 6.2 [frontend] Add the page's mobile top bar (`ownTopBarOnMobile`, back button with `aria-label="Volver"` and «Reporte», `router.back()` when there is in-app history, else Inicio); verify on a mobile viewport that back from a card returns to Inicio, back on a directly opened link goes to Inicio, the bottom bar stays fixed while scrolling, and publishing a report still opens this page with the «Reporte publicado» toast.

## 7. Integration

- [x] 7.1 [all] With the local stack (`make supabase-start`, `make storage-start`, `make run`, `bun run dev`), publish a report from Reportar as a guest, see it first in Inicio with «hace un momento», open it from its card, and check the network tab shows no author name or user id in `GET /reports`; then run `openspec validate add-report-feed-and-detail` and verify it passes.

## Workflow follow-up

- Archive the change with `/opsx:archive` after review, and close PV-50/PV-55 (spec) in Jira when the artifacts are approved.
