# Proposal

## Why

The frontend calls the API with hand-written `$fetch` code and a hand-written `Me` type (`useMe.ts`). Every new endpoint would repeat that work, and a change in the contract (`openapi/`) would not break the frontend at build time. The team wants the frontend to be generated from the same contract the backend uses.

## What Changes

- Add [Hey API](https://heyapi.dev) (`@hey-api/openapi-ts`) as a dev dependency. `bun run generate:api` reads `openapi/dist/openapi.yaml` and writes `frontend/app/api/`: types, an SDK with one function per `operationId`, the fetch client, and TanStack Query helpers for Vue. The output is committed.
- Add `openapi-ts.config.ts` with the plugins `@hey-api/typescript`, `@hey-api/client-fetch`, `@hey-api/sdk` and `@tanstack/vue-query`.
- Add `app/plugins/api.ts`: sets the client's base URL from `apiBaseUrl` and adds `Authorization: Bearer <access token>` to every request.
- `useMe` uses the generated `getMeOptions()` instead of `$fetch` and a local `Me` interface.
- Pin `typescript@^6` as a dev dependency: the generator needs the TypeScript JS API, which the version 7 pulled in by Nuxt does not provide.
- Document the flow in `frontend/AGENTS.md`.

No endpoint, schema or user-visible behavior changes. Out of scope: Zod schemas, mock generation, a CI check that generated files are up to date, a `typecheck` script.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. This is a tooling change; `specs` is skipped (`skip_specs: true`).

## Impact

- `frontend/`: `package.json`, `bun.lock`, `openapi-ts.config.ts`, `app/api/` (generated), `app/plugins/api.ts`, `app/composables/useMe.ts`, `AGENTS.md`.
- Changing the contract now also means running `make generate-api` (backend) and `bun run generate:api` (frontend).
