# Punto Vecinal backend

Also read the root `AGENTS.md`: its rules (spec first, git, secrets, language) apply here.

## Stack

- **Language:** Go. The API is our own: Supabase does not replace the backend.
- **Database:** Supabase Postgres with PostGIS (locations and distances) and pgvector (text similarity).
- **Authentication:** Supabase Auth. The API verifies the Supabase JWT on every request; RLS in the database is a second barrier, not the only one.
- **Neighbor without an account:** uses an anonymous Supabase session. When they create an account they keep the same user id, so their reports stay theirs.
- **Files:** DigitalOcean Spaces (S3-compatible API), served through signed URLs.
- **Embeddings:** Voyage AI. Only the report description is sent, never personal data.

## Contract

- The OpenAPI contract lives in `backend/api/openapi.yaml` and is the source of truth for the API.
- Update the contract first, then the code. If code and contract disagree, the contract wins; flag the mismatch.

## Code generation

### HTTP layer

The HTTP layer (routes, handler interfaces, request and response types) is always generated from the contract with [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen). Never write it by hand and never edit generated files (`*.gen.go`).

- Config: `backend/api/http/oapi-codegen.yaml` (chi router, strict server, models, embedded spec).
- Output: `internal/server/gen/api.gen.go`, package `api`.
- Command, from `backend/`:

  ```
  go tool oapi-codegen -config api/http/oapi-codegen.yaml api/openapi.yaml
  ```

  oapi-codegen is pinned as a Go tool in `go.mod` (`go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest`), so everyone uses the same version.
- Flow for any API change: update `api/openapi.yaml` → run the command → implement or adjust the methods of the generated `StrictServerInterface`. Handlers only adapt the generated types to the domain logic; business rules live outside the handlers.
- Generated code is committed together with the contract change that produced it.

### Database access

Database access code is always generated with [sqlc](https://sqlc.dev) from the Supabase migrations (the schema) and hand-written SQL queries. Never hand-write row structs or query functions, and never edit generated files.

- Config: `backend/sqlc.yaml` (PostgreSQL, pgx/v5).
- Schema: `supabase/migrations/` at the repo root. sqlc reads the migrations directly, so there is no separate schema file to keep in sync.
- Queries: one `.sql` file per area in `backend/internal/db/queries/`, each query annotated with `-- name: QueryName :one|:many|:exec`.
- Output: `backend/internal/db/`, package `db`.
- Command, from `backend/`:

  ```
  go tool sqlc generate
  ```

  sqlc is pinned as a Go tool in `go.mod` (`go get -tool github.com/sqlc-dev/sqlc/cmd/sqlc@latest`).
- Flow for any schema change: add a new Supabase migration → write or adjust the queries → run the command → use the generated methods from the domain logic.
- Types sqlc cannot map on its own (for example PostGIS `geography` or pgvector `vector`) are mapped with `overrides` in `sqlc.yaml`, not by editing generated code.

## Conventions

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
| `make generate-api` | Regenerate the HTTP layer from `api/openapi.yaml`. |
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
