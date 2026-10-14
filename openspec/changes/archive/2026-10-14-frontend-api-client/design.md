# Design

## Context

See proposal.md. The contract is bundled by Redocly into `openapi/dist/openapi.yaml` (ignored by git). The frontend is an SPA (`ssr: false`) that keeps the Supabase session in a Pinia store and uses TanStack Query for server state.

## Goals / Non-Goals

**Goals:**
- One source of truth: the frontend types and calls come from the contract.
- Same runtime behavior as before: `GET /me` with the bearer token.

**Non-Goals:**
- Runtime validation (Zod), mocks, CI drift checks.

## Decisions

### 1. Hey API with the TanStack Query plugin

Chosen over `openapi-typescript` + `openapi-fetch` (types only, hooks written by hand) and Orval (heavier output and configuration). It generates `queryOptions` helpers that plug into `useQuery`, so composables stay short while the query keys and request code come from the contract.

### 2. Generated code is committed, in `app/api/`

Same rule as the backend's `generated.go`: it is committed together with the contract change and never edited by hand. The generator input is the bundle, so `make bundle` must run first.

### 3. Auth through a client interceptor in a Nuxt plugin

`app/plugins/api.ts` configures the shared generated client once: `baseUrl` from runtime config and a request interceptor that reads the access token from the session store at request time (so refreshed tokens are used). Composables do not pass headers.

### 4. `useMe` keeps the user id in the query key

`getMeOptions()` is spread and its key extended with `session.userId`, so a different session never reuses a cached answer (same behavior as before).

### 5. `typescript@^6` pinned

`@hey-api/openapi-ts` imports the TypeScript JS API; TypeScript 7 (native port) does not ship it. Revisit when the generator supports 7.

## Risks / Trade-offs

- Generated code adds noise to diffs; mitigated by keeping it in one folder marked as generated.
- The generator's output can change between versions; the dependency is a caret on a 0.x version, so minor bumps can change it. Review the diff when upgrading.
