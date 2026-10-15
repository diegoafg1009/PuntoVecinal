# Design

## Context

See proposal.md. Current state:

- Backend: Go + chi with one module per slice of the contract (`auth`, `system`), each with `api/generated.go` (oapi-codegen strict server), `handler.go` and `service.go`; `routes.go` mounts them. `system` already receives the database pool (`s.db`). sqlc is installed but not used yet: `backend/sqlc.yaml` still points at one shared `internal/platform/database/queries/` folder and output package `database`, and there are no migrations and no queries. This change is the first to use it and moves the configuration to one entry per module (queries and generated code inside `internal/<module>/repository/`, package `sqlc`), as `backend/AGENTS.md` now documents; `internal/platform/database/` keeps only the connection pool.
- Contract: `openapi/openapi.yaml` references each module's paths; security is global bearer, inherited unless `security: []`.
- Frontend: the generated client in `app/api/` (Hey API, with TanStack Query options); the category list is a constant in `app/utils/categories.ts`, used by `ReportStepWhatHappens.vue` and by `reportDraft.ts` (the `CategorySlug` type and `isCategorySlug`).

## Goals / Non-Goals

**Goals:**
- One source of truth for the categories, in the database, that later tables can reference.
- A small, cacheable read endpoint and a form that depends on it without breaking when it is slow or down.

**Non-Goals:**
- Managing categories from the app, `reports`/`photos` tables, icons or colors.

## Decisions

### 1. Table `categories`, seeded in the migration
`categories(slug text primary key, label text not null, position smallint not null unique)`, with the five rows inserted by the same migration. The categories are reference data every environment needs (the form and later foreign keys depend on them), so they go in the migration, not in `supabase/seed.sql`, which only runs on a local `db reset`. The slug is the primary key because it is the stable identifier the frontend and `POST /reports` already use (`limpieza`, `alumbrado`, `pistas-y-veredas`, `agua-y-desague`, `otro`); PV-51 can reference `categories(slug)` without a join to fetch it. *Alternative:* a uuid id plus a unique slug. Rejected: a second identifier nobody needs for five fixed rows. *Alternative:* a Postgres enum. Rejected: labels and order would still need a table, and changing an enum is awkward.

### 2. RLS on, read-only
RLS is enabled with one `select` policy for the `anon` and `authenticated` roles and no write policies, as a second barrier behind the API (per `backend/AGENTS.md`). The API reads through its own connection, so the policy does not gate it; writes are impossible from the app roles.

### 3. Contract
New `openapi/modules/categories.yaml` with `GET /categories`, `operationId: listCategories`, tag `categories`, security inherited (bearer required). Response 200: a JSON array of `Category { slug: string, label: string }` (both required), ordered by position; 401 uses the shared `Unauthorized`. `openapi/openapi.yaml` gets the `categories` tag and `$ref: ./modules/categories.yaml#/paths/~1categories`; `openapi/config/categories.yaml` is a copy of `auth.yaml` with `output: internal/categories/api/generated.go` and `include-tags: [categories]`. The `Category` schema stays in the module (only it uses it), as the backend rules say. A bare array keeps the client simple; if pagination or metadata is ever needed it would be a new operation or version, which is acceptable for a five-row fixed list.

### 4. Backend module
`internal/categories/` with `handler.go` (adapts the generated types), `domain.go` (the `Category` model), `dto.go` (`CategoryDTO`), `service.go` (`List(ctx)` returning DTOs, not entities) and a `Repository` interface defined in `service.go` (the consumer) and implemented by `repository.PgRepository` (`internal/categories/repository/`: hand-written `repository.go` over the sqlc output in `repository/sqlc/`, whose `ListCategories` query is in `repository/queries/` and orders by `position`), so handler tests use a fake store, as `system` does with its pinger. `Register(r, verifier, store)` receives the repository built from the pool and is mounted in `routes.go`. A store error is returned from the handler, so the shared `httperror` handler answers 500 (spec: never a partial list). No caching in the API; the list is five rows and the client caches it.

### 5. Frontend
- `app/composables/useCategories.ts` wraps the generated `listCategoriesOptions()` with a long `staleTime` (the list never changes during a session) and the session guard the other composables use.
- `ReportStepWhatHappens.vue` renders skeleton chips while loading, an error `UAlert` with «Reintentar» (calls `refetch`) on failure, and the chips otherwise. «Seguir» is disabled while loading. The existing required-category check is not enough on its own: with the API down, a draft restored with a stored category would pass it. So the step also requires the chosen category to be among the loaded ones; when it is not, «Seguir» stays on step 1 with «Vuelve a cargar las categorías para continuar.», and that message is cleared as soon as the categories arrive.
- `reportDraft.ts` stops importing the constant: `category` becomes `string | null`, and the stored value is read as any string. When the categories load, the step (or the store through a small `pruneCategory(validSlugs)` action) clears a category that is not in the list. `app/utils/categories.ts` and `isCategorySlug` are deleted.
- The Pinia rule holds: the category list is server state in TanStack Query, not copied into a store.

### 6. Slugs stay as they are
The slug values do not change from the frontend constant, so existing stored drafts keep working and no migration of `localStorage` data is needed.

## Risks / Trade-offs

- [The form now depends on the API to show categories] → loading and retry states (spec); the list is fetched once per session and cached.
- [PV-51 may want different column names or a uuid key] → this change fixes the slug as the key and documents it; PV-51 owns the reports migration and can adapt it, since nothing else references `categories` yet.
- [First use of sqlc: toolchain issues surface here] → the `[db]` tasks run `go tool sqlc generate` before any code that uses it and verify the output builds.
- [A stored draft category can disappear] → handled by clearing it (spec).
