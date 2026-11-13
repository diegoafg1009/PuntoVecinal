# Proposal

## Why

The five report categories (Limpieza, Alumbrado, Pistas y veredas, Agua y desagüe, Otro) are fixed, but today they only exist as a constant in the frontend (`app/utils/categories.ts`). Reports will reference them (PV-51 creates the table, PV-75 receives a category in `POST /reports`), so the backend must own them and the frontend must read them instead of repeating them. One list in one place means the value the form sends is always one the API accepts.

## What Changes

- Add `GET /categories`, which returns the fixed categories in a fixed order, each with a stable `slug` and a Spanish `label`. It needs a valid session (anonymous or not), like every other operation.
- Add a `categories` table in the database, filled with the five categories by the migration itself (reference data that every environment needs), read-only for the app. A neighbor, moderator or case manager cannot add, rename or remove categories.
- Add the `categories` API module in the backend (contract, generated HTTP layer, handler, service, queries generated with sqlc).
- The Reportar form reads the categories from the API instead of the frontend constant, which is removed. Step 1 shows a loading state while they load and an error with a retry option if they cannot be loaded.
- A draft that holds a category the API no longer returns drops that category instead of keeping an invalid value.
- Step 1 only lets the neighbor continue with a category the API has confirmed, so a stored category cannot get past «Seguir» while the list is unavailable.

Out of scope: creating, editing or deleting categories (they are fixed, per the product rules), category icons or colors, the `reports` and `photos` tables and anything else in PV-51, filtering reports by category (HU-07), and any change to the zones.

## Capabilities

### New Capabilities
- `report-categories`: the fixed list of report categories, their stable identifiers and labels, and the API operation that exposes them.

### Modified Capabilities
- `report-creation`: step 1 gets its category choices from the API, with loading and error states, instead of a fixed list in the app.

## Impact

- `supabase/migrations/`: new migration creating and seeding `categories`; the later reports migration (PV-51) will reference it by slug.
- `backend/`: new module `internal/categories` (`handler.go`, `service.go`, `domain.go`, `dto.go` and a `repository/` layer with its sqlc queries and generated code), mounted in `internal/platform/httpserver/routes.go`; `sqlc.yaml` becomes one entry per module; `backend/AGENTS.md` lists the new module and documents the per-module database layout.
- `openapi/`: new `modules/categories.yaml`, `config/categories.yaml`, a path and a tag in `openapi.yaml`.
- `frontend/`: regenerated API client, a `useCategories` composable, changes to the step 1 component and the draft store; `app/utils/categories.ts` is removed.
- No new dependencies.
