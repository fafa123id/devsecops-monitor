
<script setup lang="ts">
interface ProjectDetail {
  id: number;
  name: string;
  docker_project_name: string;
  status: string;
  scan_source: boolean;
  weekly_scan_enabled: boolean;

  host: {
    id: number;
    name: string;
    hostname: string;
    status: string;
    last_heartbeat_at: string | null;
  } | null;

  containers: {
    id: number;
    container_id: string;
    name: string;
    service_name: string | null;
    image_reference: string | null;
    state: string;
    last_seen_at: string | null;
  }[];

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

interface ProjectResponse {
  message: string;
  data: ProjectDetail;
}

const route = useRoute();
const api = useApi();

const projectId = computed(() => String(route.params.id ?? ""));

const {
  data: response,
  pending,
  error,
  refresh,
} = await useAsyncData(
  () => `project-detail-${projectId.value}`,
  () =>
    api.get<ProjectResponse>(
      `/api/projects/${encodeURIComponent(projectId.value)}`,
    ),
  {
    watch: [projectId],
  },
);

const project = computed(() => response.value?.data ?? null);

const totalFindings = computed(() => project.value?.findings.total ?? 0);

const activeContainers = computed(
  () =>
    project.value?.containers.filter(
      (container) => container.state !== "missing",
    ) ?? [],
);

const runningContainers = computed(
  () =>
    project.value?.containers.filter(
      (container) => container.state === "running",
    ).length ?? 0,
);

const missingContainers = computed(
  () =>
    project.value?.containers.filter(
      (container) => container.state === "missing",
    ).length ?? 0,
);

const severities = [
  {
    key: "critical",
    label: "Critical",
    color: "bg-red-500",
    text: "text-red-400",
  },
  {
    key: "high",
    label: "High",
    color: "bg-orange-500",
    text: "text-orange-400",
  },
  {
    key: "medium",
    label: "Medium",
    color: "bg-yellow-500",
    text: "text-yellow-400",
  },
  {
    key: "low",
    label: "Low",
    color: "bg-sky-500",
    text: "text-sky-400",
  },
  {
    key: "unknown",
    label: "Unknown",
    color: "bg-slate-500",
    text: "text-slate-400",
  },
] as const;

function formatDate(value: string | null | undefined): string {
  if (!value) return "Never";

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) return "—";

  return new Intl.DateTimeFormat("en-US", {
    day: "numeric",
    month: "long",
    year: "numeric",
  }).format(date);
}

const lastScanDate = computed(() => {
  if (!project.value) return "Never";

  return formatDate(
    project.value.last_scan?.completed_at ||
      project.value.last_scan?.started_at ||
      project.value.last_scanned_at,
  );
});

function getProjectStatusColor(status: string): string {
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

function getContainerStatusColor(status: string): string {
  switch (status) {
    case "running":
      return "text-emerald-400";
    case "paused":
    case "restarting":
      return "text-yellow-400";
    case "missing":
      return "text-red-400";
    default:
      return "text-slate-400";
  }
}

function getContainerStatusDot(status: string): string {
  switch (status) {
    case "running":
      return "bg-emerald-400";
    case "paused":
    case "restarting":
      return "bg-yellow-400";
    case "missing":
      return "bg-red-400";
    default:
      return "bg-slate-500";
  }
}

function getErrorStatus(): number | undefined {
  const apiError = error.value as {
    statusCode?: number;
    status?: number;
  } | null;

  return apiError?.statusCode ?? apiError?.status;
}

const errorMessage = computed(() => {
  const apiError = error.value as {
    data?: { message?: string };
    message?: string;
  } | null;

  if (getErrorStatus() === 404) {
    return "This project does not exist or is no longer available.";
  }

  return (
    apiError?.data?.message ||
    apiError?.message ||
    "Unable to load project details."
  );
});
</script>

<template>
  <div class="space-y-7">
    <!-- Loading -->
    <div
      v-if="pending"
      class="rounded-xl border border-slate-800 bg-[#111827] p-12 text-center"
    >
      <div
        class="mx-auto h-6 w-6 animate-spin rounded-full border-2 border-slate-600 border-t-sky-400"
      />

      <p class="mt-4 text-sm text-slate-400">
        Loading project details...
      </p>
    </div>

    <!-- Error -->
    <div
      v-else-if="error"
      class="rounded-xl border border-slate-800 bg-[#111827] p-8"
    >
      <h1 class="text-xl font-semibold text-white">
        {{ getErrorStatus() === 404 ? "Project not found" : "Unable to load project" }}
      </h1>

      <p class="mt-2 text-sm text-slate-400">
        {{ errorMessage }}
      </p>

      <div class="mt-5 flex gap-3">
        <button
          v-if="getErrorStatus() !== 404"
          type="button"
          class="rounded-lg border border-slate-700 px-4 py-2 text-xs text-slate-300 hover:border-sky-500 hover:text-sky-400"
          @click="refresh()"
        >
          Try again
        </button>

        <NuxtLink
          to="/projects"
          class="rounded-lg bg-sky-500 px-4 py-2 text-xs font-medium text-slate-950 hover:bg-sky-400"
        >
          Back to Projects
        </NuxtLink>
      </div>
    </div>

    <!-- Project details -->
    <template v-else-if="project">
      <NuxtLink
        to="/projects"
        class="inline-block text-xs text-slate-400 transition hover:text-sky-400"
      >
        ← Back to Projects
      </NuxtLink>

      <!-- Header -->
      <section class="flex flex-wrap items-center justify-between gap-4">
        <div>
          <div class="flex items-center gap-2 text-xs text-slate-500">
            <span>Projects</span>
            <span>/</span>
            <span class="text-sky-400">{{ project.name }}</span>
          </div>

          <h1 class="mt-3 text-2xl font-semibold text-white">
            {{ project.name }}
          </h1>

          <p class="mt-2 text-sm text-slate-400">
            {{ project.host?.name || "Docker Compose project" }}
            <span v-if="project.host">
              · {{ project.host.hostname }}
            </span>
          </p>
        </div>

        <div class="flex items-center gap-2">
          <span
            class="h-2 w-2 rounded-full"
            :class="getContainerStatusDot(
              project.status === 'active' ? 'running' : project.status
            )"
          />

          <span
            class="text-xs font-medium"
            :class="getProjectStatusColor(project.status)"
          >
            {{ project.status.charAt(0).toUpperCase() + project.status.slice(1) }}
          </span>
        </div>
      </section>

      <!-- Overview -->
      <section class="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <div class="rounded-xl border border-slate-800 bg-[#111827] p-5">
          <p class="text-sm text-slate-400">Containers</p>

          <p class="mt-3 text-2xl font-semibold text-white">
            {{ activeContainers.length }}
          </p>

          <p class="mt-1 text-xs text-slate-500">
            {{ runningContainers }} running
          </p>
        </div>

        <div class="rounded-xl border border-slate-800 bg-[#111827] p-5">
          <p class="text-sm text-slate-400">Total Findings</p>

          <p class="mt-3 text-2xl font-semibold text-white">
            {{ totalFindings }}
          </p>
        </div>

        <div class="rounded-xl border border-slate-800 bg-[#111827] p-5">
          <p class="text-sm text-slate-400">Ecosystems</p>

          <p class="mt-3 text-2xl font-semibold text-white">
            {{ project.ecosystems.length }}
          </p>
        </div>
      </section>

      <!-- Severity -->
      <section class="rounded-xl border border-slate-800 bg-[#111827] p-6">
        <h2 class="font-semibold text-white">
          Vulnerability Summary
        </h2>

        <p class="mt-1 text-xs text-slate-500">
          Active vulnerability findings grouped by severity
        </p>

        <div class="mt-6 grid grid-cols-2 gap-5 lg:grid-cols-5">
          <div v-for="item in severities" :key="item.key">
            <p class="text-xs text-slate-500">
              {{ item.label }}
            </p>

            <p class="mt-2 text-2xl font-semibold" :class="item.text">
              {{ project.findings[item.key] }}
            </p>
          </div>
        </div>

        <div class="mt-6 flex h-2 overflow-hidden rounded-full bg-slate-800">
          <div
            v-for="item in severities"
            :key="item.key"
            :class="item.color"
            :style="{
              width: `${
                totalFindings
                  ? (project.findings[item.key] / totalFindings) * 100
                  : 0
              }%`,
            }"
          />
        </div>
      </section>

      <!-- Containers -->
      <section
        class="overflow-hidden rounded-xl border border-slate-800 bg-[#111827]"
      >
        <div class="border-b border-slate-800 px-6 py-5">
          <h2 class="font-semibold text-white">
            Containers
          </h2>

          <p class="mt-1 text-xs text-slate-500">
            Containers discovered in this Compose project
          </p>
        </div>

        <div v-if="project.containers.length" class="divide-y divide-slate-800">
          <div
            v-for="container in project.containers"
            :key="container.id"
            class="flex items-center justify-between gap-4 px-6 py-4"
          >
            <div class="min-w-0">
              <p class="truncate text-sm font-medium text-white">
                {{ container.name }}
              </p>

              <p class="mt-1 text-xs text-slate-500">
                {{ container.service_name || container.name }}
              </p>

              <p
                v-if="container.image_reference"
                class="mt-1 truncate text-[10px] text-slate-600"
              >
                {{ container.image_reference }}
              </p>
            </div>

            <div class="flex shrink-0 items-center gap-2">
              <span
                class="h-1.5 w-1.5 rounded-full"
                :class="getContainerStatusDot(container.state)"
              />

              <span
                class="text-xs"
                :class="getContainerStatusColor(container.state)"
              >
                {{ container.state.charAt(0).toUpperCase() + container.state.slice(1) }}
              </span>
            </div>
          </div>
        </div>

        <div v-else class="px-6 py-8 text-center">
          <p class="text-sm text-slate-500">
            No containers discovered.
          </p>
        </div>
      </section>

      <!-- Ecosystems -->
      <section class="rounded-xl border border-slate-800 bg-[#111827] p-6">
        <h2 class="font-semibold text-white">
          Detected Ecosystems
        </h2>

        <div v-if="project.ecosystems.length" class="mt-4 flex flex-wrap gap-2">
          <span
            v-for="ecosystem in project.ecosystems"
            :key="ecosystem"
            class="rounded-lg border border-slate-700 bg-slate-800/60 px-3 py-2 text-xs text-slate-300"
          >
            {{ ecosystem }}
          </span>
        </div>

        <p v-else class="mt-4 text-xs text-slate-500">
          No ecosystem data available.
        </p>

        <div class="mt-6 border-t border-slate-800 pt-5">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <p class="text-xs text-slate-500">
              Last scan
            </p>

            <p class="text-sm font-medium text-slate-300">
              {{ lastScanDate }}
            </p>
          </div>

          <div
            v-if="project.last_scan"
            class="mt-3 flex flex-wrap items-center justify-between gap-3"
          >
            <p class="text-xs text-slate-500">
              Scan status
            </p>

            <p class="text-xs font-medium text-slate-300">
              {{ project.last_scan.status }}
            </p>
          </div>

          <div
            v-if="project.last_scan?.scanner_version"
            class="mt-3 flex flex-wrap items-center justify-between gap-3"
          >
            <p class="text-xs text-slate-500">
              Scanner version
            </p>

            <p class="text-xs text-slate-300">
              {{ project.last_scan.scanner_version }}
            </p>
          </div>
        </div>
      </section>

      <!-- Scan policy -->
      <section class="rounded-xl border border-slate-800 bg-[#111827] p-6">
        <h2 class="font-semibold text-white">
          Scan Configuration
        </h2>

        <div class="mt-5 grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <p class="text-xs text-slate-500">
              Source scanning
            </p>

            <p
              class="mt-2 text-sm font-medium"
              :class="project.scan_source ? 'text-emerald-400' : 'text-slate-400'"
            >
              {{ project.scan_source ? "Enabled" : "Disabled" }}
            </p>
          </div>

          <div>
            <p class="text-xs text-slate-500">
              Weekly scanning
            </p>

            <p
              class="mt-2 text-sm font-medium"
              :class="project.weekly_scan_enabled ? 'text-emerald-400' : 'text-slate-400'"
            >
              {{ project.weekly_scan_enabled ? "Enabled" : "Disabled" }}
            </p>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>
