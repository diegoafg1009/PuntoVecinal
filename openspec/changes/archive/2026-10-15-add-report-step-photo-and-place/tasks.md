# Tasks

## 1. Setup

- [x] 1.1 [frontend] Add `leaflet` and `@types/leaflet` with `bun add`; verify `bun run typecheck` still passes and both appear in `frontend/package.json`.

## 2. Photo helpers

- [x] 2.1 [frontend] Create `app/utils/photo.ts` that decodes a file, scales its long side to at most 1600 px and re-encodes it as JPEG (quality 0.8), rejecting files that are not images; verify in the browser that a phone photo with GPS EXIF comes out smaller, upright and without EXIF (inspect the output blob), and that a text file is rejected.
- [x] 2.2 [frontend] Create `app/utils/photoStorage.ts` (IndexedDB `pv` / `report-photos`: put, remove, getAll, clear) with every access in try/catch; verify in the browser that photos survive a reload and that a failing IndexedDB (blocked in devtools) leaves the form working.

## 3. Draft store

- [x] 3.1 [frontend] Extend `app/stores/reportDraft.ts` with `photos` (max 3, `PHOTO_MAX_COUNT`), `location`, `reference` (`REFERENCE_MAX_LENGTH` 150), `photoIds` in the stored draft with a defensive read, `addPhotos`, `removePhoto`, `setLocation`, `validateStep2` (messages «Agrega al menos una foto.» and «Marca el lugar en el mapa.»), and photos/location/reference counted by `hasData` and cleared by `reset`; verify by loading a stored draft with malformed values and an old draft without the new fields, and checking both start clean without errors.

## 4. Step 2 UI

- [x] 4.1 [frontend] Create `app/components/ReportLocationMap.vue` (Leaflet + OpenStreetMap, draggable pin as `L.divIcon`, tap-to-move, attribution shown, client only); verify in the browser at mobile and desktop widths that dragging or tapping moves the pin and the position is emitted.
- [x] 4.2 [frontend] Add the «Usar mi ubicación» button to the step (requests the device position only on tap, moves the pin and sets the location, error message on denial/failure); verify in the browser that nothing is requested on open, and that the button works with permission allowed (pin at position), blocked (message, manual placement works) and with a location already marked (it is replaced).
- [x] 4.3 [frontend] Create `app/components/ReportStepPhotoAndPlace.vue` with the photo picker, thumbnails with «Quitar foto», the 3-photo limit message, the map with the hint, the optional «Referencia» field, «Seguir ›» and «‹ Atrás»; verify every scenario of the Step 2 requirements in `specs/report-creation/spec.md` in the browser (add/remove, fourth photo, several photos over the limit, non-image file, no photo, no location, complete, go back keeps data).
- [x] 4.4 [frontend] Render the new step for `draft.step === 2` in `app/pages/report.vue` (the placeholder stays for step 3); verify «Seguir» from step 1 opens step 2 and from step 2 opens the step 3 placeholder, and that the close dialog appears when only a photo or a location was entered.
- [x] 4.5 [frontend] Check draft persistence end to end: add two photos, move the pin, write a reference, reload and reopen Reportar (restored); discard (empty form on step 1); then block IndexedDB and verify text, location and reference return but photos do not and step 2 asks for them.

## 5. Final check

- [x] 5.1 [frontend] Run `bun run typecheck` and `openspec validate add-report-step-photo-and-place`; verify both pass, and confirm in the Network tab that no photo is sent to the server while filling the form.

## Workflow follow-up

- HU-10 must strip metadata from photos again on the server when it receives them.
- Update the HU-09 story in Jira (PV-18): the pilot-zone criterion was dropped from this change.
- Archive the change after review (`/opsx:archive`).
