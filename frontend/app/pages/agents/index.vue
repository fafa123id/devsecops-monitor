
<script setup lang="ts">
interface Agent {
  id: number;
  name: string;
  hostname: string | null;
  ip_address: string | null;
  status: string;
  agent_version: string | null;
  last_heartbeat_at: string | null;
  registered_at: string | null;
  projects_count: number;
  containers_count: number;
}

interface AgentsResponse {
  message: string;
  data: Agent[];
  summary: {
    total: number;
    online: number;
    offline: number;
    pending: number;
  };
  meta: {
    total: number;
  };
}

const api = useApi();

const {
  data: response,
  pending,
  error,
  refresh,
} = await useAsyncData(
  "agents",
  () => api.get<AgentsResponse>("/api/agents"),
  {
    default: () => ({
      message: "",
      data: [],
      summary: {
        total: 0,
        online: 0,
        offline: 0,
        pending: 0,
      },
      meta: {
        total: 0,
      },
    }),
  },
);

const agents = computed(() => response.value?.data ?? []);
const summary = computed(() => response.value.summary);

const apiErrorMessage = computed(() => {
  const apiError = error.value as {
    data?: { message?: string };
    message?: string;
  } | null;

  return (
    apiError?.data?.message ||
    apiError?.message ||
    "Unable to load registered agents."
  );
});

function titleCase(value: string): string {
  if (!value) return "Unknown";

  return value.charAt(0).toUpperCase() + value.slice(1);
}

function statusClass(status: string): string {
  return (
    {
      online: "text-emerald-400",
      offline: "text-red-400",
      pending: "text-yellow-400",
    }[status] || "text-slate-400"
  );
}

function statusDot(status: string): string {
  return (
    {
      online: "bg-emerald-400",
      offline: "bg-red-400",
      pending: "bg-yellow-400",
    }[status] || "bg-slate-400"
  );
}

function formatDateTime(value: string | null): string {
  if (!value) return "Never";

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) return "—";

  return new Intl.DateTimeFormat("en-US", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

function formatRegisteredDate(value: string | null): string {
  if (!value) return "—";

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) return "—";

  return new Intl.DateTimeFormat("en-US", {
    day: "numeric",
    month: "short",
    year: "numeric",
  }).format(date);
}
</script>

<template>
  <div class="space-y-7">
    <!-- Heading -->
    <section class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <p class="text-xs font-medium uppercase tracking-[0.18em] text-sky-400">
          Infrastructure monitoring
        </p>

        <h1 class="mt-2 text-2xl font-semibold text-white">
          Agents
        </h1>

        <p class="mt-2 text-sm text-slate-400">
          Monitor registered Docker agents and their connectivity.
        </p>
      </div>

      <div class="flex items-center gap-2 rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-2.5">
        <span class="h-1.5 w-1.5 rounded-full bg-sky-400" />
        <span class="text-xs text-slate-400">
          Agent monitoring
        </span>
      </div>
    </section>

    <!-- Summary -->
    <section class="grid grid-cols-1 gap-4 sm:grid-cols-3">
      <div class="rounded-xl border border-slate-800 bg-[#111827] p-5">
        <p class="text-xs text-slate-400">
          Registered Agents
        </p>

        <p class="mt-3 text-2xl font-semibold text-white">
          {{ summary.total }}
        </p>
      </div>

      <div class="rounded-xl border border-slate-800 bg-[#111827] p-5">
        <p class="text-xs text-slate-400">
          Online Agents
        </p>

        <p class="mt-3 text-2xl font-semibold text-emerald-400">
          {{ summary.online }}
        </p>
      </div>

      <div class="rounded-xl border border-slate-800 bg-[#111827] p-5">
        <p class="text-xs text-slate-400">
          Offline Agents
        </p>

        <p class="mt-3 text-2xl font-semibold text-red-400">
          {{ summary.offline }}
        </p>

        <p class="mt-1 text-[10px] text-slate-500">
          {{ summary.pending }} pending registration
        </p>
      </div>
    </section>

    <!-- Agent list -->
    <section class="space-y-4">
      <div class="flex items-center justify-between gap-3">
        <div>
          <h2 class="text-base font-semibold text-white">
            Registered Agents
          </h2>

          <p class="mt-1 text-xs text-slate-500">
            Hosts connected to the monitoring platform
          </p>
        </div>

        <span class="shrink-0 text-xs text-slate-500">
          {{ agents.length }} agents
        </span>
      </div>

      <!-- Loading -->
      <div
        v-if="pending"
        class="rounded-xl border border-slate-800 bg-[#111827] px-6 py-14 text-center"
      >
        <div
          class="mx-auto h-6 w-6 animate-spin rounded-full border-2 border-slate-600 border-t-sky-400"
        />

        <p class="mt-4 text-sm text-slate-400">
          Loading agents...
        </p>
      </div>

      <!-- Error -->
      <div
        v-else-if="error"
        class="rounded-xl border border-red-500/20 bg-[#111827] p-6"
      >
        <p class="text-sm font-medium text-red-400">
          Failed to load agents
        </p>

        <p class="mt-2 text-xs text-slate-400">
          {{ apiErrorMessage }}
        </p>

        <button
          type="button"
          class="mt-4 rounded-lg border border-slate-700 px-4 py-2 text-xs font-medium text-slate-300 transition hover:border-sky-500 hover:text-sky-400"
          @click="refresh()"
        >
          Try again
        </button>
      </div>

      <!-- Empty state -->
      <div
        v-else-if="agents.length === 0"
        class="rounded-xl border border-slate-800 bg-[#111827] px-6 py-14 text-center"
      >
        <h3 class="text-sm font-semibold text-white">
          No agents registered
        </h3>

        <p class="mx-auto mt-2 max-w-sm text-xs leading-5 text-slate-500">
          No Docker agents have been registered with the monitoring platform yet.
        </p>
      </div>

      <!-- Agent cards -->
      <template v-else>
        <article
          v-for="agent in agents"
          :key="agent.id"
          class="overflow-hidden rounded-xl border border-slate-800 bg-[#111827]"
        >
          <!-- Agent header -->
          <div class="flex flex-wrap items-start justify-between gap-4 p-5">
            <div class="flex min-w-0 items-center gap-3">
              <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-sky-500/15 bg-sky-500/10 text-lg text-sky-400">
                ⬡
              </div>

              <div class="min-w-0">
                <h3 class="truncate text-sm font-semibold text-white">
                  {{ agent.name }}
                </h3>

                <p class="mt-1 break-all text-xs text-slate-500">
                  {{ agent.hostname || "Unknown hostname" }}
                  · ID #{{ agent.id }}
                </p>
              </div>
            </div>

            <div class="flex shrink-0 items-center gap-1.5 pt-1">
              <span
                class="h-1.5 w-1.5 rounded-full"
                :class="statusDot(agent.status)"
              />

              <span
                class="text-xs font-medium"
                :class="statusClass(agent.status)"
              >
                {{ titleCase(agent.status) }}
              </span>
            </div>
          </div>

          <!-- Metrics -->
          <div class="mx-5 grid grid-cols-2 gap-3 sm:grid-cols-4">
            <div class="rounded-lg border border-slate-800 bg-[#0c1320] p-4">
              <p class="text-[10px] uppercase tracking-wider text-slate-500">
                Containers
              </p>

              <p class="mt-2 text-xl font-semibold text-white">
                {{ agent.containers_count }}
              </p>
            </div>

            <div class="rounded-lg border border-slate-800 bg-[#0c1320] p-4">
              <p class="text-[10px] uppercase tracking-wider text-slate-500">
                Projects
              </p>

              <p class="mt-2 text-xl font-semibold text-white">
                {{ agent.projects_count }}
              </p>
            </div>

            <div class="rounded-lg border border-slate-800 bg-[#0c1320] p-4">
              <p class="text-[10px] uppercase tracking-wider text-slate-500">
                Agent Version
              </p>

              <p class="mt-2 break-all text-sm font-semibold text-white">
                {{ agent.agent_version || "Unknown" }}
              </p>
            </div>

            <div class="rounded-lg border border-slate-800 bg-[#0c1320] p-4">
              <p class="text-[10px] uppercase tracking-wider text-slate-500">
                Host Status
              </p>

              <p
                class="mt-2 text-sm font-semibold"
                :class="statusClass(agent.status)"
              >
                {{ titleCase(agent.status) }}
              </p>
            </div>
          </div>

          <!-- Host information -->
          <div class="mt-5 border-t border-slate-800 px-5 py-5">
            <h4 class="text-xs font-semibold text-slate-300">
              Host Information
            </h4>

            <div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
              <div>
                <p class="text-[10px] text-slate-500">
                  Hostname
                </p>

                <p class="mt-1.5 break-all text-xs text-slate-300">
                  {{ agent.hostname || "—" }}
                </p>
              </div>

              <div>
                <p class="text-[10px] text-slate-500">
                  Host IP
                </p>

                <p class="mt-1.5 break-all text-xs text-slate-300">
                  {{ agent.ip_address || "Not reported" }}
                </p>
              </div>

              <div>
                <p class="text-[10px] text-slate-500">
                  Registered
                </p>

                <p class="mt-1.5 text-xs text-slate-300">
                  {{ formatRegisteredDate(agent.registered_at) }}
                </p>
              </div>
            </div>
          </div>

          <!-- Footer -->
          <div class="flex flex-wrap items-center justify-between gap-3 border-t border-slate-800 bg-slate-900/30 px-5 py-4">
            <div>
              <p class="text-[10px] text-slate-500">
                Last heartbeat
              </p>

              <p class="mt-1 text-xs font-medium text-slate-300">
                {{ formatDateTime(agent.last_heartbeat_at) }}
              </p>
            </div>

            <div class="text-right">
              <p class="text-[10px] text-slate-500">
                Agent ID
              </p>

              <p class="mt-1 text-xs text-slate-300">
                #{{ agent.id }}
              </p>
            </div>
          </div>
        </article>
      </template>
    </section>
  </div>
</template>
