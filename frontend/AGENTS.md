# Punto Vecinal frontend

Also read the root `AGENTS.md`: its rules (spec first, git, secrets, language) apply here.

## Stack

- **Framework:** Nuxt with Nuxt UI (v4).
- **Data:** everything goes through our own backend API, per `backend/api/openapi.yaml`. The frontend uses Supabase only for authentication (including the anonymous session for neighbors without an account); it never reads or writes tables directly.

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

To be defined once the Nuxt project exists. When adding them, document here how to install, run, and test.
