# Design

## Context

See proposal.md. Step 1 is done: `pages/report.vue` renders `ReportStepWhatHappens` for `draft.step === 1` and a «Próximamente» block otherwise; `stores/reportDraft.ts` (Pinia) holds `category`, `title`, `details` and `step` and mirrors them to `localStorage` with a defensive read. The app is a client-side SPA (`ssr: false`). There is no reports, photos or storage endpoint yet, and the team decided that photos are not saved on the server until the report is published, so this change is frontend-only: the contract, backend and migrations are untouched. Step 3 (HU-10) will read what this step leaves in the draft store and upload it.

## Goals / Non-Goals

**Goals:**
- Step 2 leaves in the draft store 1 to 3 clean, reduced photos, a location and an optional reference, ready for HU-10 to upload.
- The photo and location rules are testable without a server.

**Non-Goals:**
- Uploading or storing photos on the server, `POST /reports`, signed URLs.
- Pilot-zone validation (dropped from the story), searching an address, satellite imagery.
- Face and license-plate blurring (moderation, a different story).

## Decisions

### 1. Photos are processed on the device, before being kept
A helper (`app/utils/photo.ts`) decodes the chosen file with `createImageBitmap(file, { imageOrientation: 'from-image' })` (so the picture keeps its upright orientation once the metadata is gone), draws it on a canvas whose long side is at most 1600 px, and exports it with `canvas.toBlob('image/jpeg', 0.8)`. Re-encoding through a canvas drops EXIF (GPS included), XMP and any other hidden data by construction: only pixels are written. A typical 4–8 MB phone photo ends up around 200–500 KB. A file the browser cannot decode (not an image, or HEIC on a browser without support) rejects, and the step shows «No pudimos usar esa foto.».
*Alternative:* leave it for the backend (`backend/AGENTS.md` says photos are processed on the server). Rejected for this change because nothing is uploaded yet; the server should still strip metadata when it receives them in HU-10, as a second barrier, since a client cannot be trusted. Flag this to HU-10.
*Alternative:* a library (`browser-image-compression`, `exifr`). Rejected: a canvas does both jobs with no dependency.
Constants (assumptions, not in the story): 1600 px, JPEG quality 0.8, max 3 photos.

### 2. Picking photos
A visible `UButton` triggers a hidden `<input type="file" accept="image/*" multiple>`. Without `capture`, phones offer both «Tomar foto» and the gallery, covering «tomar o elegir». The files go through the helper one by one; the first ones that fit are added up to 3, the rest are dropped with the message «Un reporte lleva como máximo 3 fotos.». When 3 photos are in, the add button is disabled (still tappable cases such as a multi-select over the limit are covered by the message). Each thumbnail is an `<img>` on an object URL with a remove `UButton` (`aria-label="Quitar foto"`); the URL is revoked on remove and on reset.

### 3. Draft: text in localStorage, photo blobs in IndexedDB
`localStorage` cannot hold images (about 5 MB and strings only), so the photos' blobs go in a small IndexedDB store (`app/utils/photoStorage.ts`: put, delete, getAll, clear, on one database `pv` with store `report-photos`). The draft store keeps `photos` as `{ id, blob, url }[]` in memory, writes a blob on add, deletes it on remove, and on first use reads them back (the stored order is kept in the localStorage entry as `photoIds`). Photos load asynchronously, so the store exposes `ready` and the step shows the thumbnails when they arrive. Every IndexedDB access is wrapped in try/catch: if it fails the photos live only in memory, the form works, and the photos are not restored (spec scenario «Photos cannot be stored»).
*Alternative:* base64 inside `localStorage`. Rejected: 3 photos can exceed the quota and a write failure would also drop the text draft.
*Alternative:* memory only. Rejected: reloading the page, or the mobile browser discarding the tab while the neighbor goes to take a photo, would lose them, and the existing requirement says the draft survives that.
`hasData` also counts photos, location and reference, and `reset()` clears them (and the IndexedDB store).

### 4. Map: Leaflet with OpenStreetMap tiles
`app/components/ReportLocationMap.vue` creates a Leaflet map in `onMounted`, a draggable marker, and a tile layer from OpenStreetMap; it emits the position and is wrapped in `<ClientOnly>` (the app is already client-side, but this keeps Nuxt from importing Leaflet in any server pass). The marker uses an `L.divIcon` with a Lucide pin instead of Leaflet's default PNG icons, which break under bundlers. Tapping the map also moves the pin, which is easier than dragging on a phone. The component imports `leaflet/dist/leaflet.css`. Dependencies: `leaflet` and `@types/leaflet`.
*Alternative:* MapLibre GL (vector, clustering-ready, heavier) and Google Maps (API key and billing). Rejected: this step only needs a movable pin.
OSM's tile server is meant for light use and requires attribution (Leaflet shows it by default); fine for a university pilot, to swap for a provider with a key if traffic grows.

### 5. Location is required and starts unmarked
The pin is shown at a default center (an assumption: the Cooperativa Primavera area in Comas, around `-11.936, -77.050`, to be adjusted with the real coordinates), but the draft location stays `null` until the neighbor moves the pin or taps the map. «Seguir» without it shows «Marca el lugar en el mapa.». This keeps a report from being published at an arbitrary default spot. A hint under the map reads «Mueve el pin al lugar exacto.». A «Usar mi ubicación» button next to the map calls `navigator.geolocation.getCurrentPosition` (high accuracy off, 10 s timeout), which makes the browser show its own permission prompt; on success it sets the location, moves the pin there and zooms in, and on a denial, timeout or missing API it shows a short `UAlert` («No pudimos obtener tu ubicación. Márcala en el mapa.») and leaves the map as it was for manual placement. Nothing is requested when step 2 opens: asking only after a tap puts the prompt in context, so the neighbor knows why it appears and is less likely to deny it for good. A browser that remembered a denial will not prompt again; the same message appears and the neighbor can re-enable the permission in the browser settings. The button also works when a location is already marked and then replaces it. The position is only used to place the pin and is never sent anywhere in this change; the pin can always be adjusted because GPS can be off by tens of meters. Coordinates are stored as `{ lat, lng }` rounded to 6 decimals. The reference is an optional `UInput` («Referencia (opcional)», example «frente al mercado»), up to 150 characters (assumption).

### 6. Step component, validation and navigation
`app/components/ReportStepPhotoAndPlace.vue` follows `ReportStepWhatHappens`: a `UForm` with a `validate` function from the store (`validateStep2()` returns `photos` and `location` errors, shown as messages in Spanish: «Agrega al menos una foto.», «Marca el lugar en el mapa.»), `validate-on` empty so errors appear on «Seguir», and a `UFormField` per block. «Seguir ›» sets `draft.step = 3`; a secondary «‹ Atrás» sets `draft.step = 1`. Nothing entered is cleared when going back. `pages/report.vue` renders the new component for `draft.step === 2` and keeps the placeholder only for step 3. The step 1 «Seguir» is unchanged.

### 7. Draft shape
`StoredDraft` gains `location: { lat, lng } | null`, `reference: string` and `photoIds: string[]`; `readStored` validates each (finite numbers within lat/lng ranges, string cut to the limit, array of strings) and ignores anything else, so old drafts without these fields still load.

## Risks / Trade-offs

- [Stripped-by-canvas photos are re-compressed, so quality drops slightly and a second JPEG generation is created] → quality 0.8 at 1600 px is enough to identify a problem; the originals are never kept.
- [Large images can exhaust memory on old phones when decoded] → `createImageBitmap` with `resizeWidth/resizeHeight` can be used to decode at the target size if testing shows problems; process photos one at a time, not in parallel.
- [IndexedDB can be blocked (private mode) or evicted] → in-memory fallback and a clear message when step 2 comes back without photos (Decision 3).
- [Metadata is stripped only in the browser, which a client can skip] → HU-10 must strip it again on the server; recorded here so it is not lost.
- [OSM tile usage policy] → Decision 4.
- [Default map center is a guess] → easy to change; flagged for review.
- [A denied permission is remembered by the browser, so the button stops prompting] → the form never depends on it; manual placement always works, and the message points to the browser settings.
- [The original story's pilot-zone criterion is not implemented] → by decision of the team; the story in Jira still lists it and should be updated.
