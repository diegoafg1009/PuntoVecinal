# Proposal

## Why

A neighbor must be able to start using Punto Vecinal without paperwork (PV-10, HU-01; RF-01, RNF-08). Today the app has no entry flow and the API has no way to tell who is calling it. Every later feature (reporting, "También me afecta", notifications) needs a user identity, and a neighbor without an account gets one through an anonymous Supabase session.

## What Changes

- Add a welcome screen shown on first visit with two options: «Comenzar» and «Ya tengo una cuenta».
- «Comenzar» creates an anonymous Supabase session on the device, with no data requested, and enters the app.
- The chosen entry option is remembered on the device: reopening the app skips the welcome screen and keeps the same anonymous session.
- «Ya tengo una cuenta» is a placeholder: it shows «Próximamente» and lets the neighbor continue as guest. Real login is out of scope.
- The API verifies the Supabase JWT (anonymous sessions included) on every protected request and exposes the caller's identity through a new `GET /me` endpoint. Sign-in, sign-up and token refresh are done by Supabase itself (supabase-js in the frontend); the API has no endpoints for them.

Out of scope: any API endpoint for login, registration or refresh, email/password login, account creation, linking an anonymous user to an account, roles (moderator, case manager), any other screen of the app.

## Capabilities

### New Capabilities
- `app-entry`: welcome screen, guest entry with an anonymous session, remembering the choice and the session across visits, and the «Ya tengo una cuenta» «Próximamente» placeholder.
- `api-authentication`: the API accepts only valid Supabase sessions (anonymous or not) on protected endpoints and tells the caller who they are through `GET /me`.

### Modified Capabilities

None (`openspec/specs/` is empty).

## Impact

- `frontend/`: new welcome page, route guard, auth store (Pinia), Supabase client; new dependencies (`@supabase/supabase-js`, Pinia module, TanStack Query for the `/me` call) and public runtime config for the Supabase URL and anon key (`frontend/.env.example`).
- `backend/`: new auth middleware verifying the JWT against `SUPABASE_JWKSURL` (already in config), `GET /me`, updates to `backend/api/openapi.yaml` and regenerated `internal/rest/gen/api.gen.go`; a JWT/JWKS library dependency.
- `supabase/config.toml`: anonymous sign-ins are already enabled (`enable_anonymous_sign_ins = true`); no change expected.
- No database migration: identity lives in Supabase Auth.
