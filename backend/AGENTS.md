# Punto Vecinal backend

Also read the root `AGENTS.md`: its rules (spec first, git, secrets, language) apply here.

## Applying OpenSpec changes from here

When you run `/opsx:apply` (or the `openspec-apply-change` skill) from `backend/`, do only the tasks prefixed `[backend]`. That includes the contract tasks (editing `openapi/` and running `make generate-api`). Leave `[frontend]` and `[all]` tasks unchecked and, when you finish, list the tasks that remain for other folders. `[db]` tasks (Supabase migrations) are also done from here, since sqlc depends on them. If a task has no prefix, ask which area it belongs to.

## Stack

- **Language:** Go. The API is our own: Supabase does not replace the backend.
- **Database:** Supabase Postgres with PostGIS (locations and distances) and pgvector (text similarity).
- **Authentication:** Supabase Auth. The API verifies the Supabase JWT on every request; RLS in the database is a second barrier, not the only one.
- **Neighbor without an account:** uses an anonymous Supabase session. When they create an account they keep the same user id, so their reports stay theirs.
- **Files:** DigitalOcean Spaces (S3-compatible API), served through signed URLs.
- **Embeddings:** Voyage AI. Only the report description is sent, never personal data.

## Code layout

The API is split into modules. Each module owns a slice of the contract and its own code:

```
openapi/                    at the repository root: shared by backend and frontend
  openapi.yaml              root contract: info, global security, tags, /healthz, $refs to module paths
  modules/<module>.yaml     paths (and module-only schemas) of one module
  schemas/common.yaml       responses and schemas shared by several modules
  config/<module>.yaml      oapi-codegen config of one module
  dist/openapi.yaml         bundle built by Redocly (generated, ignored by git)
backend/
  internal/
    <module>/
      api/generated.go      generated HTTP layer of the module (never edited)
      handler.go            implements the generated StrictServerInterface; only adapts types
      service.go            business logic of the module
    platform/               shared infrastructure
      auth/                 token verification (Verifier, Claims) and the contract-driven auth middleware
      httpserver/           router, CORS, request logging; mounts every module
      httperror/            shared 500 handler for errors returned by handlers
      config/, database/, logging/
```

Today the modules are `auth` (`GET /me`) and `system` (`GET /healthz`). Each module exposes `Register(r chi.Router, verifier auth.RequestVerifier)`, which `internal/platform/httpserver` calls to mount it.

## Contract

The OpenAPI contract lives in `openapi/` at the repository root (the frontend generates its client from it too) and is the source of truth for the API. Update it first, then the code. If code and contract disagree, the contract wins; flag the mismatch.

### How the contract is organized

- **Root file** (`openapi/openapi.yaml`): `info`, global `security`, the `tags` list, the `bearerAuth` security scheme, and the paths that are not in a module (`/healthz`). Module paths are referenced, not copied: `$ref: ./modules/auth.yaml#/paths/~1me`.
- **Module files** (`openapi/modules/<module>.yaml`): a `paths:` map with the module's operations, plus a `components:` section for schemas only that module uses (for example `Me` in `auth.yaml`, referenced locally as `#/components/schemas/Me`).
- **Shared schemas** (`openapi/schemas/common.yaml`): top-level keys that are responses or schemas used by more than one module (for example `Unauthorized`). Reference them from a module with `$ref: ../schemas/common.yaml#/Unauthorized`. A schema used by a single module stays in that module's file; move it to `common.yaml` only when a second module needs it.
- **Conventions for every operation:**
  - `operationId` in camelCase (`getMe`); it becomes the Go method name (`GetMe`).
  - Exactly one tag, equal to the module name in lowercase (`auth`, `system`). The tag is what assigns the operation to a module's generated code, so a missing or wrong tag drops the operation from every module.
  - `security` is inherited from the root (bearer token required). Write `security: []` only to make an operation public (today only `/healthz`).
- **Bundle:** [Redocly CLI](https://redocly.com/docs/cli) resolves every `$ref` and writes `openapi/dist/openapi.yaml` (`make bundle`). `dist/` is generated and ignored by git; never edit or commit it. The bundle is the only input of code generation.

### How authentication follows the contract

`internal/platform/auth` provides a middleware (built with `OapiRequestValidatorWithOptions`) that applies the contract's `security`: operations with `security: []` pass without a token, every other operation needs a valid Supabase bearer token, and the verified `Claims` (user id and anonymous flag) are put in the request context (`auth.ClaimsFrom`). A module's `Register` builds this middleware from its own embedded contract, so a new operation is protected by default. Handlers never check tokens themselves.

## Code generation

### HTTP layer

The HTTP layer (routes, handler interfaces, request and response types) is always generated from the contract with [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen). Never write it by hand and never edit generated files (`internal/<module>/api/generated.go`).

- Config: one file per module in `openapi/config/`. All of them generate models, a chi router, a strict server and the embedded spec, into package `api`, and use `output-options.include-tags: [<module>]` so each run only generates that module's operations (unused schemas are pruned).
- Output: `internal/<module>/api/generated.go`. These files are committed together with the contract change that produced them. The embedded contract inside each file also lists the other modules' paths, but empty (no operations); this is harmless.
- Command, from `backend/`:

  ```
  make generate-api
  ```

  It runs `make bundle` (Redocly, needs Node; the version is pinned in the `Makefile` and run through `npx`) and then oapi-codegen once per file in `openapi/config/`. oapi-codegen is pinned as a Go tool in `go.mod` (`go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest`), so everyone uses the same version. Running it twice must leave `git status` unchanged.

### Changing an endpoint

1. Edit the module's file in `openapi/modules/` (and `schemas/common.yaml` if a schema is shared).
2. `make generate-api`.
3. Adjust the module's `handler.go` to the regenerated `StrictServerInterface`; put the logic in `service.go`. Handlers only adapt generated types to the service.
4. Add or update the tests of the module.

### Adding a module

1. Create `openapi/modules/<module>.yaml` with the paths, and reference each path from `openapi/openapi.yaml` (plus a tag with the module name in its `tags` list).
2. Create `openapi/config/<module>.yaml`, copying an existing config and changing `output` (`internal/<module>/api/generated.go`) and `include-tags`.
3. `make generate-api`.
4. Create `internal/<module>/` with `handler.go`, `service.go` and `Register(r chi.Router, verifier auth.RequestVerifier) error` (copy `internal/auth/` as a reference), and mount it in `internal/platform/httpserver/routes.go`.
5. Add tests: handlers through the module's router with a fake verifier, including a 401 without a token for protected operations.

### Database access

Database access code is always generated with [sqlc](https://sqlc.dev) from the Supabase migrations (the schema) and hand-written SQL queries. Never hand-write row structs or query functions, and never edit generated files.

- Config: `backend/sqlc.yaml` (PostgreSQL, pgx/v5).
- Schema: `supabase/migrations/` at the repo root. sqlc reads the migrations directly, so there is no separate schema file to keep in sync.
- Queries: one `.sql` file per area in `backend/internal/platform/database/queries/`, each query annotated with `-- name: QueryName :one|:many|:exec`.
- Output: `backend/internal/platform/database/`, package `database`.
- Command, from `backend/`:

  ```
  go tool sqlc generate
  ```

  sqlc is pinned as a Go tool in `go.mod` (`go get -tool github.com/sqlc-dev/sqlc/cmd/sqlc@latest`).
- Flow for any schema change: add a new Supabase migration → write or adjust the queries → run the command → use the generated methods from the domain logic.
- Types sqlc cannot map on its own (for example PostGIS `geography` or pgvector `vector`) are mapped with `overrides` in `sqlc.yaml`, not by editing generated code.

## Conventions

- Generated files (`internal/<module>/api/generated.go`, sqlc output) are never edited by hand; change the contract or the SQL and regenerate.
- Schema changes go in versioned Supabase migrations; never edit a migration that has already been applied.
- Every user-dependent query filters by the user's id in addition to the RLS policies.
- Photos are processed on the server before storing them (for example, stripping EXIF metadata that carries the phone's location).

## Commands

Run everything from `backend/`. The `Makefile` wraps the commands; `make help` lists them all.

Setup:

1. `cp .env.example .env` and fill in the values. The API reads real environment variables first and `.env` as a fallback (viper, see `internal/platform/config`).
2. `make supabase-start` brings up the local Supabase stack (needs Docker). `make supabase-status` prints its URLs.

Daily use:

| Command | What it does |
| --- | --- |
| `make run` | Run the API (`go run ./cmd/punto-vecinal-api`). |
| `make watch` | Run with live reload (air, config in `.air.toml`). |
| `make build` | Build the binary (`./main`). |
| `make test` | Run all tests (`go test ./... -v`). |
| `make fmt` / `make vet` / `make tidy` | Format, vet, and tidy modules. |

Code generation (see [Code generation](#code-generation)):

| Command | What it does |
| --- | --- |
| `make bundle` | Bundle `../openapi/` into `../openapi/dist/openapi.yaml` with Redocly CLI. |
| `make generate-api` | Bundle the contract and regenerate the HTTP layer of every module. |
| `make generate-db` | Regenerate database access from the migrations and queries. |
| `make generate` | Both of the above. |

Database:

| Command | What it does |
| --- | --- |
| `make migration-new name=<name>` | Create a new Supabase migration. |
| `make db-reset` | Recreate the local DB from all migrations. |
| `make db-lint` | Lint the local DB schema. |
| `make supabase-stop` | Stop the local stack. |

Check the API is up with `curl -i localhost:8080/healthz` (200 when it can reach the database).
