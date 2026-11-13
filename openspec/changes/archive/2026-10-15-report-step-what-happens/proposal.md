# Proposal

## Why

Reporting a problem is the core action of Punto Vecinal, and it starts with telling what is happening (PV-17, HU-08; RF-07, RNF-02). Today the app has no Reportar flow and no way to move between sections: after entering, the neighbor lands on a placeholder page. This change delivers the app navigation, the entry to Reportar and its first step, «Qué pasa»; the next steps (HU-09, HU-10) build on it.

## What Changes

- Add the app navigation, following the Figma frames «01 · Inicio» (mobile) and «05 · Reportar: qué pasa · PC»:
  - Mobile: a top bar with the Punto Vecinal brand and the avatar (opens Perfil), and a bottom bar with Inicio, Buscar, Reportar (highlighted button), Avisos and Mapa.
  - Desktop: a header with the brand, the menu Inicio, Buscar, Avisos and Mapa, the «Reportar» button and the avatar; content centered.
  - Buscar, Avisos, Mapa and Perfil exist as routes that show «Próximamente», so no link leads nowhere.
  - The welcome screen has no navigation.
- Add the Reportar form at `/reportar`, with a 3-step indicator: «Qué pasa», «Foto y lugar», «Revisar». On mobile it is a focused screen without the app bars: a close button, «Nuevo reporte» and the counter «1 de 3». On desktop it is a centered card under the header.
- Step 1 shows a fixed emergency notice, asks for a category from the fixed list (Limpieza, Alumbrado, Pistas y veredas, Agua y desagüe, Otro), «¿Qué pasa?» (required, a one-line title) and «Más detalles» (optional).
- Tapping «Seguir» with a missing category or an empty «¿Qué pasa?» shows what is missing and does not advance.
- The draft is kept on the device: leaving the form halfway (switching tab, reloading) and coming back restores what was written.
- Closing the form with something written asks whether to save the draft or discard it; with an empty form it closes without asking.
- Steps 2 and 3 are placeholders here (the indicator shows them; their content arrives with HU-09 and HU-10).

Out of scope: the content of Inicio, Buscar, Avisos, Mapa and Perfil, notifications, the moderator and case manager panel, photos, location and the pilot-zone check (HU-09), review, rules acceptance, publishing and anonymity choice (HU-10, HU-11), similar-report detection, and any API or database change. The criterion «no search results → offer to report» is satisfied by the `/reportar` route and the Reportar entries; the search screen that links to it belongs to its own story.

## Capabilities

### New Capabilities
- `app-navigation`: the app shell a neighbor moves through: mobile top bar and bottom bar, desktop header, avatar to Perfil, the sections and their «Próximamente» placeholders.
- `report-creation`: the Reportar flow a neighbor follows to create a report. This change defines its 3-step form, step 1 (notice, category, «¿Qué pasa?», «Más detalles») with validation, and keeping the draft. Later stories add to this capability.

### Modified Capabilities

None.

## Impact

- `frontend/` only: layouts and navigation components, pages for the sections (`index`, `buscar`, `avisos`, `mapa`, `perfil`, `reportar`), a Pinia draft store with device persistence, step components.
- No change to `openapi/`, `backend/` or `supabase/`; nothing is sent to the API in this change.
- One new dependency: `@iconify-json/lucide`, the icon collection that Nuxt Icon (already included with Nuxt UI) serves locally. Pinia and Nuxt UI are already installed.
