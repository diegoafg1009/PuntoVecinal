<script setup lang="ts">
defineProps<{ size?: 'sm' | 'md' }>()
</script>

<template>
  <NuxtLink to="/profile" aria-label="Perfil" class="inline-flex">
    <UAvatar icon="i-lucide-user" :size="size ?? 'sm'" />
  </NuxtLink>
</template>
