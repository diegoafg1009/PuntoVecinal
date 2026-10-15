<script setup lang="ts">
const route = useRoute()

// Focused screens (such as Reportar) hide the mobile bars; the desktop header stays.
const focusedOnMobile = computed(() => route.meta.focusedOnMobile === true)
</script>

<template>
  <div class="flex min-h-screen flex-col bg-default lg:bg-elevated">
    <AppHeader class="hidden lg:flex" />
    <AppTopBar v-if="!focusedOnMobile" class="lg:hidden" />

    <main class="flex-1" :class="{ 'pb-20 lg:pb-0': !focusedOnMobile }">
      <UContainer class="lg:pt-6 lg:pb-10" :ui="{ base: 'px-0 sm:px-0 lg:px-8' }">
        <slot />
      </UContainer>
    </main>

    <AppBottomBar v-if="!focusedOnMobile" class="lg:hidden" />
  </div>
</template>
