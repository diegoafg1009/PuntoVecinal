<script setup lang="ts">
import { useMutation, useQueryClient } from '@tanstack/vue-query'
import { supportReport, unsupportReport } from '~/api'
import { getReportQueryKey } from '~/api/@tanstack/vue-query.gen'
import type { Report, SupportState } from '~/api/types.gen'

const props = defineProps<{ report: Report }>()

const toast = useToast()
const queryClient = useQueryClient()

const supported = computed(() => props.report.support !== 'none')
const isAuthor = computed(() => props.report.support === 'author')

const queryKey = computed(() => getReportQueryKey({ path: { id: props.report.id } }))

function patch(affectedCount: number, support: SupportState) {
  queryClient.setQueryData<Report>(queryKey.value, old => old && { ...old, affectedCount, support })
}

const { mutate, isPending } = useMutation({
  mutationFn: async (on: boolean) => {
    const call = on ? supportReport : unsupportReport
    const { data, error } = await call({ path: { id: props.report.id } })
    if (error !== undefined || !data) throw new Error('Support request failed')
    return data
  },
  onMutate: (on) => {
    const previous = { affectedCount: props.report.affectedCount, support: props.report.support }
    patch(previous.affectedCount + (on ? 1 : -1), on ? 'supporter' : 'none')
    return previous
  },
  onSuccess: data => patch(data.affectedCount, data.support),
  onError: (_error, _on, previous) => {
    if (previous) patch(previous.affectedCount, previous.support)
    toast.add({ title: 'No pudimos guardar tu apoyo. Inténtalo de nuevo.', color: 'error' })
  }
})

// A tap while a request is pending is ignored, so the final state is the API's.
function toggle() {
  if (isPending.value || isAuthor.value) return
  mutate(!supported.value)
}
</script>

<template>
  <UButton
    :label="`También me afecta · ${report.affectedCount}`"
    icon="i-lucide-users"
    :color="supported ? 'primary' : 'neutral'"
    :variant="supported ? 'soft' : 'outline'"
    :disabled="isAuthor"
    :aria-pressed="supported"
    @click="toggle"
  />
</template>
