# Tasks

## 1. API contract

- [x] 1.1 Update `backend/api/openapi.yaml`: add `bearerAuth` security scheme, global `security`, `security: []` on `getHealthz`, and `GET /me` (operationId `getMe`, `Me` schema with `id` uuid and `isAnonymous`, 200 and 401 responses). Verify the codegen in 1.2 succeeds on the updated contract.
- [x] 1.2 Regenerate the HTTP layer from `backend/` with `go tool oapi-codegen -config api/http/oapi-codegen.yaml api/openapi.yaml` and verify `go build ./...` fails only on the missing `GetMe` method.

## 2. Backend authentication

- [x] 2.1 Add `github.com/golang-jwt/jwt/v5` and `github.com/MicahParks/keyfunc/v3`; make `SUPABASE_URL` and `SUPABASE_JWKSURL` required in `config.validate`, update `.env.example`; verify `go test ./internal/platform/config/...` covers the new validation.
- [x] 2.2 Create `internal/platform/auth` (token verification against JWKS, issuer/audience/expiry checks, `Principal` in context) with unit tests using a fake JWKS server: valid anonymous token, valid non-anonymous token, expired, bad signature, wrong issuer, missing token. Verify `go test ./internal/platform/auth/...` passes.
- [x] 2.3 Wire the auth middleware through the generated handler options in `routes.go`, enforcing only where `BearerAuthScopes` is present, and implement `GetMe` returning the `Principal`. Verify with handler tests: `/me` 200 with matching id and `isAnonymous`, `/me` 401 without or with a bad token, `/healthz` reachable without token.
- [x] 2.4 Document the auth flow and new env vars in `backend/README.md` (in Spanish where the README is Spanish) and verify with `curl` against a local stack: `/me` with an anonymous token from Supabase returns the same id twice.

## 3. Frontend entry flow

- [x] 3.1 Add `@supabase/supabase-js`, `@pinia/nuxt`, `@tanstack/vue-query` ; set `ssr: false`; add runtime config and `frontend/.env.example` for the Supabase URL, anon key and API base URL; verify `bun run dev` starts and `bun run build` succeeds.
- [x] 3.2 Request the Figma link for the welcome frame (mobile and desktop "Sin cuenta"), then build `app/pages/welcome.vue` with Nuxt UI: both buttons, «Próximamente» toast on «Ya tengo una cuenta», error alert with retry on failed guest entry. Verify visually in the browser at mobile and desktop widths.
- [x] 3.3 Implement the Pinia session store, Supabase client, and `auth.global.ts` guard (decision 1). Verify in the browser: first visit shows the welcome screen; «Comenzar» enters `/`; reload skips the welcome screen; deleting the stored session shows it again.
- [x] 3.4 Replace the placeholder `app/pages/index.vue` with a `GET /me` TanStack Query call (bearer from the session) that shows the entry succeeded; verify the same user id is returned after reloading the page.
- [x] 3.5 Fill in the frontend `AGENTS.md` Commands section with install/run/build commands, and verify they run as written.

## 4. Integration check

- [x] 4.1 Verify `[auth.sessions]` in `supabase/config.toml` has no `timebox` or `inactivity_timeout`, and note in the PR that the hosted project must match. Verify by inspecting the file.
- [x] 4.2 With local Supabase, backend and frontend running, walk through every acceptance criterion of PV-10 (welcome screen, guest entry without data, no welcome on reopen, same anonymous user id, «Próximamente») and record the result in the PV-49 test notes. Include forcing a token refresh (expire the access token) and verify `GET /me` returns the same id before and after.
- [x] 4.3 Run `openspec validate guest-entry` and `go test ./...` from `backend/`; both pass.

## Workflow follow-up

- Archive the change (`/opsx:archive`) once review requirements are met.
