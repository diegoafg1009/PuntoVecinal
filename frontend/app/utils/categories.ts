export const categories = [
  { slug: 'limpieza', label: 'Limpieza' },
  { slug: 'alumbrado', label: 'Alumbrado' },
  { slug: 'pistas-y-veredas', label: 'Pistas y veredas' },
  { slug: 'agua-y-desague', label: 'Agua y desagüe' },
  { slug: 'otro', label: 'Otro' }
] as const

export type CategorySlug = typeof categories[number]['slug']

export function isCategorySlug(value: unknown): value is CategorySlug {
  return categories.some(category => category.slug === value)
}
