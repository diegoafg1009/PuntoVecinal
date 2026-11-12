# Proposal

## Why

Neighbors can publish and read reports, but nothing stops the same problem from being reported many times, and a report cannot show how many people it affects or gather new information. Four stories of Entrega 2 (epics EP-03 «Evitar reportes repetidos» and EP-04 «Participación vecinal») close that gap:

- HU-13 (PV-22, «Ver si ya lo reportó alguien»): before publishing, the neighbor sees similar reports near the place they marked, even when the other report describes the problem with other words.
- HU-14 (PV-23, «Sumarme a un reporte parecido»): instead of publishing a duplicate, the neighbor joins an existing report. Their support counts and their text and photo stay in the report as an aporte.
- HU-15 (PV-24, «Marcar "También me afecta"»): any neighbor can mark and unmark that a problem also affects them. The author always counts as affected.
- HU-16 (PV-25, «Aportar una foto o un dato nuevo»): any neighbor can add a text with an optional photo to an existing report.

## What Changes

- **Similar reports (HU-13).** Every published report gets a text embedding of its title and details from Voyage AI, stored with pgvector. Only that text is sent, never the author name, the reference or any user data. A new operation receives the draft's text and location, and returns up to 5 open reports (not «Atendido según la entidad» nor «Cerrado») within 300 m whose meaning is close enough, best match first. In Reportar, «Seguir» on step 2 calls it. With results it opens the dialog «¿Ya lo reportó alguien?» from the Figma design, a scrollable list where each item has a photo, title, category, distance, age and affected count. Without results it goes straight to «Revisar». «No es ninguno, seguir» goes to «Revisar» with the draft intact. If the check fails, the neighbor goes on to «Revisar» and is not blocked.
- **Joining (HU-14).** Tapping a similar report opens «Sumarte a este reporte». It shows a compact card of the report, the «Mostrar mi nombre o apodo» switch (off, so the neighbor is anonymous) and «Tu aporte», which always carries the draft's text and its first photo, as the Figma note says. Confirming marks «También me afecta» and adds the aporte in one request, without creating a report. It then clears the draft and opens the report with the notice «Te sumaste al reporte» («Ahora a N vecinos les afecta. Tu aporte ya aparece abajo.») and the support shown as marked. «No es el mismo, volver» goes back to the dialog.
- **«También me afecta» (HU-15).** New operations mark and unmark the caller's support, once per session. The report detail returns the affected count, which is the author plus the supporters, and whether the caller is the author, supports the report, or neither. The report page shows the «También me afecta · N» button. Tapping it marks or unmarks support and the count follows. For the author the button shows as marked and cannot be changed. Supporting never changes the status.
- **Aportes (HU-16).** New operations add an aporte (text, an optional photo and an optional author name) and list a report's aportes. Photos follow the same rules as report photos. The client cleans and reduces them, and the server re-encodes them so they carry no location or hidden data. The report page gains «Aportar», which opens the same «Sumarte a este reporte» screen. There «Tu aporte» starts empty, and the neighbor writes the text and adds one photo. Every aporte also marks «También me afecta», so aportar and sumarse are the same action. On success the report page shows «Te sumaste al reporte». It also gains the «Aportes y comentarios · N» list, where each aporte shows its author (or «Vecino anónimo»), its date, its text and its photo. The number of aportes one neighbor can add is limited.
- Out of scope, although the Figma frames show them: comments and the «Escribe un comentario» input, «¿Cómo está ahora?» («Sigue igual» / «Ya se resolvió»), «Seguimiento del caso», the «…» menu, the «Con cuenta» variants and the «guardar con correo» alert (accounts do not exist yet), notifications to the author, and the affected count on Mapa (its Figma frame does not show it).
- **Affected count on the cards.** As the Figma frames 01 and 02 show, the cards of Inicio and Buscar show the affected count next to a people icon. The feed and search items carry it.

## Capabilities

### New Capabilities
- `similar-reports`: finding open reports similar in meaning and close to a place, keeping published reports comparable, the «¿Ya lo reportó alguien?» dialog in Reportar, and joining a similar report instead of publishing.
- `report-support`: «También me afecta». This covers marking and unmarking support, who counts as affected and how the report page shows it.
- `report-contributions`: aportes. This covers adding a text with an optional photo to an existing report, listing aportes, the «Aportar» button that opens «Sumarte a este reporte», and the aportes list on the report page.

### Modified Capabilities
- `report-detail`: the API also returns the affected count and the caller's support state. The report page adds «También me afecta», «Aportar» and the aportes list, and shows the notice «Te sumaste al reporte» after joining.
- `report-creation`: «Seguir» on step 2 looks for similar reports before moving to «Revisar».
- `report-feed`: feed items carry the affected count, and the cards of Inicio show it.
- `report-search`: search results carry the affected count, so the cards of Buscar show it too.

## Impact

- Contract: `openapi/modules/reports.yaml` gains `findSimilarReports` (`POST /reports/similar`), `supportReport` / `unsupportReport` (`PUT`/`DELETE /reports/{id}/support`), and `listReportContributions` / `createReportContribution` (`GET`/`POST /reports/{id}/contributions`). `Report` gains `affectedCount`, `contributionCount` and `support`. This regenerates `backend/internal/reports/api/generated.go` and `frontend/app/api/`.
- Database: a new migration enables `vector` and adds `reports.embedding` and the `report_supports` and `report_contributions` tables. New queries go in `reports.sql`, and sqlc is regenerated with an override for `vector`.
- Backend: a Voyage AI client in `internal/platform/embedding`, which is configured by the `VOYAGE_APIKEY` value that already exists. It also covers similar search, support and aportes in the repository, service and handler of `internal/reports`, plus a background job that embeds reports published while Voyage was unreachable. New dependency: `github.com/pgvector/pgvector-go`.
- Frontend: `ReportStepPhotoAndPlace.vue` («Seguir» runs the check), new dialog and join page components, `pages/reports/[id].vue` (support button, «Aportar», aportes list, «Te sumaste» notice), the draft store, which builds «Tu aporte», and `ReportCard.vue`, which shows the affected count.
- Operations: similar reports only work with a valid `VOYAGE_APIKEY`. Without it the check returns nothing and publishing works as before.
