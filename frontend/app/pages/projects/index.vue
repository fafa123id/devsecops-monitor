
<script setup lang="ts">
interface Project {
  id: number;
  name: string;
  docker_project_name: string;
  status: string;
  host: {
    id: number;
    name: string;
    hostname: string;
    status: string;
    last_heartbeat_at: string | null;
  } | null;
  containers: {
    running: number;
    total: number;
    missing: number;
    observed_total: number;
  };
  ecosystems: string[];
  findings: {
    total: number;
    critical: number;
    high: number;
    medium: number;
    low: number;
    unknown: number;
  };
  last_scan: {
    id: number;
    trigger_type: string;
    status: string;
    started_at: string | null;
    completed_at: string | null;
    scanner_version: string | null;
  } | null;
  last_scanned_at: string | null;
  last_seen_at: string | null;
  missing_since_at: string | null;
}

interface ProjectsResponse {
  message: string;
  data: Project[];
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
  "projects",
  () => api.get<ProjectsResponse>("/api/projects"),
  {
    default: () => ({
      message: "",
      data: [],
      meta: { total: 0 },
    }),
  },
);

const projects = computed(() => response.value?.data ?? []);

const totalRunningContainers = computed(() =>
  projects.value.reduce(
    (sum, project) => sum + project.containers.running,
    0,
  ),
);

const totalFindings = computed(() =>
  projects.value.reduce(
    (sum, project) => sum + project.findings.total,
    0,
  ),
);

const apiErrorMessage = computed(() => {
  if (!error.value) return "";

  const apiError = error.value as {
    data?: { message?: string };
    message?: string;
  };

  return (
    apiError.data?.message ||
    apiError.message ||
    "Unable to load projects. Please try again."
  );
});

function formatDate(value: string | null | undefined): string {
  if (!value) return "Never";

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return "—";
  }

  return new Intl.DateTimeFormat("en-US", {
    day: "numeric",
    month: "short",
    year: "numeric",
  }).format(date);
}

function getLastScan(project: Project): string {
  return formatDate(
    project.last_scan?.completed_at ||
    project.last_scan?.started_at ||
    project.last_scanned_at,
  );
}

function getStatusLabel(status: string): string {
  return status.charAt(0).toUpperCase() + status.slice(1);
}

function getStatusColor(status: string): string {
  switch (status) {
    case "active":
      return "text-emerald-400";

    case "degraded":
      return "text-yellow-400";

    case "missing":
      return "text-red-400";

    case "stopped":
      return "text-slate-400";

    case "archived":
      return "text-slate-500";

    default:
      return "text-slate-400";
  }
}

function getStatusDot(status: string): string {
  switch (status) {
    case "active":
      return "bg-emerald-400";

    case "degraded":
      return "bg-yellow-400";

    case "missing":
      return "bg-red-400";

    case "stopped":
    case "archived":
      return "bg-slate-500";

    default:
      return "bg-slate-500";
  }
}

function getProjectDescription(project: Project): string {
  if (project.host) {
    return `${project.host.name} · ${project.host.hostname}`;
  }

  return "Docker Compose project";
}
</script>

<template>
  <div class="space-y-7">
    <!-- Page heading -->
    <section class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <p class="text-xs font-medium uppercase tracking-[0.18em] text-sky-400">
          Infrastructure monitoring
        </p>

        <h1 class="mt-2 text-2xl font-semibold text-white">
          Projects
        </h1>

        <p class="mt-2 text-sm text-slate-400">
          Manage discovered Docker Compose projects and their security posture.
        </p>
      </div>

      <div
        class="rounded-lg border border-slate-800 bg-slate-900/60 px-4 py-2.5"
      >
        <p class="text-xs text-slate-500">Total projects</p>

        <p class="mt-1 text-xl font-semibold text-white">
          {{ projects.length }}
        </p>
      </div>
    </section>

    <!-- Summary cards -->
    <section class="grid grid-cols-1 gap-4 sm:grid-cols-3">
      <div class="rounded-xl border border-slate-800 bg-[#111827] p-5">
        <p class="text-sm text-slate-400">Total projects</p>

        <p class="mt-3 text-2xl font-semibold text-white">
          {{ projects.length }}
        </p>
      </div>

      <div class="rounded-xl border border-slate-800 bg-[#111827] p-5">
        <p class="text-sm text-slate-400">Running containers</p>

        <p class="mt-3 text-2xl font-semibold text-emerald-400">
          {{ totalRunningContainers }}
        </p>
      </div>

      <div class="rounded-xl border border-slate-800 bg-[#111827] p-5">
        <p class="text-sm text-slate-400">Total findings</p>

        <p class="mt-3 text-2xl font-semibold text-red-400">
          {{ totalFindings }}
        </p>
      </div>
    </section>

    <!-- Project list -->
    <section class="space-y-4">
      <div class="flex items-center justify-between">
        <h2 class="font-semibold text-white">
          Discovered Projects
        </h2>

        <span class="text-xs text-slate-500">
          {{ projects.length }} projects tracked
        </span>
      </div>

      <!-- Loading -->
      <div
        v-if="pending"
        class="rounded-xl border border-slate-800 bg-[#111827] p-8 text-center"
      >
        <div
          class="mx-auto h-6 w-6 animate-spin rounded-full border-2 border-slate-600 border-t-sky-400"
        />

        <p class="mt-4 text-sm text-slate-400">
          Loading projects...
        </p>
      </div>

      <!-- Error -->
      <div
        v-else-if="error"
        class="rounded-xl border border-red-500/20 bg-[#111827] p-6"
      >
        <p class="text-sm font-medium text-red-400">
          Failed to load projects
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
        v-else-if="projects.length === 0"
        class="rounded-xl border border-slate-800 bg-[#111827] px-6 py-14 text-center"
      >
        <div
          class="mx-auto flex h-12 w-12 items-center justify-center rounded-xl bg-slate-800 text-slate-400"
        >
          <svg
            class="h-6 w-6"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.6"
          >
            <rect x="3" y="4" width="18" height="16" rx="3" />
            <path d="M3 10h18M9 4v6m6-6v6" />
          </svg>
        </div>

        <h3 class="mt-4 text-sm font-semibold text-white">
          No projects discovered
        </h3>

        <p class="mx-auto mt-2 max-w-sm text-xs leading-5 text-slate-500">
          No Docker Compose projects have been registered by your connected agents yet.
        </p>
      </div>

      <!-- Actual project cards -->
      <template v-else>
        <article
          v-for="project in projects"
          :key="project.id"
          class="rounded-xl border border-slate-800 bg-[#111827] p-6 transition hover:border-slate-700"
        >
          <div class="flex flex-wrap items-start justify-between gap-4">
            <!-- Project identity -->
            <div class="flex items-center gap-4">
              <div
                class="flex h-12 w-12 items-center justify-center rounded-xl bg-sky-500/10 text-sky-400"
              >
                <svg
                  class="h-6 w-6"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.6"
                  stroke-linejoin="round"
                >
                  <rect x="3" y="3" width="8" height="8" rx="1.5" />
                  <rect x="13" y="3" width="8" height="8" rx="1.5" />
                  <rect x="3" y="13" width="8" height="8" rx="1.5" />
                  <rect x="13" y="13" width="8" height="8" rx="1.5" />
                </svg>
              </div>

              <div>
                <NuxtLink
                  :to="`/projects/${project.id}`"
                  class="font-semibold text-white hover:text-sky-400"
                >
                  {{ project.name }}
                </NuxtLink>

                <p class="mt-1 text-xs text-slate-500">
                  {{ getProjectDescription(project) }}
                </p>
              </div>
            </div>

            <!-- Project status -->
            <div class="flex items-center gap-2 pt-1">
              <span
                class="h-1.5 w-1.5 rounded-full"
                :class="getStatusDot(project.status)"
              />

              <span
                class="text-xs font-medium"
                :class="getStatusColor(project.status)"
              >
                {{ getStatusLabel(project.status) }}
              </span>
            </div>
          </div>

          <!-- Project metrics -->
          <div
            class="mt-6 grid grid-cols-2 gap-5 border-y border-slate-800 py-5 md:grid-cols-4"
          >
            <div>
              <p class="text-xs text-slate-500">Containers</p>

              <p class="mt-2 text-lg font-semibold text-white">
                {{ project.containers.running }}

                <span class="text-sm font-normal text-slate-500">
                  / {{ project.containers.total }} running
                </span>
              </p>

              <p
                v-if="project.containers.missing > 0"
                class="mt-1 text-xs text-red-400"
              >
                {{ project.containers.missing }} missing
              </p>
            </div>

            <div>
              <p class="text-xs text-slate-500">Total findings</p>

              <p class="mt-2 text-lg font-semibold text-white">
                {{ project.findings.total }}
              </p>
            </div>

            <div>
              <p class="text-xs text-slate-500">Critical / High</p>

              <p class="mt-2 text-lg font-semibold text-red-400">
                {{ project.findings.critical + project.findings.high }}
              </p>
            </div>

            <div>
              <p class="text-xs text-slate-500">Last scan</p>

              <p class="mt-2 text-sm font-medium text-slate-300">
                {{ getLastScan(project) }}
              </p>
            </div>
          </div>

          <!-- Ecosystems and severity -->
          <div class="mt-5 flex flex-wrap items-center justify-between gap-4">
            <div class="flex flex-wrap gap-2">
              <span
                v-for="ecosystem in project.ecosystems"
                :key="ecosystem"
                class="rounded-md border border-slate-700 bg-slate-800/60 px-3 py-1.5 text-xs text-slate-300"
              >
                {{ ecosystem }}
              </span>

              <span
                v-if="project.ecosystems.length === 0"
                class="text-xs text-slate-500"
              >
                No ecosystem data
              </span>
            </div>

            <div class="flex flex-wrap gap-4 text-xs">
              <span class="text-red-400">
                Critical {{ project.findings.critical }}
              </span>

              <span class="text-orange-400">
                High {{ project.findings.high }}
              </span>

              <span class="text-yellow-400">
                Medium {{ project.findings.medium }}
              </span>

              <span class="text-sky-400">
                Low {{ project.findings.low }}
              </span>
            </div>
          </div>
        </article>
      </template>
    </section>
  </div>
</template>
