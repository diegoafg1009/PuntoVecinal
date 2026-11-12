<script setup lang="ts">
// Official numbers. A Serenazgo number goes here once the one for Comas is confirmed.
const numbers = [
  { number: '105', label: 'Policía' },
  { number: '116', label: 'Bomberos' },
  { number: '106', label: 'SAMU' }
]
</script>

<template>
  <UAlert
    color="error"
    variant="subtle"
    icon="i-lucide-triangle-alert"
    title="¿Es una emergencia o un tema de seguridad?"
  >
    <template #description>
      Esta app no atiende urgencias. Llama a Serenazgo o a:
      <template v-for="(item, index) in numbers" :key="item.number">
        <template v-if="index > 0">, </template>
        <a :href="`tel:${item.number}`" class="font-semibold underline">{{ item.number }} {{ item.label }}</a>
      </template>
    </template>
  </UAlert>
</template>
