<script setup lang="ts">
// The map does not exist yet: the address sends the neighbor to Inicio.
definePageMeta({
  middleware: () => navigateTo('/', { replace: true })
})
</script>

<template>
  <div />
</template>
