import { useMutation, useQueryClient } from '@tanstack/vue-query'
import { createReportContribution } from '~/api'
import { getReportQueryKey, listReportContributionsQueryKey } from '~/api/@tanstack/vue-query.gen'

const TOO_MANY_REQUESTS = 429

export interface JoinInput {
  text: string
  authorName?: string
  photo?: Blob
}

class JoinError extends Error {
  constructor(readonly status: number) {
    super(`Joining failed with status ${status}`)
  }
}

/**
 * Joins a report: sends the aporte (which also marks the neighbor's support), refreshes the
 * report, opens it and confirms with a toast. `onJoined` runs first, e.g. to empty the Reportar draft.
 */
export function useJoinReport(reportId: () => string, onJoined?: () => void) {
  const toast = useToast()
  const queryClient = useQueryClient()
  const errorMessage = ref('')

  const { mutate, isPending } = useMutation({
    mutationFn: async (input: JoinInput) => {
      const { data, error, response } = await createReportContribution({
        path: { id: reportId() },
        body: {
          text: input.text,
          ...(input.authorName && { authorName: input.authorName }),
          ...(input.photo && { photo: input.photo })
        }
      })
      if (error !== undefined || !data) throw new JoinError(response?.status ?? 0)
      return data
    },
    onSuccess: async (created) => {
      const id = reportId()
      queryClient.invalidateQueries({ queryKey: getReportQueryKey({ path: { id } }) })
      queryClient.invalidateQueries({ queryKey: listReportContributionsQueryKey({ path: { id } }) })
      onJoined?.()
      await navigateTo(`/reports/${id}`, { replace: true })
      toast.add({
        title: 'Te sumaste al reporte',
        description: `Ahora a ${created.affectedCount} vecinos les afecta. Tu aporte ya aparece abajo.`,
        icon: 'i-lucide-circle-check',
        color: 'success'
      })
    },
    onError: (error) => {
      errorMessage.value = error instanceof JoinError && error.status === TOO_MANY_REQUESTS
        ? 'Aportaste varias veces seguidas. Espera unos minutos antes de volver a intentarlo.'
        : 'No pudimos sumarte al reporte. Inténtalo de nuevo.'
    }
  })

  function join(input: JoinInput) {
    errorMessage.value = ''
    mutate(input)
  }

  return { join, isPending, errorMessage }
}
