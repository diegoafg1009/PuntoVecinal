import type { ReportStatus } from '~/api/types.gen'

const STATUS_LABELS: Record<ReportStatus, string> = {
  publicado: 'Publicado',
  derivado: 'Derivado',
  recibido: 'Recibido por la entidad',
  en_atencion: 'En atención',
  atendido: 'Atendido según la entidad',
  cerrado: 'Cerrado'
}

type BadgeColor = 'neutral' | 'info' | 'warning' | 'success'

const STATUS_COLORS: Record<ReportStatus, BadgeColor> = {
  publicado: 'neutral',
  derivado: 'info',
  recibido: 'info',
  en_atencion: 'warning',
  atendido: 'success',
  cerrado: 'neutral'
}

// Nuxt UI CSS variables, so the pins follow the theme. Derivado and Recibido share a color, as their badges do.
// Publicado is red (a new, unattended problem): the theme's primary is the same green as success (Atendido).
const STATUS_PIN_COLORS: Record<ReportStatus, string> = {
  publicado: 'var(--ui-error)',
  derivado: 'var(--ui-info)',
  recibido: 'var(--ui-info)',
  en_atencion: 'var(--ui-warning)',
  atendido: 'var(--ui-success)',
  cerrado: 'var(--ui-text-dimmed)'
}

const MINUTE = 60_000
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

const dateFormat = new Intl.DateTimeFormat('es-PE', { day: 'numeric', month: 'short', year: 'numeric' })

export function statusLabel(status: ReportStatus): string {
  return STATUS_LABELS[status]
}

export function statusColor(status: ReportStatus): BadgeColor {
  return STATUS_COLORS[status]
}

/** Fill color of a report's point on the map. */
export function statusPinColor(status: ReportStatus): string {
  return STATUS_PIN_COLORS[status]
}

function startOfDay(date: Date): number {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime()
}

/** How long ago a report was published: relative when under 7 days, a date otherwise. */
export function publishedAgo(date: string | Date, now: Date = new Date()): string {
  const published = typeof date === 'string' ? new Date(date) : date
  const elapsed = Math.max(0, now.getTime() - published.getTime())

  if (elapsed < MINUTE) return 'hace un momento'
  if (elapsed < HOUR) return `hace ${Math.floor(elapsed / MINUTE)} min`

  const calendarDays = Math.round((startOfDay(now) - startOfDay(published)) / DAY)
  if (calendarDays === 0) return `hace ${Math.floor(elapsed / HOUR)} h`
  if (calendarDays === 1) return 'ayer'
  if (elapsed < 7 * DAY) return `hace ${calendarDays} días`

  // The month abbreviation varies by browser («nov» / «nov.»); drop the dot.
  return dateFormat.format(published).replace(/\./g, '')
}

/** Distance for a card: tens of meters under 1 km, kilometers with a comma after that. */
export function formatDistance(meters: number): string {
  if (meters < 1000) return `${Math.max(10, Math.round(meters / 10) * 10)} m`
  return `${(meters / 1000).toFixed(1).replace('.', ',')} km`
}

const EARTH_RADIUS_METERS = 6_371_000

/** Straight-line distance in whole meters between two coordinates (haversine). */
export function distanceBetween(lat1: number, lng1: number, lat2: number, lng2: number): number {
  const rad = (degrees: number) => degrees * Math.PI / 180
  const a = Math.sin(rad(lat2 - lat1) / 2) ** 2
    + Math.cos(rad(lat1)) * Math.cos(rad(lat2)) * Math.sin(rad(lng2 - lng1) / 2) ** 2
  return Math.round(2 * EARTH_RADIUS_METERS * Math.asin(Math.sqrt(a)))
}
