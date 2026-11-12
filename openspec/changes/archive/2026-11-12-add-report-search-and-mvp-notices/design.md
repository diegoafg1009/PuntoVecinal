# Design

## Context

- The feed (`GET /reports`, `listReports`) already does keyset pagination on `(created_at, id)`. It has an optional PostGIS distance returned as `distanceMeters`, and the opaque cursor is base64 (`encodeCursor`/`decodeCursor` in `internal/reports/service.go`). Items are `ReportSummary` and pages are `ReportPage` (`openapi/modules/reports.yaml`).
- Inicio (`pages/index.vue`) uses `useInfiniteQuery` with `ReportCard` and `useNeighborLocation`, which asks for the location only from `share()` and silently restores it on later visits while the browser still grants the permission.
- `search.vue`, `notices.vue` and `map.vue` render `ComingSoon`. `AppBottomBar.vue` and `AppHeader.vue` hold their own item lists, with Mapa linking to `/map`.
- The emergency notice is a `UAlert` inside `ReportStepWhatHappens.vue` (step 1 only). Its text is «Llama a Serenazgo o al 105».
- The session store keeps the Supabase `Session`, whose `user.is_anonymous` tells whether the neighbor has an account. All sessions are anonymous today.
- The pilot has a single zone with a few hundred reports at most.
- Figma (file `sC2o7u0tojnA5XjhlrE9Ts`) may have frames for Buscar and Avisos. Because of the daily call limit, the implementer asks the user for the frame links before building those screens. Without frames, the screens follow the layout of Inicio described below.

## Goals / Non-Goals

**Goals:**
- One search operation that covers words, closeness, or both, with the same items and pagination guarantees as the feed.
- Reuse the feed's card, location composable, formatting and infinite-loading pattern on Buscar.
- Small, isolated frontend changes for Mapa, Avisos and the emergency notice, easy to undo when the map, accounts and notices arrive.

**Non-Goals:**
- Full-text ranking, stemming, typo tolerance or a search index.
- Changing `GET /reports` or the Inicio screen.

## Decisions

### A separate `GET /reports/search` operation (`searchReports`)

Parameters: `q` (string, max 100), `sort` (`recent` | `distance`, default `recent`), `latitude`/`longitude` (both or neither, same ranges as the feed), `cursor` and `limit` (1–50, default 20). The response is the existing `ReportPage` with `ReportSummary` items. The response codes are 200, 400 (text, the same style as `listReports`) and 401 (`common.yaml#/Unauthorized`). The service applies these rules and answers 400 when one fails:

- `q` blank after trimming and `sort=recent` → 400.
- `sort=distance` without a location → 400.
- A cursor issued for the other order → 400.

The route `/reports/search` coexists with `/reports/{id}` because chi matches static segments before parameters. A handler test covers it.

*Alternative:* add `q` and `sort` to `GET /reports`. Rejected: the feed spec says its order never depends on the caller, and the feed and the search would share one operation with different validation rules. A separate operation keeps the `report-feed` spec and Inicio untouched and gives the frontend its own query helpers (`searchReportsInfiniteOptions`).

### Word matching with `unaccent` and `ILIKE`

The service splits `q` on whitespace and drops empty words. It escapes `\`, `%` and `_` in each word, then sends the words as a `text[]`. The query requires every word to match:

```sql
WHERE NOT EXISTS (
  SELECT 1 FROM unnest(@words::text[]) AS w
  WHERE extensions.unaccent(r.title || ' ' || coalesce(r.details, ''))
        NOT ILIKE '%' || extensions.unaccent(w) || '%'
)
```

An empty array matches every report, which is the «cerca de mí» search without words. A new migration runs `create extension if not exists unaccent with schema extensions;`. The query only references the function, so sqlc needs nothing more.

*Alternative:* Postgres full-text search (`to_tsvector('spanish', …)`). Rejected for now: stemming makes «poste» match «postes» but not substrings like «Jr.», and the story asks for «coincide con lo escrito». *Alternative:* a trigram index. Not needed at pilot size (see Risks).

### Two sqlc queries, one per order, sharing the feed's projection

- `SearchReportsRecent`: the same select as `ListReports` (cover photo join, `-1` distance sentinel when there is no location), plus the word filter, with keyset on `(created_at, id) <` and the order `created_at DESC, id DESC`.
- `SearchReportsNearest`: the location is required and the distance is computed as `ST_Distance(...)`. The keyset is `(distance, created_at, id)`, where a row comes after the cursor when `distance > d`, or when `distance = d AND (created_at, id) < (c, i)`. The order is `distance ASC, created_at DESC, id DESC`. The cursor stores the exact float distance, not the rounded meters, so ties at the same rounded value are not skipped. The result still exposes `distanceMeters` rounded to whole meters.

Both queries fetch `limit + 1` rows to detect a next page, as `List` does. The repository exposes `Search(ctx, SearchParams)` and returns the existing summary rows. The service reuses the cover photo signing of `List`, moved into a shared helper, so search results are built the same way as feed items.

*Alternative:* one query with `CASE` in the `ORDER BY`. Rejected: keyset conditions differ per order, and sqlc output stays readable with two queries.

### Cursor format carries the order

The search cursor is base64 JSON `{"o":"recent"|"distance","t":createdAt,"i":id,"d":distance?}`. `decodeSearchCursor` rejects a malformed value, a cursor whose `o` differs from the requested `sort`, and a feed cursor, each with `ErrInvalidCursor`. The feed cursor format is left as it is.

### Buscar screen

- Layout: the same centered panel as Inicio (max 560 px, title «Buscar» on desktop). Below it go a `UInput` (`type="search"`, icon `i-lucide-search`, placeholder «Busca por calle o problema», `maxlength=100`) inside a form that submits on Enter or with a «Buscar» button, and a «Cerca de mí» toggle (`USwitch`, or a chip as in Figma).
- State lives in the route query: `/search?q=poste&near=1`. Submitting does `router.replace` with the new query, so a search does not add a history entry for every submit. The infinite query is keyed on `(q, near, location)` and enabled only when the route has `q` or `near=1`. The input is initialized from `route.query.q`.
- «Cerca de mí»: turning it on calls `useNeighborLocation().share()`. On success it sets `near=1` and searches with `sort=distance` and the location. On failure it turns the toggle back off, and the toast from `share()` already says the location could not be used. On mount, Buscar always calls `restore()`, as Inicio does. If the page opened with `near=1` and no location is available afterwards, it drops `near` from the query.
- Without «Cerca de mí», Buscar still sends the location when `useNeighborLocation` already has one, so cards show distances, as on Inicio.
- States: skeletons while the first page loads, then «No encontramos reportes para «{q}».» (or «No hay reportes cerca de ti.» for a search without words), and on failure «No pudimos hacer la búsqueda.» with «Reintentar». The end-of-list sentinel and the next-page error work as on Inicio. Before the first search only the form is shown.
- `staleTime` and `gcTime` match Inicio, because the cached pages hold signed photo links that expire after one hour.

### Mapa not available

`AppBottomBar` and `AppHeader` render Mapa as a button, not a link. It is styled muted, uses `aria-disabled="true"`, and shows a small «Pronto» badge (mobile: a dot or a tiny label under the icon; desktop: a `UBadge` next to the label). On click it calls `useToast().add({ title: 'Próximamente', description: 'El mapa estará disponible pronto.' })` and does not navigate. In the desktop `UNavigationMenu`, the Mapa item gets an `onSelect` handler with no `to`. `pages/map.vue` becomes a page whose `definePageMeta` middleware returns `navigateTo('/', { replace: true })`. That keeps the route, so the redirect works with SSR and with client navigation, and the route can simply be restored later.

*Alternative:* remove `pages/map.vue` and add a `routeRules` redirect. Rejected: the redirect would only apply on the server, and the page is the natural place to restore the map.

### Emergency notice component

A new `ReportEmergencyNotice.vue` holds a `UAlert` (error color, subtle, `i-lucide-triangle-alert`, no `close`). Its title is «¿Es una emergencia o un tema de seguridad?» and its description is «Esta app no atiende urgencias. Llama a Serenazgo o a:», followed by three inline links: `<a href="tel:105">105 Policía</a>`, `tel:116` «116 Bomberos» and `tel:106` «106 SAMU». The numbers live in a constant in the component. `pages/report.vue` renders the notice once, between the stepper and the step component, so every step shows it. The `UAlert` is removed from `ReportStepWhatHappens.vue`. «Fijo» is read as always shown at the top of every step and impossible to close, not as sticky positioning. The review step's community-rules line that mentions 105 stays as it is.

Serenazgo is named without a number, as the user decided, until the Comas Serenazgo number is confirmed. Adding it later means adding one entry to the constant.

### Avisos for a neighbor without an account

The session store gains `isAnonymous = computed(() => session.value?.user.is_anonymous ?? false)`. The local Supabase claim is enough, so no `/me` call is needed. When `isAnonymous` is true, `pages/notices.vue` shows a centered empty state: `i-lucide-bell-off` icon, «Los avisos necesitan una cuenta», and «Con una cuenta podrás recibir novedades de los reportes que sigues. La creación de cuentas estará disponible próximamente.» It has no buttons. Otherwise it keeps `ComingSoon`, a branch no neighbor can reach today.

## Risks / Trade-offs

- [Sequential scan on every search because `unaccent(...) ILIKE` cannot use an index] → acceptable at pilot size (hundreds of rows). If search gets slow, add an immutable `unaccent` wrapper and a `pg_trgm` GIN index in a later migration, without any contract change.
- [Distance ordering computes `ST_Distance` for every matching row before limiting] → same pilot-size argument, since the existing GiST index on `location` is not used for this order. A KNN `<->` order can replace it later.
- [A cursor reused with a different location gives odd but valid pages] → the location is part of the frontend query key, so a new location always starts from the first page. The API only checks that the order matches.
- [`unaccent` is missing in an environment] → the migration creates it, and Supabase ships it, so local `make db-reset` and hosted projects both get it.
- [Mapa looking like a broken link] → muted style, the «Pronto» mark, `aria-disabled`, and the toast make the state explicit.

## Migration Plan

Additive. Apply the migration (it only enables `unaccent`), deploy the API with the new operation, then the frontend. Rollback is reverting the frontend and the API. The extension can stay enabled.
