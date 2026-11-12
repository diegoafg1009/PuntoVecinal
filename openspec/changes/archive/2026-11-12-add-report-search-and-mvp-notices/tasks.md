# Tasks

## 1. Contract

- [x] 1.1 [backend] Add `searchReports` (`GET /reports/search`) to `openapi/modules/reports.yaml`, as design.md describes. Its query parameters are `q` (max 100), `sort` (`recent` | `distance`, default `recent`), `latitude`, `longitude`, `cursor` and `limit` (1–50, default 20). It answers `200` with `ReportPage`, `400` (text) and the common `401`. Reference the path in `openapi/openapi.yaml`. Verify that `make bundle` (from `backend/`) succeeds and that the bundle has `/reports/search` alongside `/reports/{id}`.
- [x] 1.2 [backend] Run `make generate-api` from `backend/`. Verify that `internal/reports/api/generated.go` has `SearchReports` in `StrictServerInterface` and that `go build ./...` fails only because `Handler` does not implement it yet.

## 2. Database

- [x] 2.1 [db] Add a migration (`make migration-new name=enable_unaccent`) with `create extension if not exists unaccent with schema extensions;`. Verify that `make db-reset` and `make db-lint` pass and that `select extensions.unaccent('agüa')` returns `agua`.
- [x] 2.2 [backend] Add the `SearchReportsRecent :many` and `SearchReportsNearest :many` queries to `internal/reports/repository/queries/reports.sql`, as design.md describes: the every-word `unaccent ILIKE` filter over a `text[]` and the feed's projection, never selecting `author_id` or `author_name`. Recent uses keyset `(created_at, id)`. Nearest uses keyset `(exact distance, created_at, id)`, returns the exact distance for the cursor, and returns rounded meters. Run `go tool sqlc generate` and verify that both methods are generated and that a second run leaves `git status` unchanged.

## 3. Backend search

- [x] 3.1 [backend] Add `SearchParams` and the search order to `domain.go`, and `SearchReportsInput` to `dto.go`. Add `Search` to the `Repository` interface and implement `PgRepository.Search`, which picks the query by order and maps rows to the existing summary type plus the exact distance. Verify with `go vet ./internal/reports/...`.
- [x] 3.2 [backend] Implement `Service.Search`, which does the following:
  - Splits `q` into words, escaping `\`, `%` and `_`, and rejects a `q` over 100 characters.
  - Rejects a search with no words when `sort=recent`, and `sort=distance` without a location.
  - Validates the limit and the coordinates as `List` does.
  - Encodes and decodes the order-tagged search cursor.
  - Fetches `limit+1` rows and signs the covers through a helper shared with `List`.

  Verify with unit tests in `service_test.go` covering the following:
  - Word splitting and escaping (`100%`, `a_b`, extra spaces).
  - Every rejection case.
  - Cursor round trip for both orders, and a cursor rejected when used with the other order or when it is a feed cursor.
  - `nextCursor` only when there are more rows.
  - A distance tie across a page boundary is not skipped.
  - The existing `List` tests still pass.
- [x] 3.3 [backend] Implement `Handler.SearchReports`, mapping params and DTOs to `api.ReportPage` and invalid input to 400. Verify with `handler_test.go` cases through the module router:
  - 200 with results in order.
  - `/reports/search` is not taken as a report id.
  - 400 for a missing `q` with `sort=recent`, `sort=distance` without a location, a bad cursor, and a 101-character `q`.
  - 401 without a token.
  - The JSON has no `authorName`, `details` or author id.

  `make test` must pass.
- [x] 3.4 [backend] Add a database-backed check of the queries against the local stack: `make db-reset`, then insert reports through `POST /reports` or SQL. Verify by hand with `curl` that «FUGA agüa» finds «Fuga de agua», «basura parque» excludes «Basura en la avenida», and `sort=distance` orders by distance. Record the commands in the PR description.
- [x] 3.5 [backend] Update the module list in `backend/AGENTS.md` (`reports` now also serves `GET /reports/search`) and verify that the text matches the contract.

## 4. Frontend client and shared pieces

- [x] 4.1 [frontend] Run `bun run generate:api` (after `make bundle`). Verify that `app/api/` exports `searchReports` and `searchReportsInfiniteOptions`, and that `bun run typecheck` passes.
- [x] 4.2 [frontend] Add `isAnonymous` to `stores/session.ts` (`session.user.is_anonymous`). Verify with `bun run typecheck` and in the browser devtools that a guest session reports `true`.

## 5. Frontend Buscar

- [x] 5.1 [frontend] Ask the user for the Figma «Buscar» frame link (daily call limit) and use it if there is one. Otherwise follow the layout in design.md.
- [x] 5.2 [frontend] Replace `pages/search.vue`, as design.md describes:
  - A search form (`UInput`, `maxlength=100`, submit on Enter or «Buscar») and a «Cerca de mí» toggle.
  - State in the route query (`q`, `near=1`) via `router.replace`.
  - An infinite query on `searchReportsInfiniteOptions`, keyed by `q`, `near` and the location, and enabled only with `q` or `near`.
  - `restore()` on mount. `share()` when the toggle turns on, and the toggle turns back off if that fails.
  - `ReportCard` results with the end-of-list sentinel.
  - Skeletons, «No encontramos reportes para «{q}».», «No hay reportes cerca de ti.», and an error state with «Reintentar».
  - The same stale and gc times as Inicio.

  Verify in the browser against the local API:
  - Nothing is requested on opening.
  - «poste» shows its matches.
  - «xyz» shows the empty message.
  - «Cerca de mí» prompts for the location, then orders nearest first with distances.
  - An empty search with «Cerca de mí» on lists everything nearest first.
  - Declining keeps the toggle off and shows the toast.
  - Back from a result restores the search.
  - Reloading `/search?q=poste&near=1` restores it.
  - Stopping the API shows the error with a working «Reintentar».

## 6. Frontend navigation, Avisos and Reportar

- [x] 6.1 [frontend] Make Mapa not available in `AppBottomBar.vue` and `AppHeader.vue`: a muted button with `aria-disabled="true"` and a «Pronto» mark, which shows the «Próximamente» toast and does not navigate. Verify on mobile and desktop widths that tapping Mapa on Buscar with results keeps the same screen, results and current-section mark.
- [x] 6.2 [frontend] Make `pages/map.vue` redirect to `/` with `replace`. Verify that opening `/map` directly, both on a fresh load and from a link inside the app, lands on Inicio.
- [x] 6.3 [frontend] Replace `pages/notices.vue` with the account-needed message for `isAnonymous` sessions, with no buttons, and keep `ComingSoon` otherwise. Ask for the Figma «Avisos» frame link first and use it if there is one. Verify in the browser as a guest that the message is shown with no button and that moving to Inicio, Buscar and Reportar from Avisos works on mobile and desktop.
- [x] 6.4 [frontend] Add `components/ReportEmergencyNotice.vue`: an alert that cannot be closed, with Serenazgo named and `tel:` links for 105 Policía, 116 Bomberos and 106 SAMU. Render it in `pages/report.vue` above the step content and remove the old alert from `ReportStepWhatHappens.vue`. Verify in the browser that the notice is at the top of steps 1, 2 and 3 with no close button, that each number is an `<a href="tel:…">`, and that tapping one on a phone (or the mobile emulator) offers the call.

## 7. Integration

- [x] 7.1 [all] With the local stack running (`make supabase-start`, `make storage-start`, `make run`, `bun run dev`), enter as a guest and do the following:
  - Publish a report titled «Fuga de agua en el pasaje».
  - Find it from Buscar with «fuga agua» and open it.
  - Go back to the same results.
  - Tap Mapa and stay there.
  - Open Avisos and see the account message.
  - Check that the network tab shows no author name or user id in `GET /reports/search`.

  Then run `openspec validate add-report-search-and-mvp-notices --strict` and verify that it passes.
