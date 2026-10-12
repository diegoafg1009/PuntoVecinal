# Punto Vecinal

Instructions for coding agents (Codex and Claude Code). This file is the single source of truth: `CLAUDE.md` only imports it, so any new rule goes here. `backend/` and `frontend/` each have their own `AGENTS.md` with what is specific to that part; read it in addition to this one when working there.

## The project

A citizen reporting web platform for Cooperativa Primavera, Comas (Lima, Peru). Neighbors report problems in the neighborhood with a photo and location, other neighbors back a report with "También me afecta" ("This affects me too"), and case managers forward reports to the responsible entity and update their status. It is a university project (UNMSM) built with Design Thinking.

## Structure

- `backend/`: our own API in Go.
- `frontend/`: responsive web app in Nuxt.
- `supabase/`: Supabase project; `supabase/migrations/` holds the versioned database schema.
- `openspec/`: specifications (spec-driven development with OpenSpec). `openspec/specs/` holds current behavior and `openspec/changes/` holds changes in progress.

## Domain

The app's users speak Spanish, so UI labels are given in Spanish below.

- **Actors:** neighbor without an account (anonymous session), neighbor with an account, moderator, and case manager. There is no administrator.
  - The moderator only moderates (reviews flags, blurs faces and license plates, hides or restores content), from a web panel.
  - The case manager moderates, forwards reports, updates statuses, marks duplicates, and sees metrics.
- **Categories (fixed):** Limpieza, Alumbrado, Pistas y veredas, Agua y desagüe, Otro. Zones are also fixed; nobody manages them from the app.
- **Report statuses:** Publicado → Derivado → Recibido por la entidad → En atención → Atendido según la entidad, or Cerrado. They only move forward. "Derivado, sin respuesta" is an automatic label applied 15 business days after forwarding, not a status.
- **Rules:** each report has 1 to 3 photos (at least one). A single neighbor's "it's fixed" photo is enough for the case manager to mark it Atendido.
- **Notifications:** always by email to anyone with a registered email; web notifications only if the neighbor opts in. New-report alerts cover a radius of ~300 m around the neighbor.

## How we work

- **Spec first:** every behavior change starts as an OpenSpec change, and every API change updates the OpenAPI contract first. Code is implemented afterwards, following the spec.
  - Flow: propose the change (`openspec-propose` skill, or `/opsx:propose` in Claude Code), implement it (`openspec-apply-change`), and archive it when done (`openspec-archive-change`). Run `openspec validate` before calling a change ready.
- **User stories:** they live in Jira. When a task cites one (for example `PV-12`), read it with the Atlassian MCP server (`atlassian`) instead of guessing; if that server is unavailable, ask the user for the story's text.
- **Scope:** do only what was asked. Don't add features, screens, or endpoints on your own; if something seems missing, ask.
- **Language:** code, comments, logs, documentation, and specs are written in English. Exceptions: text shown to users in the app, and `README.md`, are in Spanish.

## Exploring the code

Use the CodeGraph tools (MCP server `codegraph`, configured in `.mcp.json` and `.codex/config.toml`) to find symbols, callers, and dependencies before walking through files by hand.

If the CodeGraph tools are not available in your session, tell the user at the start, before working, and give them these steps:

1. Install: `npm i -g @colbymchenry/codegraph`
2. Build the index at the project root: `codegraph init`
3. Restart the agent so it loads the server.

Same for Figma and Jira: if a task needs the designs or a user story and the `figma` or `atlassian` server does not respond, ask the user to sign in (`/mcp` in Claude Code, or `codex mcp login figma` / `codex mcp login atlassian` in Codex).

## Git (mandatory)

- Do not create commits, pushes, branches, tags, or pull requests unless the user explicitly asks for that change.
- Never write directly to `main`.
- When in doubt, ask before touching history or the remote.

## Secrets

- Never put keys, tokens, or passwords in code or in versioned files. They go in `.env` (ignored by git) and are documented with sample values in `.env.example`.
