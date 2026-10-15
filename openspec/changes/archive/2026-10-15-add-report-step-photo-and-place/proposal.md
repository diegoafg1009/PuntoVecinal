# Proposal

## Why

Step 2 of the Reportar form («Foto y lugar») is still a «Próximamente» placeholder, so a neighbor cannot attach the photos and the exact place that make a problem verifiable (HU-09, PV-18). Without it the form cannot reach step 3 or become a publishable report.

## What Changes

- Replace the step 2 placeholder with a real step: photo picker (take or choose, 1 to 3), thumbnails with a remove option, a map with a movable pin, and an optional location reference («frente al mercado»).
- Offer a «Usar mi ubicación» button. Only when the neighbor taps it does the browser ask for permission to read the device location; if they accept, the pin moves to their position, and if they decline (or it is unavailable), they place the pin by hand. Either way they can adjust the pin.
- Validate step 2 on «Seguir»: at least 1 photo, never more than 3, and a location chosen; otherwise a message in Spanish and the form stays on step 2.
- Process each photo on the device before keeping it: re-encode it (which drops the device location and other hidden file data) and shrink it to save mobile data.
- Keep the photos, pin and reference in the draft so they are restored when the neighbor comes back.
- Photos are NOT sent or stored on the server in this change; they are uploaded together with the report when it is published (a later story).
- Out of scope: the pilot-zone check of the original story («lugar fuera de la zona del piloto») was dropped by decision of the team; no zone is defined or validated. No API, database or backend change.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `report-creation`: adds the step 2 requirements (photos, place, validation, on-device photo processing) and extends the draft requirements so they cover photos, pin and reference; the current «Closing the form» requirement counts step 2 data as «something written».

## Impact

- `frontend/` only: new step component and photo/map helpers, `reportDraft` store (new fields, step 2 validation, IndexedDB for photo blobs), `pages/report.vue` (render step 2, «Seguir»/«Atrás»).
- New frontend dependency: Leaflet (OpenStreetMap tiles, no API key).
- Reuses no API; `openapi/`, `backend/` and `supabase/` are untouched.
