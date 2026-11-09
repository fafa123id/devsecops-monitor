
<script setup lang="ts">
const route = useRoute();

defineProps<{
  mobileOpen: boolean;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const navigation = [
  {
    label: "OVERVIEW",
    items: [
      { name: "Dashboard", path: "/", icon: "◫" },
    ],
  },
  {
    label: "MONITORING",
    items: [
      { name: "Projects", path: "/projects", icon: "▦" },
      { name: "Vulnerabilities", path: "/vulnerabilities", icon: "⬡" },
      { name: "Scan History", path: "/scans", icon: "◷" },
    ],
  },
  {
    label: "SYSTEM",
    items: [
      { name: "Agents", path: "/agents", icon: "⌘" },
      { name: "Settings", path: "/settings", icon: "⚙" },
    ],
  },
];

function isActive(path: string) {
  return route.path === path;
}

function closeMobile() {
  emit("close");
}
</script>

<template>
  <aside
    :class="[
      'fixed inset-y-0 left-0 z-50 flex h-screen w-[270px] shrink-0 flex-col border-r border-slate-800 bg-[#0d1422] transition-transform duration-300 ease-in-out lg:sticky lg:top-0 lg:z-20 lg:w-[250px] lg:translate-x-0',
      mobileOpen ? 'translate-x-0' : '-translate-x-full',
    ]"
  >
    <!-- Brand -->
    <div class="flex h-[76px] shrink-0 items-center gap-3 border-b border-slate-800 px-5 sm:px-6">
      <div
        class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-sky-500/15 text-xl text-sky-400"
      >
        ⬡
      </div>

      <div class="min-w-0">
        <h1 class="whitespace-nowrap text-sm font-bold tracking-wide text-white">
          DevSecOps<span class="text-sky-400">Monitor</span>
        </h1>

        <p class="mt-0.5 text-[10px] tracking-[0.18em] text-slate-500">
          SECURITY OPERATIONS
        </p>
      </div>

      <!-- Close button: mobile only -->
      <button
        type="button"
        aria-label="Close navigation"
        class="ml-auto flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-slate-400 transition hover:bg-slate-800 hover:text-white lg:hidden"
        @click="closeMobile"
      >
        <svg
          class="h-5 w-5"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.8"
          stroke-linecap="round"
        >
          <path d="m18 6-12 12M6 6l12 12" />
        </svg>
      </button>
    </div>

    <!-- Navigation -->
    <nav class="min-h-0 flex-1 space-y-7 overflow-y-auto overscroll-contain px-4 py-6 sm:py-7">
      <div v-for="group in navigation" :key="group.label">
        <p class="mb-3 px-3 text-[10px] font-semibold tracking-[0.16em] text-slate-500">
          {{ group.label }}
        </p>

        <div class="space-y-1">
          <NuxtLink
            v-for="item in group.items"
            :key="item.path"
            :to="item.path"
            class="flex items-center gap-3 rounded-lg px-3 py-2.5 text-[13px] transition-colors"
            :class="
              isActive(item.path)
                ? 'bg-sky-500/10 font-medium text-sky-400'
                : 'text-slate-400 hover:bg-slate-800/70 hover:text-white'
            "
            @click="closeMobile"
          >
            <span class="w-5 shrink-0 text-center text-base">
              {{ item.icon }}
            </span>

            <span>{{ item.name }}</span>

            <span
              v-if="isActive(item.path)"
              class="ml-auto h-1.5 w-1.5 shrink-0 rounded-full bg-sky-400"
            />
          </NuxtLink>
        </div>
      </div>
    </nav>

    <!-- Agent status -->
    <div class="shrink-0 border-t border-slate-800 p-4">
      <div class="rounded-xl border border-slate-800 bg-slate-900/70 p-3">
        <div class="flex items-center gap-2">
          <span class="relative flex h-2 w-2 shrink-0">
            <span
              class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-40"
            />
            <span
              class="relative inline-flex h-2 w-2 rounded-full bg-emerald-400"
            />
          </span>

          <span class="text-xs font-medium text-slate-300">
            Agent service
          </span>

          <span class="ml-auto text-[10px] text-emerald-400">
            Online
          </span>
        </div>

        <p class="mt-2 text-[10px] text-slate-500">
          Security monitoring system
        </p>
      </div>

      <p class="mt-4 text-center text-[10px] text-slate-600">
        DevSecOps Monitor · v1.0.0
      </p>
    </div>
  </aside>
</template>
