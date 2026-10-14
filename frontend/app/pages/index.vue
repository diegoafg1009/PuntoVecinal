<script setup lang="ts">
const { data: me, isPending, isError } = useMe()
</script>

<template>
  <UContainer class="py-10">
    <UCard>
      <template #header>
        <h1 class="text-2xl font-bold">
          Punto Vecinal
        </h1>
      </template>

      <p v-if="isPending">
        Entrando…
      </p>
      <p v-else-if="isError">
        No pudimos conectar con el servidor.
      </p>
      <div v-else-if="me">
        <p>{{ me.isAnonymous ? 'Entraste como invitado.' : 'Entraste con tu cuenta.' }}</p>
        <p class="mt-2 text-sm text-muted" data-testid="user-id">
          {{ me.id }}
        </p>
      </div>
    </UCard>
  </UContainer>
</template>
