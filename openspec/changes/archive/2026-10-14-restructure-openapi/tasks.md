# Tasks

## 1. Contract tree

- [x] 1.1 Create `backend/openapi/openapi.yaml`, `modules/auth.yaml` and `schemas/common.yaml` with the current contract split as in design decision 1 (tags `system` and `auth` on every operation; `/healthz` stays in the root file). Verify the files parse and every `$ref` resolves in task 2.1.
- [x] 1.2 Add `backend/openapi/config/auth.yaml` and `config/system.yaml` (design decision 3). Verify both are valid oapi-codegen configs by running them in task 2.2.

## 2. Bundle and generation

- [x] 2.1 Add `make bundle` (pinned `npx @redocly/cli`, output `openapi/dist/openapi.yaml`) and ignore `backend/openapi/dist/` in `.gitignore`. Verify `make bundle` exits 0, and that the bundle has the same paths, operation ids, schemas, responses and security as the old `api/openapi.yaml` (compare with a normalized diff).
- [x] 2.2 Make `make generate-api` run `bundle` and then oapi-codegen once per file in `openapi/config/`. Verify it creates `internal/auth/api/generated.go` and `internal/system/api/generated.go`, and that each one exposes only its own operations.

## 3. Go modules

- [x] 3.1 Create `internal/auth/` (`handler.go`, `service.go`, `Register(r, verifier)`) with `GetMe`, and port the `/me` tests. Verify `go test ./internal/auth/...` passes with the same cases as before (200 anonymous, same id twice, 401 without or with a bad token).
- [x] 3.2 Create `internal/system/` with the `GetHealthz` handler and the readiness `pinger`, and port its test. Verify `/healthz` returns 200 without a token and 503 when the ping fails.
- [x] 3.3 Rename `Principal` to `Claims` (`Claims`, `WithClaims`, `ClaimsFrom`) across `internal/platform/auth`, handlers and tests. Verify `go build ./...` and the existing auth tests pass unchanged apart from the rename.
- [x] 3.4 Add `auth.NewAuthFunc(verifier)` (accepts only `bearerAuth`, verifies the token, sets `Claims` on the request) and wire `OapiRequestValidatorWithOptions` through each module's `Register` using the module's embedded spec, with `ErrorHandlerWithOpts` mapping security failures to `401 unauthorized` and missing routes to 404; delete the strict-handler `requireAuth`, `publicOperations` and `handleResponseError`'s auth case; reduce `internal/platform/httpserver` to router, CORS, logging and module mounting. Verify with tests: valid token reaches the handler with `Claims` in context; missing, malformed or invalid token returns 401 and the handler is not called; `/healthz` returns 200 without a token; an unknown path returns 404; a scheme other than `bearerAuth` is rejected.
- [x] 3.5 Delete `backend/api/` and `internal/rest/gen`. Verify `go build ./...`, `go vet ./...` and `go test ./...` pass and nothing imports the removed packages.

## 4. Documentation

- [x] 4.1 Update `backend/AGENTS.md` (Contract and Code generation sections, command table), `backend/README.md`, and the contract paths named in the `design` and `tasks` rules of `openspec/config.yaml`. Verify `grep -rn "api/openapi.yaml\|api/http\|rest/gen"` finds no stale reference outside archived changes.

## 5. Integration check

- [x] 5.1 From a clean checkout state (`openapi/dist/` removed) run `make generate` and confirm `git status` shows no changes in generated files. Then, with local Supabase and the API running, repeat the `guest-entry` live check: `/me` twice with the same id, after a token refresh, 401 without or with a bad token, and `/healthz` 200. Finally run `openspec validate restructure-openapi` and `go test ./...`.
