# Proposal

## Why

The API contract is a single file (`backend/api/openapi.yaml`) and all generated code lands in one package (`internal/rest/gen`). As soon as reports, comments and notifications arrive, one file and one package would hold every endpoint. The team wants the contract split per module and each module to own its generated code, handler and service, with Redocly CLI producing one bundled contract.

## What Changes

- Split the contract into `backend/openapi/`: a root `openapi.yaml`, one file per module in `modules/`, shared schemas and responses in `schemas/common.yaml`, and one oapi-codegen config per module in `config/`.
- Add Redocly CLI (pinned, run through `npx`) to bundle the tree into `backend/openapi/dist/openapi.yaml`. `openapi/dist/` is ignored by git.
- Generate each module's HTTP layer into `internal/<module>/api/generated.go` from the bundle, filtered by tag, instead of one package for everything.
- Move the Go code to modules, each with its own handler and service:
  - `internal/auth/` (`GET /me`).
  - `internal/system/` (`GET /healthz`, kept in the root contract file).
- `internal/platform/httpserver` keeps only what is shared: router, CORS, request logging and mounting the modules. Authentication moves to the `OapiRequestValidatorWithOptions` middleware from `oapi-codegen/nethttp-middleware`: it applies the `security` of each operation in the contract, calling our `NewAuthFunc(verifier)` only where the contract requires a token. `Principal` is renamed to `Claims`.
- Remove `backend/api/` and `internal/rest/gen`. Update `Makefile`, `.gitignore`, `backend/AGENTS.md`, `backend/README.md` and the paths named in `openspec/config.yaml` rules.

No endpoint, schema, status code or behavior changes. The contract's `security` stays the single source of truth for which operations need a token. Out of scope: `reports`, `comments` and `notifications` modules (they are added with their first endpoint), linting rules beyond what bundling validates.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. This is a structure and tooling change; `specs` is skipped (`skip_specs: true`).

## Impact

- `backend/`: new `openapi/` tree and `openapi/dist/` (ignored); new `internal/auth/` and `internal/system/`; reduced `internal/platform/httpserver`; `internal/platform/auth` gains `NewAuthFunc` and renames `Principal` to `Claims`; new dependency `github.com/oapi-codegen/nethttp-middleware`; `Makefile`, `.gitignore`.
- Tooling: Node and `npx @redocly/cli@<pinned>` become a requirement of `make generate-api` (Node is already used by the frontend).
- Docs: `backend/AGENTS.md`, `backend/README.md`, `openspec/config.yaml` (artifact rule paths).
- Interaction with `guest-entry`: it is implemented but not archived, and its tasks mention the old paths. Nothing in its specs changes; archive it before or after this change.
