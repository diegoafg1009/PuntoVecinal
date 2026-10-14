# Tasks

## 1. Generator

- [x] 1.1 Add `@hey-api/openapi-ts` and `typescript@^6` as dev dependencies, `openapi-ts.config.ts`, and the `generate:api` script. Verify `make bundle` then `bun run generate:api` writes `app/api/` with `getMe`, `getHealthz`, the `Me` type and the query helpers.

## 2. Wiring

- [x] 2.1 Add `app/plugins/api.ts` (base URL and bearer interceptor). Verify with a script that signs in anonymously and calls `getMe()` through the generated client: it returns the anonymous user.
- [x] 2.2 Switch `useMe` to `getMeOptions()` keeping the user id in the query key, and drop the local `Me` interface. Verify `/` still shows the user id in the browser.

## 3. Docs

- [x] 3.1 Document the API client and `bun run generate:api` in `frontend/AGENTS.md`.
