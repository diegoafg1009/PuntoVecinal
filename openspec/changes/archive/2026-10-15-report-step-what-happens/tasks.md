# Tasks

## 1. App shell and navigation

- [x] 1.1 [frontend] Install the Lucide collection for Nuxt Icon (`@iconify-json/lucide`; `@nuxt/icon` already comes with Nuxt UI), then create `app/layouts/blank.vue` (no navigation) and `app/layouts/default.vue` with the `focusedOnMobile` page-meta rule; verify `bun run typecheck` passes and `i-lucide-*` icons render with the network off.
- [x] 1.2 [frontend] Build `AppHeader` (desktop: brand, Inicio, Buscar, Avisos, Mapa, «Reportar» button, avatar) following Figma node `5517:1958`; verify at desktop width that the current section is marked and the avatar opens `/perfil`.
- [x] 1.3 [frontend] Build `AppTopBar` (brand and avatar) and `AppBottomBar` (Inicio, Buscar, highlighted Reportar, Avisos, Mapa) following Figma node `5508:1539`; verify at mobile width that the current section is marked, each item navigates, and the avatar opens `/perfil`.
- [x] 1.4 [frontend] Add pages `/buscar`, `/avisos`, `/mapa`, `/perfil` with a shared «Próximamente» block, move `welcome.vue` to the `blank` layout and put `index.vue` under the shell; verify each route shows «Próximamente» with the navigation visible, the welcome screen shows no bars, and resizing keeps the same section.

## 2. Draft store and categories

- [x] 2.1 [frontend] Add `app/utils/categories.ts` with the five fixed categories (slug and Spanish label); verify `bun run typecheck` passes.
- [x] 2.2 [frontend] Add `app/stores/reportDraft.ts` (Pinia: `category`, `title`, `details`, `step`, `validateStep1()`) with device persistence in `localStorage`, defensive reading and try/catch around storage; verify in the browser that the draft survives a reload and that blocked storage does not break the form.

## 3. Reportar form

- [x] 3.1 [frontend] Build `app/pages/reportar.vue` with `focusedOnMobile`, the form bar (close, «Nuevo reporte», «n de 3»), the `UStepper` (Qué pasa, Foto y lugar, Revisar) and placeholders for steps 2 and 3, following Figma nodes `5509:1828` (mobile) and `5517:1958` (desktop card); verify on mobile that the bars are hidden and on desktop that the card sits under the header.
- [x] 3.2 [frontend] Build step 1: fixed emergency `UAlert`, single-choice category chips, «¿Qué pasa?» (required, max 100) with its help text, «Más detalles» (optional, max 1000) and «Seguir ›»; verify only the five categories are offered, choosing one deselects the other, and a valid step 1 moves to step 2.
- [x] 3.3 [frontend] Show Spanish error messages and block «Seguir» when the category is missing or «¿Qué pasa?» is empty or whitespace-only; verify each case in the browser stays on step 1 with the message.
- [x] 3.4 [frontend] Make the close button go to `/` keeping the draft; verify that closing and opening Reportar again restores the text and the category.
- [x] 3.5 [frontend] Add `reset()` to `reportDraft` (empties the fields, step and stored copy) and a `UModal` on close when the draft has data («¿Guardar borrador?» with «Guardar borrador» and «Descartar»; dismissing stays on the form; an empty draft closes without asking); verify each scenario of «Closing the form asks what to do with the draft» in the browser.

## 4. Integration check

- [x] 4.1 [all] Walk through every acceptance criterion of PV-17 in the browser at mobile and desktop widths (3-step indicator, required fields, optional details, empty «¿Qué pasa?» blocked, leave and return restores the draft, closing with data asks to save or discard; the search-offer criterion is covered by the Reportar entries) plus the navigation scenarios of the specs, and record the result in the PV-68 test notes.
- [x] 4.2 [all] Run `openspec validate report-step-what-happens` and `bun run typecheck` from `frontend/`; both pass.

## Workflow follow-up

- Archive the change (`/opsx:archive`) once review requirements are met.
