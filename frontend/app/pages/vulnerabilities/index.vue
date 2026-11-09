<script setup lang="ts">
interface Vulnerability {
  id: number;
  finding_id: string;
  vulnerability_id: string | null;
  title: string | null;
  description: string | null;
  primary_url: string | null;
  package: string | null;
  installed_version: string | null;
  fixed_version: string | null;
  ecosystem: string | null;
  manifest_file: string | null;
  severity: string;
  status: string;
  project: {
    id: number;
    name: string;
    docker_project_name: string;
  } | null;
  first_seen_at: string | null;
  last_seen_at: string | null;
}

interface VulnerabilitiesResponse {
  message: string;
  data: Vulnerability[];
  summary: {
    total: number;
    critical: number;
    high: number;
    medium: number;
    low: number;
    unknown: number;
  };
  filters: {
    ecosystems: string[];
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
const selectedSeverity = ref("All");
const selectedEcosystem = ref("All");
const selectedProject = ref("All");
const selectedStatus = ref("active");

const currentPage = ref(1);
const perPage = 15;

const severities = [
  { value: "All", label: "All Severities" },
  { value: "critical", label: "Critical" },
  { value: "high", label: "High" },
  { value: "medium", label: "Medium" },
  { value: "low", label: "Low" },
  { value: "unknown", label: "Unknown" },
];

const statuses = [
  { value: "active", label: "Active Findings" },
  { value: "all", label: "All Statuses" },
  { value: "new", label: "New" },
  { value: "existing", label: "Existing" },
  { value: "reopened", label: "Reopened" },
  { value: "resolved", label: "Resolved" },
];

// Reset pagination whenever a filter changes.
watch(
  [
    search,
    selectedSeverity,
    selectedEcosystem,
    selectedProject,
    selectedStatus,
  ],
  () => {
    currentPage.value = 1;
  },
);

// Build backend query parameters.
const apiPath = computed(() => {
  const params = new URLSearchParams();

  if (search.value.trim()) {
    params.set("q", search.value.trim());
  }

  if (selectedSeverity.value !== "All") {
    params.set("severity", selectedSeverity.value);
  }

  if (selectedEcosystem.value !== "All") {
    params.set("ecosystem", selectedEcosystem.value);
  }

  if (selectedProject.value !== "All") {
    params.set("project_id", selectedProject.value);
  }

  params.set("status", selectedStatus.value);
  params.set("page", String(currentPage.value));
  params.set("per_page", String(perPage));

  return `/api/vulnerabilities?${params.toString()}`;
});

const {
  data: response,
  pending,
  error,
  refresh,
} = await useAsyncData(
  "vulnerabilities",
  () => api.get<VulnerabilitiesResponse>(apiPath.value),
  {
    watch: [apiPath],
    default: () => ({
      message: "",
      data: [],
      summary: {
        total: 0,
        critical: 0,
        high: 0,
        medium: 0,
        low: 0,
        unknown: 0,
      },
      filters: {
        ecosystems: [],
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

const vulnerabilities = computed(() => response.value?.data ?? []);
const summary = computed(() => response.value.summary);
const pagination = computed(() => response.value.pagination);
const pageItems = computed<(number | "...")[]>(() => {
  const last = pagination.value.last_page;
  const current = currentPage.value;

  if (last <= 7) {
    return Array.from({ length: last }, (_, index) => index + 1);
  }

  const start = Math.max(2, current - 2);
  const end = Math.min(last - 1, current + 2);

  const pages: (number | "...")[] = [1];

  if (start > 2) {
    pages.push("...");
  }

  for (let page = start; page <= end; page++) {
    pages.push(page);
  }

  if (end < last - 1) {
    pages.push("...");
  }

  pages.push(last);

  return pages;
});
const ecosystems = computed(() => [
  "All",
  ...response.value.filters.ecosystems,
]);

const projects = computed(() => response.value.filters.projects ?? []);

const apiErrorMessage = computed(() => {
  const apiError = error.value as {
    data?: { message?: string };
    message?: string;
  } | null;

  return (
    apiError?.data?.message ||
    apiError?.message ||
    "Unable to load vulnerabilities."
  );
});

function vulnerabilityIdentifier(item: Vulnerability): string {
  return item.vulnerability_id || item.finding_id || `FINDING-${item.id}`;
}

function titleCase(value: string): string {
  if (!value) return "Unknown";

  return value.charAt(0).toUpperCase() + value.slice(1);
}

function severityColor(severity: string): string {
  return (
    {
      critical: "bg-red-500/10 text-red-400",
      high: "bg-orange-500/10 text-orange-400",
      medium: "bg-yellow-500/10 text-yellow-400",
      low: "bg-sky-500/10 text-sky-400",
      unknown: "bg-slate-700 text-slate-300",
    }[severity] || "bg-slate-700 text-slate-300"
  );
}

function statusColor(status: string): string {
  return (
    {
      new: "text-sky-400",
      existing: "text-slate-400",
      reopened: "text-orange-400",
      resolved: "text-emerald-400",
    }[status] || "text-slate-400"
  );
}

function changePage(page: number) {
  if (page < 1 || page > pagination.value.last_page) return;

  currentPage.value = page;
}

function formatDate(value: string | null): string {
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
    <section>
      <p class="text-xs font-medium uppercase tracking-[0.18em] text-sky-400">
        Security findings
      </p>

      <h1 class="mt-2 text-2xl font-semibold text-white">Vulnerabilities</h1>

      <p class="mt-2 text-sm text-slate-400">
        Review and track dependency vulnerabilities across your projects.
      </p>
    </section>

    <!-- Summary -->
    <section class="grid grid-cols-2 gap-4 xl:grid-cols-4">
      <div
        v-for="item in [
          { label: 'Critical', value: summary.critical, color: 'text-red-400' },
          { label: 'High', value: summary.high, color: 'text-orange-400' },
          { label: 'Medium', value: summary.medium, color: 'text-yellow-400' },
          { label: 'Low', value: summary.low, color: 'text-sky-400' },
        ]"
        :key="item.label"
        class="rounded-xl border border-slate-800 bg-[#111827] p-5"
      >
        <p class="text-xs text-slate-400">{{ item.label }}</p>

        <p class="mt-3 text-2xl font-semibold" :class="item.color">
          {{ item.value }}
        </p>
      </div>
    </section>

    <!-- Filters -->
    <section class="rounded-xl border border-slate-800 bg-[#111827] p-5">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <h2 class="text-sm font-semibold text-white">Finding Explorer</h2>

        <span class="text-xs text-slate-500">
          {{ pagination.total }} results
        </span>
      </div>

      <div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-5">
        <input
          v-model="search"
          type="search"
          placeholder="Search CVE, GHSA, package..."
          class="min-w-0 rounded-lg border border-slate-700 bg-[#0c1320] px-3 py-2.5 text-xs text-white outline-none placeholder:text-slate-600 focus:border-sky-500"
        />

        <select
          v-model="selectedSeverity"
          class="rounded-lg border border-slate-700 bg-[#0c1320] px-3 py-2.5 text-xs text-slate-300 outline-none focus:border-sky-500"
        >
          <option
            v-for="item in severities"
            :key="item.value"
            :value="item.value"
          >
            {{ item.label }}
          </option>
        </select>

        <select
          v-model="selectedEcosystem"
          class="rounded-lg border border-slate-700 bg-[#0c1320] px-3 py-2.5 text-xs text-slate-300 outline-none focus:border-sky-500"
        >
          <option v-for="item in ecosystems" :key="item" :value="item">
            {{ item === "All" ? "All Ecosystems" : item }}
          </option>
        </select>

        <select
          v-model="selectedProject"
          class="rounded-lg border border-slate-700 bg-[#0c1320] px-3 py-2.5 text-xs text-slate-300 outline-none focus:border-sky-500"
        >
          <option value="All">All Projects</option>

          <option
            v-for="project in projects"
            :key="project.id"
            :value="String(project.id)"
          >
            {{ project.name }}
          </option>
        </select>

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
      </div>
    </section>

    <!-- Vulnerability table -->
    <section
      class="overflow-hidden rounded-xl border border-slate-800 bg-[#111827]"
    >
      <div
        class="flex items-center justify-between border-b border-slate-800 px-5 py-4"
      >
        <div>
          <h2 class="text-sm font-semibold text-white">
            Detected Vulnerabilities
          </h2>

          <p class="mt-1 text-xs text-slate-500">
            Individual dependency security findings
          </p>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="pending" class="px-5 py-14 text-center">
        <div
          class="mx-auto h-6 w-6 animate-spin rounded-full border-2 border-slate-600 border-t-sky-400"
        />

        <p class="mt-4 text-sm text-slate-400">Loading vulnerabilities...</p>
      </div>

      <!-- Error -->
      <div v-else-if="error" class="px-5 py-12 text-center">
        <p class="text-sm font-medium text-red-400">
          Failed to load vulnerabilities
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

      <!-- Table -->
      <template v-else>
        <div class="overflow-x-auto">
          <table class="w-full min-w-[850px] text-left">
            <thead class="bg-slate-900/60">
              <tr
                class="text-[10px] font-medium uppercase tracking-wider text-slate-500"
              >
                <th class="px-5 py-4">Vulnerability / Package</th>
                <th class="px-5 py-4">Project</th>
                <th class="px-5 py-4">Ecosystem</th>
                <th class="px-5 py-4">Severity</th>
                <th class="px-5 py-4">Version</th>
                <th class="px-5 py-4">Status</th>
              </tr>
            </thead>

            <tbody class="divide-y divide-slate-800">
              <tr
                v-for="item in vulnerabilities"
                :key="item.id"
                class="transition hover:bg-slate-800/30"
              >
                <td class="px-5 py-4">
                  <a
                    v-if="item.primary_url"
                    :href="item.primary_url"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="text-xs font-medium text-sky-400 hover:text-sky-300"
                  >
                    {{ vulnerabilityIdentifier(item) }}
                  </a>

                  <p v-else class="text-xs font-medium text-sky-400">
                    {{ vulnerabilityIdentifier(item) }}
                  </p>

                  <p class="mt-1.5 text-xs font-medium text-white">
                    {{ item.package || item.title || "Unknown package" }}
                  </p>
                </td>

                <td class="px-5 py-4 text-xs text-slate-300">
                  {{ item.project?.name || "Unknown project" }}
                </td>

                <td class="px-5 py-4">
                  <span
                    class="rounded-md border border-slate-700 bg-slate-800/60 px-2 py-1 text-[10px] text-slate-300"
                  >
                    {{ item.ecosystem || "Unknown" }}
                  </span>
                </td>

                <td class="px-5 py-4">
                  <span
                    class="rounded-md px-2 py-1 text-[10px] font-medium"
                    :class="severityColor(item.severity)"
                  >
                    {{ titleCase(item.severity) }}
                  </span>
                </td>

                <td class="px-5 py-4">
                  <p class="text-xs text-slate-400">
                    {{ item.installed_version || "—" }}
                  </p>

                  <p
                    v-if="item.fixed_version"
                    class="mt-1 text-[10px] text-emerald-400"
                  >
                    Fix: {{ item.fixed_version }}
                  </p>

                  <p v-else class="mt-1 text-[10px] text-slate-500">
                    No fix version
                  </p>
                </td>

                <td class="px-5 py-4">
                  <div class="flex items-center gap-1.5">
                    <span
                      class="h-1.5 w-1.5 rounded-full bg-current"
                      :class="statusColor(item.status)"
                    />

                    <span class="text-xs" :class="statusColor(item.status)">
                      {{ titleCase(item.status) }}
                    </span>
                  </div>
                </td>
              </tr>

              <tr v-if="vulnerabilities.length === 0">
                <td colspan="6" class="px-5 py-12 text-center">
                  <p class="text-sm text-slate-400">No vulnerabilities found</p>

                  <p class="mt-1 text-xs text-slate-600">
                    Try changing your search or filters.
                  </p>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <!-- Responsive numbered pagination -->
        <div
          class="flex flex-col gap-3 border-t border-slate-800 px-4 py-4 sm:flex-row sm:items-center sm:justify-between sm:px-5"
        >
          <p class="text-center text-xs text-slate-500 sm:text-left">
            Page {{ pagination.current_page }} of {{ pagination.last_page }}
            <span class="mx-1 text-slate-700">·</span>
            {{ pagination.total }} findings
          </p>

          <div
            class="w-full max-w-full overflow-x-auto sm:w-auto sm:overflow-visible"
          >
            <div
              class="mx-auto flex w-max min-w-full items-center justify-center gap-1 sm:mx-0 sm:min-w-0 sm:justify-end"
            >
              <!-- Previous -->
              <button
                type="button"
                :disabled="currentPage <= 1 || pending"
                aria-label="Previous page"
                class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-700 text-xs text-slate-300 transition hover:border-sky-500 hover:text-sky-400 disabled:cursor-not-allowed disabled:opacity-30"
                @click="changePage(currentPage - 1)"
              >
                ←
              </button>

              <!-- Page numbers -->
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

              <!-- Next -->
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
