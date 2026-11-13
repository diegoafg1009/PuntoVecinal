<script setup lang="ts">
// The «Mostrar mi nombre o apodo» switch and its input. Neighbors without an account can
// choose a name; a registered user's name arrives with accounts. Must live inside a UForm.
const showName = defineModel<boolean>('showName', { required: true })
const name = defineModel<string>('name', { required: true })

const session = useSessionStore()

const canChooseName = computed(() => session.session?.user.is_anonymous === true)
</script>

<template>
  <div v-if="canChooseName" class="flex flex-col gap-2">
    <USwitch v-model="showName" label="Mostrar mi nombre o apodo" />
    <UFormField v-if="showName" name="authorName" label="Nombre o apodo" :ui="{ label: 'font-semibold text-highlighted' }">
      <UInput v-model="name" :maxlength="AUTHOR_NAME_MAX_LENGTH" class="w-full" />
    </UFormField>
  </div>
</template>
