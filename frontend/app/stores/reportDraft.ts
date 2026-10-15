const STORAGE_KEY = 'pv:report-draft'

export const TITLE_MAX_LENGTH = 100
export const DETAILS_MAX_LENGTH = 1000
export const STEP_COUNT = 3

export interface Step1Errors {
  category?: string
  title?: string
}

interface StoredDraft {
  category: CategorySlug | null
  title: string
  details: string
  step: number
}

/** Reads the stored draft, ignoring anything unknown or malformed. */
function readStored(): Partial<StoredDraft> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const data: unknown = JSON.parse(raw)
    if (typeof data !== 'object' || data === null) return {}
    const { category, title, details, step } = data as Record<string, unknown>
    return {
      ...(isCategorySlug(category) && { category }),
      ...(typeof title === 'string' && { title: title.slice(0, TITLE_MAX_LENGTH) }),
      ...(typeof details === 'string' && { details: details.slice(0, DETAILS_MAX_LENGTH) }),
      ...(Number.isInteger(step) && (step as number) >= 1 && (step as number) <= STEP_COUNT && { step: step as number })
    }
  } catch {
    return {}
  }
}

function writeStored(draft: StoredDraft) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(draft))
  } catch {
    // Storage is blocked or full: the form keeps working, it just cannot be restored later.
  }
}

export const useReportDraftStore = defineStore('reportDraft', () => {
  const stored = readStored()

  const category = ref<CategorySlug | null>(stored.category ?? null)
  const title = ref(stored.title ?? '')
  const details = ref(stored.details ?? '')
  const step = ref(stored.step ?? 1)

  watch([category, title, details, step], () => {
    writeStored({ category: category.value, title: title.value, details: details.value, step: step.value })
  })

  function validateStep1(): Step1Errors {
    const errors: Step1Errors = {}
    if (!category.value) errors.category = 'Elige una categoría.'
    if (!title.value.trim()) errors.title = 'Cuéntanos qué pasa.'
    return errors
  }

  /** True when the neighbor wrote or chose something worth keeping or discarding. */
  const hasData = computed(() => category.value !== null || title.value.trim() !== '' || details.value.trim() !== '')

  /** Empties the draft; the watcher above overwrites the stored copy with the empty one. */
  function reset() {
    category.value = null
    title.value = ''
    details.value = ''
    step.value = 1
  }

  return { category, title, details, step, hasData, validateStep1, reset }
})
