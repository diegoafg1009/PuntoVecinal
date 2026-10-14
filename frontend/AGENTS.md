# Punto Vecinal frontend

Also read the root `AGENTS.md`: its rules (spec first, git, secrets, language) apply here.

## Applying OpenSpec changes from here

When you run `/opsx:apply` (or the `openspec-apply-change` skill) from `frontend/`, do only the tasks prefixed `[frontend]`. Leave `[backend]`, `[db]` and `[all]` tasks unchecked and, when you finish, list the tasks that remain for other folders. If a task depends on a contract change that the `[backend]` tasks have not done yet (`openapi/` updated and `bun run generate:api` able to produce the new client), stop and say so instead of editing the contract yourself. If a task has no prefix, ask which area it belongs to.

## Stack

- **Framework:** Nuxt with Nuxt UI (v4).
- **Data:** everything goes through our own backend API, per the contract in `openapi/` (bundled in `openapi/dist/openapi.yaml` by `make bundle`). The frontend uses Supabase only for authentication (including the anonymous session for neighbors without an account); it never reads or writes tables directly.
- **API client:** generated from the contract with [Hey API](https://heyapi.dev) into `app/api/` (committed, never edited by hand): types, an SDK with one function per `operationId`, and TanStack Query helpers (`getMeOptions()`, ...). Import from `~/api/...`. The plugin `app/plugins/api.ts` sets the base URL and adds the Supabase access token to every request; don't build `$fetch` calls to the API by hand.
- **Server state:** TanStack Query (Vue Query) handles all data fetched from or sent to the backend API: queries, mutations, caching, and invalidation. Don't copy API responses into Pinia stores.
- **Client state:** Pinia handles state that lives only in the frontend, such as the auth session, the Reportar draft across its 3 steps, and UI preferences.

## Design

UI labels are in Spanish, as users see them.

- Responsive web with the same sections on mobile and desktop: Inicio, Buscar, Reportar, Avisos, Mapa, and Perfil. Whatever changes in one version changes in the other.
- **Mobile:** bottom bar with Inicio, Buscar, Reportar, Avisos, and Mapa; Perfil opens from the avatar at the top right.
- **Desktop:** top menu with Inicio, Buscar, Avisos, and Mapa, plus the Reportar button and the avatar; content is centered.
- **Reportar** has 3 steps: Qué pasa, Foto y lugar, Revisar. If there are similar reports, they show up in a dialog when leaving step 2, not as an extra step.
- Publishing opens the report detail with a confirmation toast; there is no separate "published" screen.
- The support reaction on a report is called **"También me afecta"**.
- The moderator and case manager panel is desktop only.
- Prefer Nuxt UI components over custom ones.

## Figma

The designs are in Figma: https://www.figma.com/design/sC2o7u0tojnA5XjhlrE9Ts

- Page **"📱 Punto Vecinal · Celular"**: mobile screens.
- Page **"🖥 Punto Vecinal · PC"**: desktop screens and the moderator and case manager panel.
- Each page has a "Sin cuenta" (no account) block and a "Con cuenta" (with account) block.
- Screens use the Nuxt UI kit components in the same file; implement them with the equivalent Nuxt UI components.

Read them through the Figma MCP server (configured in `.mcp.json` for Claude Code and `.codex/config.toml` for Codex). Ask for the link to the specific frame instead of browsing the whole file: the account has a daily call limit.

## Commands

Run everything from `frontend/` with [bun](https://bun.sh).

Setup:

1. `bun install`.
2. `cp .env.example .env` and set `NUXT_PUBLIC_SUPABASE_ANON_KEY` (the `ANON_KEY` printed by `make supabase-status` in `backend/`). The other values point at the local Supabase stack and the API on `:8080`. `.env` is ignored by git.
3. Start the local Supabase stack and the API first (see `backend/AGENTS.md`); the app needs both to let a neighbor in.

| Command | What it does |
| --- | --- |
| `bun run dev` | Dev server at `http://localhost:3000`. |
| `bun run build` | Production build (`.output/`). |
| `bun run preview` | Serve the production build locally. |
| `bun run typecheck` | Type-checks the app (`nuxt typecheck` with `vue-tsc`). |
| `bun run generate:api` | Regenerates `app/api/` from `openapi/dist/openapi.yaml`. Run `make bundle` in `backend/` first, and commit the result with the contract change. |

There is no test runner yet. The app runs as an SPA (`ssr: false`), so the Supabase session is read in the browser.
