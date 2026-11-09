
<script setup lang="ts">
const route = useRoute();

const emit = defineEmits<{
  (event: "toggle-menu"): void;
}>();

const { user, logout, loading } = useAuth();

const logoutError = ref("");

const pageTitles: Record<string, { title: string; description: string }> = {
  "/": {
    title: "Dashboard Overview",
    description: "Monitor your infrastructure security posture",
  },
  "/projects": {
    title: "Projects",
    description: "Manage and monitor Docker Compose projects",
  },
  "/vulnerabilities": {
    title: "Vulnerabilities",
    description: "Review and track detected security vulnerabilities",
  },
  "/scans": {
    title: "Scan History",
    description: "Track dependency security scan activities",
  },
  "/agents": {
    title: "Agents",
    description: "Monitor connected Docker agents",
  },
  "/settings": {
    title: "Settings",
    description: "Configure your monitoring environment",
  },
};

const page = computed(
  () =>
    pageTitles[route.path] || {
      title: "DevSecOps Monitor",
      description: "Security operations center",
    },
);

const currentDate = new Intl.DateTimeFormat("en-US", {
  weekday: "short",
  day: "numeric",
  month: "short",
  year: "numeric",
}).format(new Date());

const displayName = computed(
  () => user.value?.name || user.value?.username || "Administrator",
);

const displayUsername = computed(
  () => `@${user.value?.username || "administrator"}`,
);

const avatarInitials = computed(() => {
  return (
    displayName.value
      .trim()
      .split(/\s+/)
      .slice(0, 2)
      .map((part) => part.charAt(0).toUpperCase())
      .join("") || "U"
  );
});

async function handleLogout() {
  logoutError.value = "";

  try {
    await logout();

    await navigateTo("/login", {
      replace: true,
    });
  } catch (error) {
    console.error("Logout failed:", error);
    logoutError.value = "Logout failed. Please try again.";
  }
}
</script>

<template>
  <header
    class="relative flex min-h-[72px] w-full items-center justify-between gap-3 border-b border-slate-800 bg-[#0d1422] px-4 py-3 sm:min-h-[76px] sm:px-6 lg:gap-5 lg:px-8"
  >
    <!-- Page title and mobile menu -->
    <div class="flex min-w-0 items-center gap-3">
      <button
        type="button"
        aria-label="Open navigation menu"
        aria-controls="dashboard-sidebar"
        class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-slate-700 text-slate-300 transition hover:border-sky-500 hover:bg-sky-500/10 hover:text-sky-400 lg:hidden"
        @click="emit('toggle-menu')"
      >
        <svg
          class="h-5 w-5"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.8"
          stroke-linecap="round"
        >
          <path d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>

      <div class="min-w-0">
        <h2 class="truncate text-sm font-semibold tracking-tight text-white sm:text-lg">
          {{ page.title }}
        </h2>

        <p class="mt-1 hidden truncate text-xs text-slate-500 sm:block">
          {{ page.description }}
        </p>
      </div>
    </div>

    <!-- Header actions -->
    <div class="flex shrink-0 items-center gap-2 sm:gap-3 lg:gap-5">
      <!-- Current date -->
      <div class="hidden items-center gap-2 text-xs text-slate-400 lg:flex">
        <svg
          class="h-4 w-4 text-slate-500"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.6"
        >
          <rect x="3" y="5" width="18" height="16" rx="2" />
          <path d="M16 3v4M8 3v4M3 10h18" />
        </svg>

        {{ currentDate }}
      </div>

      <div class="hidden h-7 w-px bg-slate-800 lg:block" />

      <!-- Notifications -->
      <button
        type="button"
        class="relative hidden h-9 w-9 items-center justify-center rounded-lg border border-slate-700 text-slate-400 transition hover:border-slate-500 hover:text-white sm:flex"
        aria-label="Notifications"
      >
        <svg
          class="h-4 w-4"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.6"
        >
          <path
            d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
          <path d="M10 21h4" stroke-linecap="round" />
        </svg>

        <span
          class="absolute right-2 top-2 h-1.5 w-1.5 rounded-full bg-sky-400"
        />
      </button>

      <!-- User -->
      <div class="flex min-w-0 items-center gap-2 sm:gap-3">
        <div class="hidden min-w-0 text-right sm:block">
          <p class="max-w-[120px] truncate text-xs font-medium text-slate-200 lg:max-w-[160px]">
            {{ displayName }}
          </p>

          <p class="mt-0.5 truncate text-[10px] text-slate-500">
            {{ displayUsername }}
          </p>
        </div>

        <div
          class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-sky-500/30 bg-sky-500/10 text-xs font-semibold text-sky-400"
        >
          {{ avatarInitials }}
        </div>
      </div>

      <!-- Logout -->
      <button
        type="button"
        :disabled="loading"
        aria-label="Logout"
        title="Sign out"
        class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-slate-700 text-slate-400 transition hover:border-red-500/40 hover:bg-red-500/10 hover:text-red-400 disabled:cursor-not-allowed disabled:opacity-50"
        @click="handleLogout"
      >
        <svg
          class="h-4 w-4"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.7"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M10 17l5-5-5-5" />
          <path d="M15 12H3" />
          <path d="M12 3h6a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-6" />
        </svg>
      </button>
    </div>

    <!-- Logout error -->
    <div
      v-if="logoutError"
      class="absolute right-4 top-full z-[60] mt-2 max-w-[calc(100vw-2rem)] rounded-lg border border-red-500/30 bg-[#1c1420] px-3 py-2 text-xs text-red-400 shadow-xl sm:right-8"
    >
      {{ logoutError }}
    </div>
  </header>
</template>
