# Design

## Context

See proposal.md for motivation. Current state:

- One contract, `backend/api/openapi.yaml`, with `/healthz` (public) and `/me` (bearer auth); oapi-codegen config in `backend/api/http/oapi-codegen.yaml`; output in `internal/rest/gen` (package `api`).
- Handlers and auth middleware live in `internal/platform/httpserver`. Authorization is a hand-written strict-handler middleware that reads `security: []` from the embedded spec and is keyed by operation id.
- Each generated package defines its own `StrictHandlerFunc` and `StrictMiddlewareFunc` types, so a hand-written strict middleware cannot be shared across generated packages without per-module adapters.
- Target layout (from the team's diagram): `openapi/{openapi.yaml, modules/, schemas/common.yaml, config/}` and `internal/<module>/{api/generated.go, handler.go, service.go}`.

## Goals / Non-Goals

**Goals:**
- The authored contract is split by module and bundled by Redocly into `openapi/dist/openapi.yaml`.
- Each module owns its generated code, handler and service.
- Zero behavior change: same endpoints, bodies, status codes and auth rules, proven by the existing tests and the live check.

**Non-Goals:**
- New endpoints or modules; changes to authentication rules.
- Committing `openapi/dist/` or adding a CI pipeline.

## Decisions

### 1. Layout and what each file holds
```
backend/
  openapi/
    openapi.yaml          # info, global security, tags, /healthz inline, and $refs to module paths
    modules/auth.yaml     # `paths:` with /me, plus the schemas only that module uses (Me)
    schemas/common.yaml   # shared schemas and responses (Unauthorized)
    config/auth.yaml      # oapi-codegen config for module auth
    config/system.yaml    # oapi-codegen config for /healthz
    dist/openapi.yaml     # bundle, ignored by git
  internal/auth/    api/generated.go, handler.go, service.go
  internal/system/  api/generated.go, handler.go
```
The root `openapi.yaml` references each module path with `$ref: ./modules/auth.yaml#/paths/~1me`, so module files stay readable on their own. `/healthz` stays in the root file as requested; its tag is `system`. `config/system.yaml` and `internal/system/` are the only additions to the team's diagram, needed so `/healthz` still has generated code and a home.

### 2. Redocly CLI through a pinned `npx`
`make bundle` runs `npx --yes @redocly/cli@2.63.0 bundle openapi/openapi.yaml -o openapi/dist/openapi.yaml`. The version is pinned in the Makefile so everyone bundles the same way; Node is already required by the frontend. `openapi/dist/` is git-ignored, so generated Go code stays committed (as before) and `make generate-api` always rebuilds the bundle first. *Alternatives:* Docker image (extra dependency), committing `openapi/dist/` (a second copy of the contract to keep in sync).

### 3. One oapi-codegen run per module, filtered by tag
Each `config/<module>.yaml` sets `package: api`, `output: internal/<module>/api/generated.go`, the same generators as today (models, chi-server, strict-server, embedded-spec) and `output-options.include-tags: [<module>]`; every operation carries its module tag. All runs read `openapi/dist/openapi.yaml`, so cross-file `$ref`s are already resolved. Unused schemas are pruned by oapi-codegen, so a module only gets the types it references. `make generate-api` loops over `openapi/config/*.yaml`. *Alternative:* generating from each module file directly, rejected because oapi-codegen needs import mappings to resolve `../schemas/common.yaml` across packages.

### 4. Contract-driven auth with `OapiRequestValidatorWithOptions`
Authentication uses the validator middleware from `github.com/oapi-codegen/nethttp-middleware` (a plain `func(http.Handler) http.Handler`), configured with the module's spec and `openapi3filter.Options{AuthenticationFunc: auth.NewAuthFunc(verifier)}`. For each request the library finds the operation through a router built from the spec, and:
- if the operation declares `security: []` (today `/healthz`), `AuthenticationFunc` is not called and the request passes without a token;
- otherwise it calls `NewAuthFunc`, which accepts only the `bearerAuth` scheme, verifies the bearer token with the `Verifier`, and attaches the `Claims` to the request (see below);
- a route that is not in the spec gets 404 from the validator, so nothing skips authentication by being unknown.

The contract's `security` is the only place that says what is public, and a new operation is protected by default. Each module builds its own validator from its embedded spec (`api.GetSpec()`) and passes it through the generated `ChiServerOptions.Middlewares`, which sidesteps the per-package strict types. The embedded spec of every generated package also lists the other modules' paths, but as empty entries: oapi-codegen removes the operations that do not match the module's tag (and prunes unused schemas) yet keeps the path keys. A path with no operations never matches a request, and a module's validator only wraps that module's routes, so this is harmless. A prefix, if the API is ever mounted under one, is set once through `Options.Prefix`.

Details settled here:
- **Claims in the context.** `AuthenticationFunc` only returns an error and cannot replace the request context, so `NewAuthFunc` sets the context on the incoming request through the pointer the validator receives (`*req = *req.WithContext(auth.WithClaims(ctx, claims))`), which is the same request the next handler gets. It is commented in code and covered by a test (handler reads `ClaimsFrom`).
- **401 body.** `ErrorHandlerWithOpts` maps a security failure to `401 unauthorized` (`text/plain`, as today), a missing route to 404, and anything else to the library's default.
- **Rename.** `Principal` becomes `Claims` (`auth.Claims`, `auth.WithClaims`, `auth.ClaimsFrom`) in the `auth` package, handlers and tests; fields stay `UserID` and `IsAnonymous`.
- **Extra validation.** The validator also checks parameters and request bodies against the contract. `/me` and `/healthz` have none, so behavior does not change now; for future endpoints a 400 on a request that breaks the contract is accepted behavior.
- `Verifier` and `Claims` know nothing about OpenAPI; only `NewAuthFunc` and the router wiring do.

*Alternatives:* a hand-written `net/http` middleware that looks operations up in the spec by route pattern (reimplements what the library does and needs a start-up route check); a middleware that never reads the contract and leaves public routes to the router (two places to keep in sync); one strict middleware adapter per module (repeated code).

### 5. Module shape
- `handler.go` implements the module's generated `StrictServerInterface` and only adapts types; `service.go` holds the logic (for `auth`: building the caller identity from the `Claims`; `system` has only a handler for the readiness ping).
- Each module exposes `Register(r chi.Router, verifier auth.RequestVerifier) error` (`system` also takes its database `Pinger`). `Register` builds the module's auth validator from its own embedded spec and mounts the routes with the generated `HandlerWithOptions`, so `httpserver` mounts modules without knowing their routes or how they are protected. Passing the verifier instead of ready-made middlewares keeps the wiring in one place and uniform for every module.
- Handler errors that are not part of the contract (anything returned as `error`) answer 500 through a shared `internal/platform/httperror` package, which logs the cause and never exposes it.
- `internal/platform/httpserver` keeps router, CORS, request logging and `NewServer`; the readiness `Pinger` interface moves to `internal/system` (`httpserver` keeps a one-method `pinger` only to hold the connection pool).

### 6. Verifying nothing changed
The bundled `openapi/dist/openapi.yaml` is compared with the previous `api/openapi.yaml` (same paths, operations, schemas, security) before the old file is deleted. Existing handler and auth tests are ported to the new packages unchanged in intent, and the live `curl` check from `guest-entry` is repeated.

## Risks / Trade-offs

- [Redocly renames hoisted components, so generated Go type names change] → compare bundle against the old contract and keep names via explicit `components` keys; the tests fail on a renamed type.
- [`include-tags` or the embedded spec of a module may include more than its own operations] → a test per module asserts which operation ids its generated spec exposes.
- [The validator also enforces parameters and bodies against the contract, which would change responses for future endpoints] → no current endpoint has either; revisit with the first endpoint that does (options can exclude body or query validation).
- [A bug in the context workaround would hide the `Claims` from handlers] → covered by a handler test that reads `ClaimsFrom`.
- [New dependency `nethttp-middleware`, which brings `gorilla/mux` for its router] → maintained by the oapi-codegen project; used only inside `internal/platform`.
- [`make generate-api` now needs Node and network on the first `npx` run] → documented in `backend/AGENTS.md`; Node is already a project requirement.
- [`guest-entry` task text mentions old paths] → no code impact; it is a completed planning record.
