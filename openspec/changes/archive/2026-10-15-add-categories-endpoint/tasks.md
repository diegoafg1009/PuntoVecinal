# Tasks

## 1. API contract

- [x] 1.1 [backend] Add `openapi/modules/categories.yaml` (`GET /categories`, `listCategories`, `Category` schema, 200 and 401), the `categories` tag and the path reference in `openapi/openapi.yaml`, and `openapi/config/categories.yaml`; verify `make bundle` (from `backend/`) succeeds and the bundle contains `/categories`.
- [x] 1.2 [backend] Run `make generate-api` from `backend/` and verify `internal/categories/api/generated.go` exists, the other modules' generated files are unchanged, `go build ./...` fails only on the missing `ListCategories` method, and running it twice leaves `git status` unchanged.

## 2. Database

- [x] 2.1 [db] Add a migration in `supabase/migrations/` (`make migration-new name=create_categories` from `backend/`) that creates `categories(slug, label, position)`, enables RLS with a read policy and inserts the five categories; verify `make db-reset` applies it and `select slug, label, position from categories order by position` returns the five rows in order.
- [x] 2.2 [db] Add `backend/internal/categories/repository/queries/categories.sql` (sqlc output in `repository/sqlc/`) with `ListCategories :many` ordered by `position` and run `go tool sqlc generate` from `backend/`; verify the generated `ListCategories` exists, `go build ./...` still fails only on the missing handler method, and `make db-lint` passes.

## 3. Backend module

- [x] 3.1 [backend] Create `internal/categories` (`service.go`, `handler.go`, `Register`) with a `Repository` interface defined in the service and implemented by `repository.PgRepository` over the generated queries, and mount it in `internal/platform/httpserver/routes.go`; verify `go build ./...` and `go vet ./...` pass.
- [x] 3.2 [backend] Add tests in `internal/categories` with a fake store and a fake verifier: 200 with the five items in order and `slug` and `label` on each, same answer on two calls, 401 without a token and with a bad one, 500 (no list) when the store fails; verify `go test ./internal/categories/...` passes. Extend `internal/platform/httpserver/routes_test.go` if it checks the mounted routes.
- [x] 3.3 [backend] Update `backend/AGENTS.md` (the modules list now includes `categories`) and `backend/README.md` if it lists endpoints; verify with `curl` against the local stack that `GET /categories` returns the five categories with an anonymous token and 401 without a token.

## 4. Frontend

- [x] 4.1 [frontend] Run `make bundle` in `backend/` and `bun run generate:api` in `frontend/`; verify `app/api/` contains `listCategories` and its query options, `bun run typecheck` passes, and running the generator twice changes nothing.
- [x] 4.2 [frontend] Add `app/composables/useCategories.ts` (generated query options, long `staleTime`, enabled only with a session); verify with the dev server that the call returns the five categories with the session token.
- [x] 4.3 [frontend] Update `ReportStepWhatHappens.vue` to render the categories from `useCategories`: skeleton while loading, error alert with «Reintentar», chips otherwise, and «Seguir» disabled while loading; verify in the browser each state (stop the API to see the error and retry once it is back).
- [x] 4.4 [frontend] Update `reportDraft.ts` so `category` is a plain string, clear a stored category that is not in the loaded list, and delete `app/utils/categories.ts`; verify `bun run typecheck` and `bun run build` pass, that no reference to the constant remains, and that a draft stored with an unknown category loses only the category.
- [x] 4.5 [frontend] Make step 1 accept only a category confirmed by the loaded list: «Seguir» stays on step 1 with «Vuelve a cargar las categorías para continuar.» when the list is unavailable, and the message clears when the categories arrive; verified in the browser with the API failing and a stored category.

## 5. Integration check

- [x] 5.1 [all] With local Supabase, backend and frontend running, walk through the spec scenarios of `report-categories` and the new `report-creation` requirements (list in order, 401 without token, loading, failure and retry, unknown stored category) and the existing step 1 scenarios still pass; record the result in the test notes of HU-02 (PV-11).
- [x] 5.2 [all] Run `openspec validate --all --strict`, `go test ./...` from `backend/`, and `bun run typecheck` and `bun run build` from `frontend/`; all pass.

## Workflow follow-up

- Archive the change (`/opsx:archive`) once review requirements are met.
