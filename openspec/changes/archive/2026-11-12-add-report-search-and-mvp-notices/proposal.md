# Proposal

## Why

The MVP sprint goal is that a neighbor without an account can enter, see, search and publish reports. Feed, detail and publishing are done, but three sections still open a bare «Próximamente» placeholder and the Reportar form only warns about emergencies on its first step. Four MVP stories close that gap:

- HU-04 (PV-13, «Buscar reportes»): search reports by words and by closeness, to know whether a street's problem was already reported.
- HU-05 (PV-14, «Ver el mapa como opción no disponible»): Mapa is shown as not available instead of opening an empty screen.
- HU-12 (PV-21, «Ver dónde llamar en una emergencia»): every step of Reportar shows where to call in an emergency, and the numbers can be dialed.
- HU-22 (PV-31, «Saber que Avisos necesita una cuenta»): Avisos explains that it needs an account instead of showing «Próximamente».

## What Changes

- New `GET /reports/search` operation: published reports whose title or details contain every word typed (ignoring case and accents), ordered from newest to oldest, or by distance from the caller when they ask for «cerca de mí». It can be called with words, with «cerca de mí», or with both. Results come in cursor-based pages with the same items as the feed. `GET /reports` stays as it is.
- Buscar becomes a search screen: a search box and a «Cerca de mí» option. The results are shown with the feed's cards, page by page, along with a message when nothing matches and states for loading and errors. «Cerca de mí» asks for the device location only when the neighbor turns it on. The search is kept in the address, so going back from a report shows the same results.
- Mapa stays in the mobile bar and the desktop header, marked as not available. Tapping it shows a «Próximamente» notice and keeps the neighbor on the current screen. Opening `/map` directly redirects to Inicio.
- The emergency notice moves from step 1 to a fixed notice at the top of every Reportar step. It names Serenazgo and lists the official numbers 105 (Policía), 116 (Bomberos) and 106 (SAMU), which open a call when tapped. No Serenazgo number is given until one is confirmed.
- For a neighbor without an account, Avisos says that notices need an account and that accounts will be available soon, with no button to a screen that does not exist yet.
- These criteria are already met and are kept as they are: the report page shows the written place reference (HU-05) and the fixed categories include no safety topic (HU-12). The specs record them as scenarios.
- Out of scope: search filters by category or status, search suggestions or history, the map itself, accounts and the notices list.

## Capabilities

### New Capabilities
- `report-search`: searching published reports by words and closeness, covering both the `GET /reports/search` operation and the Buscar screen.
- `notices`: the Avisos section. For now this covers only what a neighbor without an account sees.

### Modified Capabilities
- `app-navigation`: Mapa is marked as not available and does not open a section, `/map` redirects to Inicio, and only Perfil still shows «Próximamente».
- `report-creation`: the emergency notice is on every step, fixed, with Serenazgo and the official numbers as call links, and no category covers safety.

## Impact

- Contract: `openapi/modules/reports.yaml` gains `GET /reports/search` (`searchReports`), which reuses `ReportPage`. This regenerates `backend/internal/reports/api/generated.go` and `frontend/app/api/`.
- Database: a new migration enables the `unaccent` extension. A new search query goes in `backend/internal/reports/repository/queries/reports.sql`, and sqlc is regenerated.
- Backend: search methods in the repository, service and handler of `internal/reports`, plus tests.
- Frontend: `pages/search.vue` (Buscar), `pages/notices.vue` (Avisos), `pages/map.vue` (redirect), `AppBottomBar.vue` and `AppHeader.vue` (Mapa not available), `pages/report.vue` and `ReportStepWhatHappens.vue` (the emergency notice moves into its own component), and the session store (whether the session is anonymous).
- No new dependencies.
