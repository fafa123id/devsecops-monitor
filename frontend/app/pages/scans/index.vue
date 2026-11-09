
<script setup lang="ts">
interface ScanRecord {
  id: number;
  project: {
    id: number;
    name: string;
    docker_project_name: string;
  } | null;
  trigger_type: string;
  status: string;
  scanner_version: string | null;
  ecosystems: string[];
  ecosystems_count: number;
  ecosystem_results: {
    ecosystem: string;
    manifest_file: string;
    status: string;
    error_message: string | null;
  }[];
  findings_count: number;
  started_at: string | null;
  completed_at: string | null;
  duration_ms: number | null;
  error_message: string | null;
}

interface ScanHistoryResponse {
  message: string;
  data: ScanRecord[];
  summary: {
    total_scans: number;
    completed: number;
    partial: number;
    failed: number;
    findings_detected: number;
  };
  filters: {
    projects: {
      id: number;
      name: string;
    }[];
  };
  pagination: {
    current_page: number;
    last_page: number;
    per_page: number;
    total: number;
  };
}

const api = useApi();

const search = ref("");
const selectedStatus = ref("All");
const selectedTrigger = ref("All");
const selectedProject = ref("All");

const currentPage = ref(1);
const perPage = 15;

const statuses = [
  { value: "All", label: "All statuses" },
  { value: "queued", label: "Queued" },
  { value: "preparing", label: "Preparing" },
  { value: "scanning", label: "Scanning" },
  { value: "processing", label: "Processing" },
  { value: "completed", label: "Completed" },
  { value: "partial", label: "Partial" },
  { value: "failed", label: "Failed" },
  { value: "cancelled", label: "Cancelled" },
  { value: "unsupported", label: "Unsupported" },
];

const triggers = [
  { value: "All", label: "All triggers" },
  { value: "discovery", label: "Discovery" },
  { value: "manual", label: "Manual" },
  { value: "scheduled", label: "Scheduled" },
  { value: "image_updated", label: "Image Updated" },
  { value: "remediation", label: "Remediation" },
];

watch(
  [search, selectedStatus, selectedTrigger, selectedProject],
  () => {
    currentPage.value = 1;
  },
);

const apiPath = computed(() => {
  const params = new URLSearchParams();

  if (search.value.trim()) {
    params.set("q", search.value.trim());
  }

  if (selectedStatus.value !== "All") {
    params.set("status", selectedStatus.value);
  }

  if (selectedTrigger.value !== "All") {
    params.set("trigger_type", selectedTrigger.value);
  }

  if (selectedProject.value !== "All") {
    params.set("project_id", selectedProject.value);
  }

  params.set("page", String(currentPage.value));
  params.set("per_page", String(perPage));

  return `/api/scans?${params.toString()}`;
});

const {
  data: response,
  pending,
  error,
  refresh,
} = await useAsyncData(
  "scan-history",
  () => api.get<ScanHistoryResponse>(apiPath.value),
  {
    watch: [apiPath],
    default: () => ({
      message: "",
      data: [],
      summary: {
        total_scans: 0,
        completed: 0,
        partial: 0,
        failed: 0,
        findings_detected: 0,
      },
      filters: {
        projects: [],
      },
      pagination: {
        current_page: 1,
        last_page: 1,
        per_page: perPage,
        total: 0,
      },
    }),
  },
);

const scans = computed(() => response.value?.data ?? []);
const summary = computed(() => response.value.summary);
const pagination = computed(() => response.value.pagination);

const projects = computed(() => response.value.filters.projects ?? []);

const pageItems = computed<(number | "...")[]>(() => {
  const last = pagination.value.last_page;
  const current = currentPage.value;

  if (last <= 7) {
    return Array.from({ length: last }, (_, index) => index + 1);
  }

  const start = Math.max(2, current - 2);
  const end = Math.min(last - 1, current + 2);

  const pages: (number | "...")[] = [1];

  if (start > 2) pages.push("...");

  for (let page = start; page <= end; page++) {
    pages.push(page);
  }

  if (end < last - 1) pages.push("...");

  pages.push(last);

  return pages;
});

const apiErrorMessage = computed(() => {
  const apiError = error.value as {
    data?: { message?: string };
    message?: string;
  } | null;

  return (
    apiError?.data?.message ||
    apiError?.message ||
    "Unable to load scan history."
  );
});

function titleCase(value: string): string {
  if (!value) return "Unknown";

  return value
    .split("_")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function statusClass(status: string): string {
  return (
    {
      completed: "text-emerald-400",
      partial: "text-amber-400",
      failed: "text-red-400",
      unsupported: "text-slate-400",
      queued: "text-sky-400",
      preparing: "text-sky-400",
      scanning: "text-sky-400",
      processing: "text-sky-400",
      cancelled: "text-slate-400",
    }[status] || "text-slate-400"
  );
}

function statusDot(status: string): string {
  return (
    {
      completed: "bg-emerald-400",
      partial: "bg-amber-400",
      failed: "bg-red-400",
      unsupported: "bg-slate-400",
      queued: "bg-sky-400",
      preparing: "bg-sky-400",
      scanning: "bg-sky-400",
      processing: "bg-sky-400",
      cancelled: "bg-slate-400",
    }[status] || "bg-slate-400"
  );
}

function formatDateTime(value: string | null): string {
  if (!value) return "—";

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) return "—";

  return new Intl.DateTimeFormat("en-US", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(date);
}

function formatDuration(duration: number | null): string {
  if (duration === null) return "—";

  if (duration < 1000) {
    return `${duration}ms`;
  }

  return `${(duration / 1000).toFixed(1)}s`;
}

function changePage(page: number) {
  if (page < 1 || page > pagination.value.last_page) return;

  currentPage.value = page;
}
</script>

<template>
  <div class="space-y-7">
    <!-- Heading -->
    <section>
      <p class="text-xs font-medium uppercase tracking-[0.18em] text-sky-400">
        Security operations
      </p>

      <h1 class="mt-2 text-2xl font-semibold text-white">
        Scan History
      </h1>

      <p class="mt-2 text-sm text-slate-400">
        Track dependency security scans across your monitored projects.
      </p>
    </section>

    <!-- Summary -->
    <section class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <div
        v-for="item in [
          { label: 'Total Scans', value: summary.total_scans, color: 'text-white' },
          { label: 'Completed', value: summary.completed, color: 'text-emerald-400' },
          { label: 'Partial', value: summary.partial, color: 'text-amber-400' },
          { label: 'Findings Detected', value: summary.findings_detected, color: 'text-sky-400' },
        ]"
        :key="item.label"
        class="rounded-xl border border-slate-800 bg-[#111827] p-5"
      >
        <p class="text-xs text-slate-400">
          {{ item.label }}
        </p>

        <p class="mt-3 text-2xl font-semibold" :class="item.color">
          {{ item.value }}
        </p>
      </div>
    </section>

    <!-- Filters -->
    <section class="rounded-xl border border-slate-800 bg-[#111827] p-5">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <h2 class="text-sm font-semibold text-white">
          Scan Explorer
        </h2>

        <span class="text-xs text-slate-500">
          {{ pagination.total }} results
        </span>
      </div>

      <div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <input
          v-model="search"
          type="search"
          placeholder="Search project or scan ID..."
          class="min-w-0 rounded-lg border border-slate-700 bg-[#0c1320] px-3 py-2.5 text-xs text-white outline-none placeholder:text-slate-600 focus:border-sky-500"
        />

        <select
          v-model="selectedStatus"
          class="rounded-lg border border-slate-700 bg-[#0c1320] px-3 py-2.5 text-xs text-slate-300 outline-none focus:border-sky-500"
        >
          <option
            v-for="item in statuses"
            :key="item.value"
            :value="item.value"
          >
            {{ item.label }}
          </option>
        </select>

        <select
          v-model="selectedTrigger"
          class="rounded-lg border border-slate-700 bg-[#0c1320] px-3 py-2.5 text-xs text-slate-300 outline-none focus:border-sky-500"
        >
          <option
            v-for="item in triggers"
            :key="item.value"
            :value="item.value"
          >
            {{ item.label }}
          </option>
        </select>

        <select
          v-model="selectedProject"
          class="rounded-lg border border-slate-700 bg-[#0c1320] px-3 py-2.5 text-xs text-slate-300 outline-none focus:border-sky-500"
        >
          <option value="All">All projects</option>

          <option
            v-for="project in projects"
            :key="project.id"
            :value="String(project.id)"
          >
            {{ project.name }}
          </option>
        </select>
      </div>
    </section>

    <!-- Scan table -->
    <section class="overflow-hidden rounded-xl border border-slate-800 bg-[#111827]">
      <div class="border-b border-slate-800 px-5 py-4">
        <h2 class="text-sm font-semibold text-white">
          Scan Records
        </h2>

        <p class="mt-1 text-xs text-slate-500">
          Historical dependency scan executions
        </p>
      </div>

      <!-- Loading -->
      <div
        v-if="pending"
        class="px-5 py-14 text-center"
      >
        <div
          class="mx-auto h-6 w-6 animate-spin rounded-full border-2 border-slate-600 border-t-sky-400"
        />

        <p class="mt-4 text-sm text-slate-400">
          Loading scan history...
        </p>
      </div>

      <!-- Error -->
      <div
        v-else-if="error"
        class="px-5 py-12 text-center"
      >
        <p class="text-sm font-medium text-red-400">
          Failed to load scan history
        </p>

        <p class="mt-2 text-xs text-slate-500">
          {{ apiErrorMessage }}
        </p>

        <button
          type="button"
          class="mt-4 rounded-lg border border-slate-700 px-4 py-2 text-xs text-slate-300 transition hover:border-sky-500 hover:text-sky-400"
          @click="refresh()"
        >
          Try again
        </button>
      </div>

      <!-- Records -->
      <template v-else>
        <div class="overflow-x-auto">
          <table class="w-full min-w-[950px] text-left">
            <thead class="bg-slate-900/60">
              <tr class="text-[10px] uppercase tracking-wider text-slate-500">
                <th class="px-5 py-4 font-medium">Scan ID</th>
                <th class="px-5 py-4 font-medium">Project</th>
                <th class="px-5 py-4 font-medium">Trigger</th>
                <th class="px-5 py-4 font-medium">Ecosystems</th>
                <th class="px-5 py-4 font-medium">Findings</th>
                <th class="px-5 py-4 font-medium">Duration</th>
                <th class="px-5 py-4 font-medium">Status</th>
                <th class="px-5 py-4 font-medium">Started At</th>
              </tr>
            </thead>

            <tbody class="divide-y divide-slate-800">
              <tr
                v-for="scan in scans"
                :key="scan.id"
                class="transition hover:bg-slate-800/30"
              >
                <td class="px-5 py-4 text-xs font-medium text-sky-400">
                  #{{ scan.id }}
                </td>

                <td class="px-5 py-4 text-xs font-medium text-white">
                  {{ scan.project?.name || "Unknown project" }}
                </td>

                <td class="px-5 py-4">
                  <span class="rounded-md border border-slate-700 bg-slate-800/60 px-2 py-1 text-[10px] text-slate-300">
                    {{ titleCase(scan.trigger_type) }}
                  </span>
                </td>

                <td class="px-5 py-4">
                  <div class="flex max-w-[180px] flex-wrap gap-1.5">
                    <span
                      v-for="ecosystem in scan.ecosystems"
                      :key="ecosystem"
                      class="rounded bg-slate-800 px-2 py-1 text-[10px] text-slate-300"
                    >
                      {{ ecosystem }}
                    </span>

                    <span
                      v-if="scan.ecosystems_count === 0"
                      class="text-xs text-slate-500"
                    >
                      —
                    </span>
                  </div>
                </td>

                <td class="px-5 py-4 text-xs font-medium text-white">
                  {{ scan.findings_count }}
                </td>

                <td class="px-5 py-4 text-xs text-slate-400">
                  {{ formatDuration(scan.duration_ms) }}
                </td>

                <td class="px-5 py-4">
                  <div class="flex items-center gap-2">
                    <span
                      class="h-1.5 w-1.5 rounded-full"
                      :class="statusDot(scan.status)"
                    />

                    <span
                      class="text-xs"
                      :class="statusClass(scan.status)"
                    >
                      {{ titleCase(scan.status) }}
                    </span>
                  </div>
                </td>

                <td class="px-5 py-4 text-xs text-slate-500">
                  {{ formatDateTime(scan.started_at) }}
                </td>
              </tr>

              <tr v-if="scans.length === 0">
                <td colspan="8" class="px-5 py-12 text-center">
                  <p class="text-sm text-slate-400">
                    No scan records found.
                  </p>

                  <p class="mt-1 text-xs text-slate-600">
                    Try changing your search or filters.
                  </p>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Numbered pagination -->
        <div
          class="flex flex-col gap-3 border-t border-slate-800 px-4 py-4 sm:flex-row sm:items-center sm:justify-between sm:px-5"
        >
          <p class="text-center text-xs text-slate-500 sm:text-left">
            Page {{ pagination.current_page }} of {{ pagination.last_page }}
            <span class="mx-1 text-slate-700">·</span>
            {{ pagination.total }} scans
          </p>

          <div class="w-full max-w-full overflow-x-auto sm:w-auto sm:overflow-visible">
            <div class="mx-auto flex w-max min-w-full items-center justify-center gap-1 sm:mx-0 sm:min-w-0 sm:justify-end">
              <button
                type="button"
                :disabled="currentPage <= 1 || pending"
                aria-label="Previous page"
                class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-700 text-xs text-slate-300 transition hover:border-sky-500 hover:text-sky-400 disabled:cursor-not-allowed disabled:opacity-30"
                @click="changePage(currentPage - 1)"
              >
                ←
              </button>

              <template
                v-for="(item, index) in pageItems"
                :key="`${item}-${index}`"
              >
                <span
                  v-if="item === '...'"
                  class="flex h-8 w-6 shrink-0 items-center justify-center text-xs text-slate-500"
                >
                  ...
                </span>

                <button
                  v-else
                  type="button"
                  :disabled="pending"
                  :aria-label="`Page ${item}`"
                  :aria-current="item === currentPage ? 'page' : undefined"
                  class="flex h-8 min-w-8 shrink-0 items-center justify-center rounded-lg border px-2 text-xs font-medium transition disabled:cursor-not-allowed"
                  :class="
                    item === currentPage
                      ? 'border-sky-500/40 bg-sky-500/10 text-sky-400'
                      : 'border-slate-700 text-slate-400 hover:border-sky-500 hover:text-sky-400'
                  "
                  @click="changePage(item)"
                >
                  {{ item }}
                </button>
              </template>

              <button
                type="button"
                :disabled="currentPage >= pagination.last_page || pending"
                aria-label="Next page"
                class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-700 text-xs text-slate-300 transition hover:border-sky-500 hover:text-sky-400 disabled:cursor-not-allowed disabled:opacity-30"
                @click="changePage(currentPage + 1)"
              >
                →
              </button>
            </div>
          </div>
        </div>
      </template>
    </section>
  </div>
</template>
