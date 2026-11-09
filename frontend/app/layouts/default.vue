
<script setup lang="ts">
const route = useRoute();
const mobileMenuOpen = ref(false);

watch(
  () => route.fullPath,
  () => {
    mobileMenuOpen.value = false;
  },
);
</script>

<template>
  <div class="flex min-h-screen w-full overflow-x-hidden bg-[#090d16]">
    <AppSidebar
      :mobile-open="mobileMenuOpen"
      @close="mobileMenuOpen = false"
    />

    <!-- Mobile backdrop -->
    <button
      v-if="mobileMenuOpen"
      type="button"
      aria-label="Close navigation menu"
      class="fixed inset-0 z-40 bg-black/60 backdrop-blur-[2px] lg:hidden"
      @click="mobileMenuOpen = false"
    />

    <div class="flex min-h-screen min-w-0 flex-1 flex-col">
      <AppHeader @toggle-menu="mobileMenuOpen = !mobileMenuOpen" />

      <main class="min-w-0 flex-1 overflow-x-hidden p-4 sm:p-6 lg:p-8">
        <slot />
      </main>
    </div>
  </div>
</template>
