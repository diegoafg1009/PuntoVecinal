# Design

## Context

See proposal.md. The frontend (Nuxt 4, Nuxt UI v4, Pinia, SPA) has only `welcome.vue` and `index.vue` (a `/me` check), wrapped by `app.vue` in `UApp`; there is no layout and no navigation. `frontend/AGENTS.md` fixes the navigation (mobile bottom bar, desktop top menu, avatar to Perfil), says the Reportar draft lives in Pinia, and asks for Nuxt UI components. Nothing here talks to the API, so the contract and backend are untouched.

Figma frames used (file `sC2o7u0tojnA5XjhlrE9Ts`):
- Mobile «01 · Inicio» (`5508:1539`): top bar with brand and avatar (28 px); bottom bar with Inicio, Buscar, Reportar (small inline green square button with «+», green label), Avisos, Mapa; current item green.
- Mobile «05 · Reportar: qué pasa (1 de 3)» (`5509:1828`): no app bars; own bar with close (X), «Nuevo reporte», «1 de 3»; `UStepper`, red `UAlert`, category buttons, `UInput`, `UTextarea`, full-width «Seguir ›» button.
- Desktop «05 · Reportar: qué pasa · PC» (`5517:1958`): `UHeader` 64 px with brand, `UNavigationMenu`, green «+ Reportar» button, avatar; the same form in a 560 px card centered on a gray background.

## Goals / Non-Goals

**Goals:**
- One app shell that switches between the mobile and desktop navigation by width, with the same routes on both.
- A draft store shaped so HU-09 and HU-10 add photos, location and rules acceptance without reworking step 1.
- Validation and persistence testable without a backend.

**Non-Goals:**
- Content of Inicio, Buscar, Avisos, Mapa and Perfil; photos, location, review, publishing; API calls; the moderator and case manager panel.

## Decisions

### 1. Layouts and navigation components
- `app/layouts/default.vue` is the app shell: `AppHeader` (desktop, from the `lg` breakpoint up) or `AppTopBar` + `AppBottomBar` (below it), around `<slot />`. Visibility is done with Tailwind responsive classes, so resizing switches the bars without remounting the page. Desktop content is centered in a `UContainer`.
- `app/layouts/blank.vue` has no navigation; `welcome.vue` uses it.
- A page that is a focused screen on mobile (Reportar) sets `definePageMeta({ focusedOnMobile: true })`; the default layout hides `AppTopBar` and `AppBottomBar` for it, while `AppHeader` still shows on desktop. A page-meta flag keeps the rule next to the page instead of a route-name list in the layout.
- Components: `UNavigationMenu` for the desktop menu and the bottom bar items; the Reportar item of the bottom bar is a small green square (as in Figma) between the items. The current section comes from the route.
*Alternative:* two separate layouts (mobile/desktop) chosen by a JS media query. Rejected: causes a flash and remounts on resize.

### 2. Sections and routes
Routes: `/` (Inicio), `/buscar`, `/reportar`, `/avisos`, `/mapa`, `/perfil`. Buscar, Avisos, Mapa and Perfil are pages that render a shared `ComingSoon` block with «Próximamente». The existing auth guard already protects them. The avatar shows the neighbor's initial or a generic user icon (no profile data exists yet) and links to `/perfil`.

### 3. Reportar: one page, step held in the draft store
`/reportar` renders the component of the current step (`draft.step`: 1 to 3) under the form bar (close, «Nuevo reporte», «n de 3») and a `UStepper`. Steps 2 and 3 render a placeholder until their stories. On desktop the form is a 560 px card (`UCard`) on the gray page background; on mobile it fills the screen. *Alternative:* one route per step. Rejected for now: no deep-link need, and the store already has to know the step to restore it.

### 4. Close button
The X goes to `/`; Inicio is the assumption, since the Figma frame does not say where it goes and Reportar is opened from the app sections. If the draft has data (a category, or a non-blank «¿Qué pasa?» or «Más detalles»), a `UModal` first asks «¿Guardar borrador?» with «Guardar borrador» (go to `/`, keep the draft) and «Descartar» (reset the draft, go to `/`). Dismissing the modal (Esc or outside click) stays on the form. With an empty draft the X goes straight to `/`. Leaving without the X (switching tab, reloading) never asks and keeps the draft. *Alternative:* always keep the draft, as before. Rejected: the neighbor had no way to start over, and apps like X ask on close.

### 5. Draft in Pinia, mirrored to localStorage
`app/stores/reportDraft.ts` holds `category`, `title` («¿Qué pasa?»), `details` («Más detalles») and `step`. Pinia keeps it across navigation; a `watch` writes it to `localStorage` under one key, and the store reads it on first use. Every storage access is wrapped in try/catch so blocked storage only disables restoring (spec scenario). Stored data is read defensively (ignore unknown or malformed values and start empty) so later stories can change the shape. *Alternative:* `pinia-plugin-persistedstate`. Rejected: a new dependency for a ~15-line need. The store also has `reset()`, which empties the draft and its stored copy; the close dialog calls it on «Descartar», and HU-10 will call it on publish.

### 6. Draft belongs to the device, not to the user id
The key is not scoped by user id, so anonymous and account sessions on the same device share one draft, matching «recuperar lo que había escrito». Revisit if account switching on a shared device matters.

### 7. Step 1 form and validation
Category: five `UButton`s as single-choice chips (selected state shows the check, as in Figma), driven by a constant in `app/utils/categories.ts` with a stable slug (such as `limpieza`) and the Spanish label; HU-10 will send the slug and the contract will define the enum then. «¿Qué pasa?» is a `UInput` with the help text, «Más detalles» a `UTextarea`, «Seguir ›» a full-width `UButton`. The emergency notice is a fixed `UAlert` (error color). Validation uses a plain `validate` function with `UForm` (zod is not installed): category required, «¿Qué pasa?» required after trim; errors appear on «Seguir», not while typing. Limits are an assumption (not in the story): «¿Qué pasa?» up to 100 characters (one line), «Más detalles» up to 1000, via `maxlength`.

### 8. Icons
Icons come from [Nuxt Icon](https://github.com/nuxt/icon) (`@nuxt/icon`, already installed and registered by `@nuxt/ui`), used as `UIcon` or the `icon` prop with `i-lucide-*` names. The Figma frames use Lucide (house, search, bell, map, map-pin, plus, user, x, chevron-right, camera, flag, message-circle). By default Nuxt Icon fetches icons from the Iconify API at runtime; to avoid depending on it, install the Lucide collection locally (`@iconify-json/lucide`), which Nuxt Icon then serves from the bundle. No `@nuxt/icon` module entry is added by hand.

## Risks / Trade-offs

- [Stale draft shown after a long time] → acceptable for MVP; HU-10 clears it on publish.
- [Draft shared by accounts on one device] → Decision 6.
- [Stored drafts break when later stories change the shape] → defensive read (Decision 5).
- [Field limits and the X destination are assumed] → flagged for review; easy to change.
- [Bars hidden by CSS are still in the DOM] → acceptable; the `lg` split keeps one set visible, and hidden sets are not focusable (`hidden` removes them from the accessibility tree).
