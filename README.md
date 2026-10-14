# Punto Vecinal

Plataforma de reportes ciudadanos para la Cooperativa Primavera, Comas (Lima). Los vecinos reportan problemas del barrio (por ejemplo, basura acumulada) con foto y ubicación, y los gestores de casos los derivan y les dan seguimiento.

Proyecto del curso Taller de Aplicaciones Sociales (UNMSM, 2026-II), desarrollado con Design Thinking.

## Stack

- **Frontend:** Nuxt con Nuxt UI (web responsiva)
- **API:** Go (capa HTTP generada con oapi-codegen y acceso a datos con sqlc)
- **Base de datos y autenticación:** Supabase (Postgres + PostGIS, Auth, RLS)
- **Archivos:** DigitalOcean Spaces (compatible con S3)
- **Embeddings:** Voyage AI

## Metodología

Desarrollo guiado por especificaciones (SDD) con OpenSpec, y contrato OpenAPI antes de implementar cada endpoint.

## Preparar el entorno de los agentes

El proyecto usa Codex y Claude Code con las mismas reglas (`AGENTS.md`) y los mismos servidores MCP (`.mcp.json` y `.codex/config.toml`).

1. Instalar OpenSpec (Node 20.19+): `npm install -g @fission-ai/openspec@latest`
2. Instalar CodeGraph: `npm i -g @colbymchenry/codegraph`
3. En la raíz del proyecto: `codegraph init`
4. Iniciar sesión en Figma y en Jira (Atlassian): `/mcp` en Claude Code, o `codex mcp login figma` y `codex mcp login atlassian` en Codex (en Codex, marca el proyecto como confiable).

## Flujo de trabajo (SDD con OpenSpec)

Cada historia de usuario (HU) de Jira sigue este ciclo. En Claude Code se usan los comandos `/opsx:*`; en Codex, las skills del mismo nombre (por ejemplo, pidiéndole "usa openspec-propose").

1. **Explorar (opcional):** `/opsx:explore` si la HU tiene dudas. Solo conversa y lee código; no escribe nada.
2. **Proponer:** `/opsx:propose` con la clave de la HU, por ejemplo `/opsx:propose PV-12`. El agente lee la HU y sus criterios desde Jira; si no puede, pega su texto. El nombre del cambio incluye la clave. Crea `openspec/changes/<nombre>/` con `proposal.md`, `specs/` (requisitos y escenarios), `design.md` y `tasks.md`, y se detiene ahí.
3. **Revisar:** lee esos archivos, sobre todo los escenarios de `specs/` y el orden de `tasks.md`. Para corregir algo, `/opsx:update` con lo que quieres cambiar. Luego `openspec validate`.
4. **Implementar:** `/opsx:apply`. Sigue `tasks.md` y marca cada tarea. Si el cambio toca la API, primero actualiza el contrato en `openapi/` (módulos y esquemas) y regenera con `make generate-api` (desde `backend/`); si toca la base de datos, primero crea la migración y regenera con sqlc; después escribe el código.
5. **Probar y hacer commit:** revisa el resultado y pide el commit (los agentes no hacen commits por su cuenta). Usa la clave de Jira en la rama y el mensaje.
6. **Archivar:** `/opsx:archive`. Pasa los requisitos del cambio a `openspec/specs/` (la descripción vigente del sistema) y mueve el cambio a `openspec/changes/archive/`.

`/opsx:sync` solo pasa las specs a `openspec/specs/` sin archivar; normalmente no hace falta.

## Estado

En construcción.
