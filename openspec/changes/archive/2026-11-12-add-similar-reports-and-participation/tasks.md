# Tasks

## 1. Contract

- [x] 1.1 [backend] Update `openapi/modules/reports.yaml` as design.md §1 describes.
  - Add `findSimilarReports` (`POST /reports/similar`), `supportReport` and `unsupportReport` (`PUT`/`DELETE /reports/{id}/support`), and `listReportContributions` and `createReportContribution` (`GET`/`POST /reports/{id}/contributions`), with their responses.
  - Add the schemas `SimilarReportsRequest`, `SimilarReports`, `SimilarReport`, `SupportState`, `ReportSupport`, `Contribution`, `ContributionPage`, `CreateContributionRequest` and `CreatedContribution`.
  - Add `affectedCount`, `contributionCount` and `support` (required) to `Report`.
  - Reference the three new paths in `openapi/openapi.yaml`.

  Verify that `make bundle` (from `backend/`) succeeds and that the bundle has the five new operations.
- [x] 1.2 [backend] Run `make generate-api` from `backend/`. Verify that `StrictServerInterface` has `FindSimilarReports`, `SupportReport`, `UnsupportReport`, `ListReportContributions` and `CreateReportContribution`, and that `go build ./...` fails only because `Handler` does not implement them yet.

## 2. Database and queries

- [x] 2.1 [db] Add the migration from design.md §2 (`make migration-new name=add_similarity_and_participation`). Run `make db-reset` and `make db-lint`, and check that both pass.
- [x] 2.2 [backend] Add `github.com/pgvector/pgvector-go`. Add the `vector` override to both entries of `sqlc.yaml`, and register the pgvector types in `internal/platform/database` (`AfterConnect`). Verify with `go build ./...`.
- [x] 2.3 [backend] In `internal/reports/repository/queries/reports.sql`, change `GetReport` and `InsertReport`, add `affected_count` to `ListReports`, `SearchReportsRecent` and `SearchReportsNearest`, and add the queries listed in design.md §3. Run `go tool sqlc generate`. Verify that:
  - every method is generated with `pgvector.Vector` where needed;
  - no query that feeds a response selects `author_id`;
  - a second run leaves `git status` unchanged.

## 3. Backend

- [x] 3.1 [backend] Add `internal/platform/embedding` with the Voyage client (design.md §4). It has a configurable base URL for tests and returns `ErrUnavailable` for an empty key, transport errors and non-2xx answers. Wire `config.Voyage.APIKey` in `main.go`. Verify with unit tests against an `httptest` server: request body (model, `input_type`, one input), parsing of the 1024-dim answer, a 500 → `ErrUnavailable`, and an empty key → `ErrUnavailable` without a request.
- [x] 3.2 [backend] Update `domain.go`, `dto.go` and `PgRepository` with the new models and methods. `CreateContribution` runs the contribution insert and, unless the caller is the author, the support insert, in one transaction. Add `Embedder` and the new repository methods to the interfaces in `service.go`. Verify with `go vet ./internal/reports/...`.
- [x] 3.3 [backend] Embed on publish with a 3 s timeout, and insert `NULL` when that fails. Add `comparableText` and `RunEmbeddingBackfill`, and start the backfill from `main.go`. Verify with `service_test.go`:
  - the embedder receives only the title and details (an author name and a reference are present in the input but not in the text);
  - a failing embedder still creates the report with no embedding;
  - one backfill round embeds the missing reports and stops at the first `ErrUnavailable`.
- [x] 3.4 [backend] Implement `Service.Similar`, `Service.Support`/`Unsupport`, `Service.Contribute`/`Contributions`, and `Get` with the caller id (design.md §6–§8). Verify with `service_test.go` cases:
  - Similar: validation errors, and the embedder failing → `ErrComparisonUnavailable`.
  - Support: unknown report → `ErrNotFound`, author → `ErrIsAuthor`, and idempotent marking and unmarking.
  - Contribute: blank or too long text, a name that is too long, unknown report, the rate limit with `RetryAfter`, a photo that is cleaned and stored under `contributions/`, a photo removed when the insert fails, every aporte marking support for a non-author (count +1, and +0 when already supporting), and an aporte from the author not inserting support.
  - Contributions: cursor round trip, and an invalid cursor or limit.
  - `Get`: the `author`, `supporter` and `none` states.
- [x] 3.5 [backend] Implement the five handlers and update `GetReport` (handlers only map types and errors). `readContributionForm` rejects a second photo. Verify with `handler_test.go` through the module router:
  - `POST /reports/similar` → 200 with items and no `authorName`/`details`, and → 503 when the service reports it unavailable;
  - `/reports/similar` is not taken as a report id;
  - `PUT`/`DELETE` support → 200, 404 and 409;
  - `POST` contributions → 201 with text only and with a photo, 400 for two photos, 413, and 429 with `Retry-After`;
  - `GET` contributions → 200 with no author id in the JSON, and 404;
  - `GET /reports/{id}` carries `affectedCount`, `contributionCount` and `support`;
  - `GET /reports` and `GET /reports/search` items carry `affectedCount` (1 for a report nobody else supports), and the existing feed and search tests still pass;
  - 401 without a token on every new operation.

  `make test` must pass.
- [x] 3.6 [backend] Calibrate `maxSimilarDistance` with the real Voyage API (a local `.env` key; never commit it). Write about 10 pairs of Spanish report texts:
  - same problem in other words, for example «Poste sin luz» / «No hay alumbrado en el pasaje»;
  - different problems in the same category;
  - different categories.

  Print their cosine distances with a throwaway test or script outside the repo, then choose the threshold that keeps every "same problem" pair and drops the others. Record the value and the reasoning in the constant's comment, and the pairs in the PR description.
- [x] 3.7 [backend] Check against the local stack (`make db-reset`, with `make storage-start` and a Voyage key). Use `curl` to:
  - publish 2 reports in other words at about 80 m from each other, and one unrelated one;
  - check that `POST /reports/similar` returns only the related one, with its distance and `affectedCount` 1;
  - change one report's status to `cerrado` with SQL and see it disappear;
  - support with a second session → `affectedCount` 2, and the author → 409;
  - add a contribution with a photo whose file has GPS data from a third session, and check that `affectedCount` goes up by one; confirm with `exiftool` that the stored file has no GPS data;
  - list the contributions.

  Record the commands in the PR description.
- [x] 3.8 [backend] Update `backend/AGENTS.md`: the module list (the new `reports` operations), and that the Voyage key enables similar reports and the backfill job. Verify that the text matches the contract.

## 4. Frontend client and shared pieces

- [x] 4.1 [frontend] Run `bun run generate:api` (after `make bundle`). Verify that `app/api/` exports `findSimilarReports`, `supportReport`, `unsupportReport`, `listReportContributions` and `createReportContribution`, that `Report` has the three new fields and `ReportSummary` has `affectedCount`, and that `bun run typecheck` passes.
- [x] 4.2 [frontend] Extend `stores/reportDraft.ts` with `similar`, `similarOpen`, `joinTarget` (in memory, not persisted, cleared by `reset()`) and the `contributionText` getter (design.md §9). Verify with `bun run typecheck`. In the devtools, check that reloading the page never restores `joinTarget`.

## 5. Frontend Reportar

- [x] 5.1 [frontend] Read the Figma frames `07` (mobile `5550:1997`, PC `5553:2655`). Build `ReportSimilarDialog.vue` and make `ReportStepPhotoAndPlace.next()` run the check (design.md §9). Verify in the browser against the local stack:
  - with a similar report nearby the dialog opens with photo, title, category, distance, count and age;
  - without one, or with the API stopped, step 3 opens directly;
  - «No es ninguno, seguir» opens step 3 with the draft intact;
  - closing the dialog keeps step 2;
  - «Seguir» shows loading and a double tap sends one request;
  - the dialog is a bottom sheet at 390 px and centered on desktop.
- [x] 5.2 [frontend] Read the Figma frames `10` (mobile `5551:2019`, PC `5553:2807`) and `11` (`5551:2110`). Extract `ReportAuthorNameField.vue` from `ReportStepReview.vue`. Build `ReportCompactCard.vue`, `ReportJoin.vue` (with the `editable` flag) and `composables/useJoinReport.ts`. Render the join screen, not editable, from `pages/report.vue` (design.md §9). Verify in the browser:
  - step 3 still shows and validates the name switch as before;
  - the card and «Tu aporte» (text plus the first photo of a 2-photo draft);
  - the name switch rules;
  - «No es el mismo, volver» brings the dialog back with the same results;
  - joining opens the report with the toast «Te sumaste al reporte» / «Ahora a N vecinos les afecta. Tu aporte ya aparece abajo.», the support marked and the count +1, the aporte in the list, and Reportar empty afterwards;
  - an API error keeps the screen and the draft;
  - no new report appears in Inicio.

## 6. Frontend report page and cards

- [x] 6.0 [frontend] Read the Figma cards of Inicio (`5508:1539`) and Buscar (`5522:1935`). Show the affected count with the people icon in `ReportCard.vue` (design.md §10). Verify in the browser that:
  - the cards of Inicio and Buscar show the count, with 1 for a new report;
  - after marking support on a report and going back, the card shows the new count once the list refreshes;
  - the cards still fit at 390 px with the distance shown.

- [x] 6.1 [frontend] Read the Figma frame `04` (mobile `5509:1714`, PC `5517:1814`). Build `ReportSupportButton.vue` and add it with «Aportar» to `pages/reports/[id].vue` (design.md §10). Verify in the browser:
  - marking and unmarking moves the count by one and survives a reload;
  - on your own report the button is marked and disabled;
  - with the API stopped, it rolls back with a Spanish message;
  - quick taps end in the server's state;
  - the status badge never changes.
- [x] 6.2 [frontend] Move `pages/reports/[id].vue` to `pages/reports/[id]/index.vue`, and check that the report page behaves exactly as before. Build `pages/reports/[id]/join.vue`, which renders `ReportJoin` with `editable` and is opened by «Aportar» (design.md §10). Build `ReportContributions.vue` (the list) and add it to the report page. Verify in the browser:
  - «Aportar» opens «Sumarte a este reporte» with the report card, «a N les afecta», no distance unless the location was already shared, and an empty «Tu aporte»;
  - «No es el mismo, volver» and the back button return to the report without sending;
  - a text-only aporte, and a text + photo aporte, return to the report with «Te sumaste al reporte» / «Ahora a N vecinos les afecta. Tu aporte ya aparece abajo.»; they appear in the list with «Vecino anónimo · hace un momento», and «También me afecta» is marked with the counts updated;
  - aportar again on a report you already support leaves the affected count as it was;
  - aportar on your own report adds the aporte and leaves the count as it was;
  - a named aporte shows the name;
  - empty text and a switch on with a blank name are blocked with messages;
  - a photo can be replaced;
  - more than 20 aportes → «Ver más» loads the rest;
  - an error keeps the screen content;
  - the empty and error states of the list work.

  `bun run typecheck` must pass.

## 7. Integration

- [x] 7.1 [all] Run the whole flow at 390 px and on desktop against the local stack: publish a report, then draft the same problem in other words at the same place and join it from the dialog; mark and unmark support from a second browser session; from a third session, add an aporte with a photo through «Aportar». Check every scenario of the change's specs that the earlier tasks did not cover, and run `openspec validate add-similar-reports-and-participation --strict`.
