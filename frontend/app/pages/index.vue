<script setup lang="ts">
interface Project {
  id: number;
  name: string;
  docker_project_name: string;
  status: string;
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
  } | null;
}

interface Scan {
  id: number;
  project: {
    id: number;
    name: string;
  } | null;
  trigger_type: string;
  status: string;
  findings_count: number;
  started_at: string | null;
  created_at: string | null;
}
interface TrendScan {
  id: number;
  date: string;
  project: string | null;
  findings_count: number;
  status: string;
}
interface AgentSummary {
  total: number;
  online: number;
  offline: number;
  pending: number;
}
interface VulnerabilitySummary {
  total: number;
  critical: number;
  high: number;
  medium: number;
  low: number;
  unknown: number;
}

const api = useApi();

const loading = ref(true);
const error = ref("");
const refreshError = ref("");
const lastUpdatedAt = ref<Date | null>(null);
const projects = ref<Project[]>([]);
const recentScans = ref<Scan[]>([]);
const vulnerabilityTrend = ref<TrendScan[]>([]);
const agentSummary = ref<AgentSummary>({
  total: 0,
  online: 0,
  offline: 0,
  pending: 0,
});

const monitoringStatus = computed(() => {
  const agents = agentSummary.value;

  if (agents.total === 0) {
    return {
      label: "No agents registered",
      dot: "bg-slate-500",
      text: "text-slate-400",
    };
  }

  if (agents.online === agents.total) {
    return {
      label: "All agents online",
      dot: "bg-emerald-400",
      text: "text-emerald-400",
    };
  }

  if (agents.pending === agents.total) {
    return {
      label: "Pending",
      dot: "bg-yellow-400",
      text: "text-yellow-400",
    };
  }

  if (agents.offline === agents.total) {
    return {
      label: "All agents offline",
      dot: "bg-red-400",
      text: "text-red-400",
    };
  }

  return {
    label: "Partial connectivity",
    dot: "bg-yellow-400",
    text: "text-yellow-400",
  };
});
const summary = ref<VulnerabilitySummary>({
  total: 0,
  critical: 0,
  high: 0,
  medium: 0,
  low: 0,
  unknown: 0,
});

const stats = computed(() => [
  {
    label: "Total Vulnerabilities",
    value: summary.value.total,
    detail: "Across all monitored projects",
    icon: "◉",
    color: "text-sky-400",
    bg: "bg-sky-400/10",
  },
  {
    label: "Critical",
    value: summary.value.critical,
    detail: "Require immediate attention",
    icon: "!",
    color: "text-red-400",
    bg: "bg-red-400/10",
  },
  {
    label: "High",
    value: summary.value.high,
    detail: "High priority findings",
    icon: "↑",
    color: "text-orange-400",
    bg: "bg-orange-400/10",
  },
  {
    label: "Monitored Projects",
    value: projects.value.length,
    detail: "Docker Compose projects",
    icon: "▦",
    color: "text-emerald-400",
    bg: "bg-emerald-400/10",
  },
]);

const severityDefinitions = [
  { label: "Critical", key: "critical" as const, color: "bg-red-500" },
  { label: "High", key: "high" as const, color: "bg-orange-500" },
  { label: "Medium", key: "medium" as const, color: "bg-yellow-500" },
  { label: "Low", key: "low" as const, color: "bg-sky-500" },
  { label: "Unknown", key: "unknown" as const, color: "bg-slate-500" },
];

const severities = computed(() => {
  return severityDefinitions
    .map((item) => {
      const count = summary.value[item.key];

      return {
        ...item,
        count,
        percentage:
          summary.value.total > 0
            ? Math.round((count / summary.value.total) * 100)
            : 0,
      };
    })
    .filter((item) => item.count > 0 || item.key !== "unknown");
});

const severityBarItems = computed(() =>
  severities.value.filter((item) => item.count > 0),
);

const severityBarStyle = (item: (typeof severityBarItems.value)[number]) => ({
  width: `${item.percentage}%`,
});
const trendChart = computed(() => {
  const items = vulnerabilityTrend.value;
  const width = 720;
  const height = 220;
  const paddingX = 36;
  const paddingY = 24;

  const maxValue = Math.max(5, ...items.map((item) => item.findings_count));

  const points = items.map((item, index) => {
    const x =
      items.length <= 1
        ? width / 2
        : paddingX + (index / (items.length - 1)) * (width - paddingX * 2);

    const y =
      height -
      paddingY -
      (item.findings_count / maxValue) * (height - paddingY * 2);

    return {
      ...item,
      x,
      y,
      label: new Intl.DateTimeFormat("en-US", {
        month: "short",
        day: "numeric",
      }).format(new Date(item.date)),
    };
  });

  return {
    points,
    line: points.map((point) => `${point.x},${point.y}`).join(" "),
    area: points.length
      ? `${points[0].x},${height - paddingY} ${points.map((point) => `${point.x},${point.y}`).join(" ")} ${points[points.length - 1].x},${height - paddingY}`
      : "",
    maxValue,
    width,
    height,
  };
});

const formatStatus = (status: string) => {
  if (!status) return "Unknown";

  return status
    .replaceAll("_", " ")
    .replace(/\b\w/g, (char) => char.toUpperCase());
};

const statusColor = (status: string) => {
  switch (status?.toLowerCase()) {
    case "completed":
    case "active":
    case "online":
      return "text-emerald-400";

    case "partial":
    case "degraded":
    case "pending":
      return "text-yellow-400";

    case "failed":
    case "offline":
    case "missing":
      return "text-red-400";

    case "processing":
    case "scanning":
    case "queued":
    case "preparing":
      return "text-sky-400";

    default:
      return "text-slate-400";
  }
};

const formatDate = (date: string | null) => {
  if (!date) return "—";

  const parsed = new Date(date);

  if (Number.isNaN(parsed.getTime())) return "—";

  return new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(parsed);
};

const getScanDate = (scan: Scan) => scan.started_at ?? scan.created_at;

const getApiError = (err: unknown) => {
  if (err && typeof err === "object" && "data" in err) {
    const data = (err as { data?: { message?: string } }).data;

    if (data?.message) return data.message;
  }

  if (err instanceof Error) return err.message;

  return "Unable to retrieve dashboard data.";
};
const isRefreshing = ref(false);
const hasLoaded = ref(false);
const fetchDashboard = async () => {
  if (isRefreshing.value) return;

  isRefreshing.value = true;

  if (!hasLoaded.value) {
    loading.value = true;
  }

  error.value = "";

  try {
    const [
      projectResponse,
      vulnerabilityResponse,
      scanResponse,
      agentResponse,
      trendResponse,
    ] = await Promise.all([
      api.get<{
        data: Project[];
        meta: { total: number };
      }>("/api/projects"),

      api.get<{
        summary: VulnerabilitySummary;
      }>("/api/vulnerabilities?status=active&per_page=1"),

      api.get<{
        data: Scan[];
      }>("/api/scans?per_page=5"),

      api.get<{
        summary: AgentSummary;
      }>("/api/agents"),
      api.get<{
        data: TrendScan[];
      }>("/api/dashboard/vulnerability-trend"),
    ]);

    projects.value = projectResponse.data;
    summary.value = vulnerabilityResponse.summary;
    recentScans.value = scanResponse.data;
    agentSummary.value = agentResponse.summary;
    vulnerabilityTrend.value = trendResponse.data;
    refreshError.value = "";
    hasLoaded.value = true;
    lastUpdatedAt.value = new Date();
  } catch (err) {
    const message = getApiError(err);

    if (hasLoaded.value) {
      refreshError.value = message;
    } else {
      error.value = message;
    }
  } finally {
    loading.value = false;
    isRefreshing.value = false;
  }
};

let refreshInterval: ReturnType<typeof setInterval> | undefined;

onMounted(() => {
  fetchDashboard();

  refreshInterval = setInterval(() => {
    fetchDashboard();
  }, 30_000);
});

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval);
  }
});
</script>

<template>
  <div class="space-y-8">
    <!-- Page introduction -->
    <section class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <p class="text-xs font-medium uppercase tracking-[0.18em] text-sky-400">
          Security overview
        </p>

        <h1 class="mt-2 text-2xl font-semibold text-white">
          Infrastructure Overview
        </h1>

        <p class="mt-2 text-sm text-slate-400">
          Monitor vulnerabilities across your Docker Compose projects.
        </p>
        <p v-if="lastUpdatedAt" class="mt-2 text-xs text-slate-500">
          Last updated:
          {{
            lastUpdatedAt.toLocaleTimeString("id-ID", {
              hour: "2-digit",
              minute: "2-digit",
              second: "2-digit",
            })
          }}
        </p>
        <p v-if="refreshError" class="mt-1 text-xs text-red-400">
          Refresh failed: {{ refreshError }}
        </p>
      </div>

      <div class="flex items-center gap-3">
        <div
          class="flex items-center gap-2 rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-2"
        >
          <span class="h-2 w-2 rounded-full" :class="monitoringStatus.dot" />
          <span class="text-xs" :class="monitoringStatus.text">
            {{ monitoringStatus.label }}
          </span>
        </div>

        <button
          type="button"
          :disabled="isRefreshing"
          @click="fetchDashboard"
          class="flex items-center gap-2 rounded-lg border border-slate-700 bg-slate-900/60 px-3 py-2 text-xs font-medium text-slate-300 transition hover:border-sky-500/50 hover:text-sky-400 disabled:cursor-not-allowed disabled:opacity-50"
        >
          <span :class="{ 'animate-spin': isRefreshing }">↻</span>
          {{ isRefreshing ? "Refreshing..." : "Refresh" }}
        </button>
      </div>
    </section>

    <!-- Loading -->
    <div
      v-if="loading"
      class="rounded-xl border border-slate-800 bg-[#111827] px-6 py-16 text-center"
    >
      <div
        class="mx-auto h-7 w-7 animate-spin rounded-full border-2 border-slate-700 border-t-sky-400"
      />
      <p class="mt-4 text-sm text-slate-400">Loading dashboard data...</p>
    </div>

    <!-- Error -->
    <div
      v-else-if="error"
      class="rounded-xl border border-red-500/20 bg-red-500/5 p-6"
    >
      <div class="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h2 class="font-medium text-red-400">Failed to load dashboard</h2>
          <p class="mt-2 text-sm text-slate-400">
            {{ error }}
          </p>
        </div>

        <button
          class="rounded-lg border border-slate-700 bg-slate-800 px-4 py-2 text-xs font-medium text-white transition hover:bg-slate-700"
          @click="fetchDashboard"
        >
          Retry
        </button>
      </div>
    </div>

    <template v-else>
      <!-- Statistics -->
      <section class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <div
          v-for="stat in stats"
          :key="stat.label"
          class="rounded-xl border border-slate-800 bg-[#111827] p-5 transition hover:border-slate-700"
        >
          <div class="flex items-center justify-between">
            <p class="text-sm text-slate-400">{{ stat.label }}</p>

            <div
              class="flex h-9 w-9 items-center justify-center rounded-lg text-lg"
              :class="[stat.bg, stat.color]"
            >
              {{ stat.icon }}
            </div>
          </div>

          <p class="mt-5 text-3xl font-semibold tracking-tight text-white">
            {{ stat.value.toLocaleString() }}
          </p>

          <p class="mt-2 text-xs text-slate-500">
            {{ stat.detail }}
          </p>
        </div>
      </section>

      <!-- Main content -->
      <section class="grid grid-cols-1 gap-5 xl:grid-cols-5">
        <!-- Severity distribution -->
        <div
          class="rounded-xl border border-slate-800 bg-[#111827] p-6 xl:col-span-2"
        >
          <div class="flex items-center justify-between">
            <div>
              <h2 class="font-semibold text-white">Severity Distribution</h2>

              <p class="mt-1 text-xs text-slate-500">
                Vulnerability breakdown by severity
              </p>
            </div>

            <span
              class="rounded-md bg-slate-800 px-2 py-1 text-xs text-slate-400"
            >
              {{ summary.total.toLocaleString() }} findings
            </span>
          </div>

          <div class="mt-7 flex h-3 overflow-hidden rounded-full bg-slate-800">
            <div
              v-for="item in severityBarItems"
              :key="item.label"
              :class="item.color"
              :style="severityBarStyle(item)"
            />
          </div>

          <div class="mt-7 space-y-5">
            <div
              v-for="item in severities"
              :key="item.label"
              class="flex items-center justify-between"
            >
              <div class="flex items-center gap-3">
                <span class="h-2.5 w-2.5 rounded-full" :class="item.color" />

                <span class="text-sm text-slate-300">
                  {{ item.label }}
                </span>
              </div>

              <div class="flex items-center gap-4">
                <span class="text-sm font-medium text-white">
                  {{ item.count }}
                </span>

                <span class="w-10 text-right text-xs text-slate-500">
                  {{ item.percentage }}%
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- Project monitoring -->
        <div
          class="rounded-xl border border-slate-800 bg-[#111827] p-6 xl:col-span-3"
        >
          <div class="flex items-center justify-between">
            <div>
              <h2 class="font-semibold text-white">Monitored Projects</h2>

              <p class="mt-1 text-xs text-slate-500">
                Security status of discovered projects
              </p>
            </div>

            <NuxtLink
              to="/projects"
              class="text-xs font-medium text-sky-400 hover:text-sky-300"
            >
              View all →
            </NuxtLink>
          </div>

          <div v-if="projects.length" class="mt-5 divide-y divide-slate-800">
            <NuxtLink
              v-for="project in projects"
              :key="project.id"
              :to="`/projects/${project.id}`"
              class="flex flex-wrap items-center justify-between gap-4 py-5 transition hover:bg-slate-800/20"
            >
              <div class="flex items-center gap-3">
                <div
                  class="flex h-10 w-10 items-center justify-center rounded-lg bg-slate-800 text-sky-400"
                >
                  ▦
                </div>

                <div>
                  <p class="text-sm font-medium text-white">
                    {{ project.name }}
                  </p>

                  <p class="mt-1 text-xs text-slate-500">
                    Docker Compose · {{ formatStatus(project.status) }}
                  </p>

                  <div
                    v-if="project.ecosystems.length"
                    class="mt-2 flex flex-wrap gap-2"
                  >
                    <span
                      v-for="ecosystem in project.ecosystems"
                      :key="ecosystem"
                      class="rounded bg-slate-800 px-2 py-1 text-[10px] text-slate-400"
                    >
                      {{ ecosystem }}
                    </span>
                  </div>
                </div>
              </div>

              <div class="text-right">
                <p class="text-lg font-semibold text-white">
                  {{ project.findings.total }}
                </p>

                <p class="text-[10px] text-slate-500">Findings</p>

                <p
                  class="mt-2 text-xs"
                  :class="statusColor(project.last_scan?.status ?? '')"
                >
                  {{
                    project.last_scan
                      ? formatStatus(project.last_scan.status)
                      : "Not scanned"
                  }}
                </p>
              </div>
            </NuxtLink>
          </div>

          <div v-else class="py-12 text-center text-sm text-slate-500">
            No monitored projects found.
          </div>
        </div>
      </section>
      <!-- Vulnerability Trend -->
      <section class="rounded-xl border border-slate-800 bg-[#111827] p-6">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 class="font-semibold text-white">Vulnerability Trend</h2>
            <p class="mt-1 text-xs text-slate-500">
              Vulnerability findings detected across recent scans
            </p>
          </div>

          <span
            class="rounded-md bg-slate-800 px-2 py-1 text-xs text-slate-400"
          >
            Last {{ vulnerabilityTrend.length }} scans
          </span>
        </div>

        <div v-if="vulnerabilityTrend.length" class="mt-6 overflow-x-auto">
          <svg
            :viewBox="`0 0 ${trendChart.width} ${trendChart.height}`"
            class="h-auto min-w-[560px] w-full"
            role="img"
            aria-label="Vulnerability findings trend chart"
          >
            <defs>
              <linearGradient id="trendGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stop-color="#38bdf8" stop-opacity="0.25" />
                <stop offset="100%" stop-color="#38bdf8" stop-opacity="0" />
              </linearGradient>
            </defs>

            <line
              v-for="line in 4"
              :key="line"
              x1="36"
              :y1="24 + (line - 1) * 57"
              x2="684"
              :y2="24 + (line - 1) * 57"
              stroke="#334155"
              stroke-dasharray="4 5"
            />

            <polygon
              v-if="trendChart.points.length > 1"
              :points="trendChart.area"
              fill="url(#trendGradient)"
            />

            <polyline
              v-if="trendChart.points.length > 1"
              :points="trendChart.line"
              fill="none"
              stroke="#38bdf8"
              stroke-width="3"
              stroke-linecap="round"
              stroke-linejoin="round"
            />

            <g v-for="(point, index) in trendChart.points" :key="point.id">
              <circle
                :cx="point.x"
                :cy="point.y"
                r="6"
                fill="#0f172a"
                stroke="#38bdf8"
                stroke-width="2.5"
                class="cursor-pointer transition-all hover:fill-sky-400"
              >
                <title>
                  {{ point.project ?? "Unknown project" }}
                  | {{ point.findings_count }} vulnerabilities |
                  {{ formatDate(point.date) }} |
                  {{ formatStatus(point.status) }}
                </title>
              </circle>

              <text
                :x="point.x"
                :y="trendChart.height - 3"
                fill="#64748b"
                font-size="11"
                text-anchor="middle"
              >
                {{ point.label }}
              </text>

              <text
                :x="point.x"
                :y="point.y - 12"
                fill="#cbd5e1"
                font-size="11"
                text-anchor="middle"
              >
                {{ point.findings_count }}
              </text>
            </g>
          </svg>
        </div>

        <div v-else class="py-12 text-center">
          <p class="text-sm text-slate-400">
            No vulnerability trend data available.
          </p>
          <p class="mt-1 text-xs text-slate-600">
            Trend data will appear after dependency scans are completed.
          </p>
        </div>
      </section>
      <!-- Recent scans -->
      <section
        class="overflow-hidden rounded-xl border border-slate-800 bg-[#111827]"
      >
        <div
          class="flex items-center justify-between border-b border-slate-800 px-6 py-5"
        >
          <div>
            <h2 class="font-semibold text-white">Recent Scan Activity</h2>

            <p class="mt-1 text-xs text-slate-500">
              Latest dependency security scans
            </p>
          </div>

          <NuxtLink
            to="/scans"
            class="text-xs font-medium text-sky-400 hover:text-sky-300"
          >
            View history →
          </NuxtLink>
        </div>

        <div v-if="recentScans.length" class="overflow-x-auto">
          <table class="w-full text-left">
            <thead class="bg-slate-900/50">
              <tr class="text-[11px] uppercase tracking-wider text-slate-500">
                <th class="px-6 py-4 font-medium">Project</th>
                <th class="px-6 py-4 font-medium">Trigger</th>
                <th class="px-6 py-4 font-medium">Findings</th>
                <th class="px-6 py-4 font-medium">Status</th>
                <th class="px-6 py-4 font-medium">Date</th>
              </tr>
            </thead>

            <tbody class="divide-y divide-slate-800">
              <tr
                v-for="scan in recentScans"
                :key="scan.id"
                class="text-sm transition hover:bg-slate-800/30"
              >
                <td class="px-6 py-4 font-medium text-white">
                  {{ scan.project?.name ?? "Unknown project" }}
                </td>

                <td class="px-6 py-4 text-slate-400">
                  {{ formatStatus(scan.trigger_type) }}
                </td>

                <td class="px-6 py-4 text-slate-300">
                  {{ scan.findings_count }}
                </td>

                <td class="px-6 py-4">
                  <span
                    class="rounded-full px-2.5 py-1 text-xs"
                    :class="{
                      'bg-emerald-400/10 text-emerald-400':
                        scan.status === 'completed',
                      'bg-yellow-400/10 text-yellow-400':
                        scan.status === 'partial',
                      'bg-red-400/10 text-red-400': scan.status === 'failed',
                      'bg-sky-400/10 text-sky-400': ![
                        'completed',
                        'partial',
                        'failed',
                      ].includes(scan.status),
                    }"
                  >
                    {{ formatStatus(scan.status) }}
                  </span>
                </td>

                <td class="px-6 py-4 text-xs text-slate-500">
                  {{ formatDate(getScanDate(scan)) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-else class="px-6 py-12 text-center text-sm text-slate-500">
          No scan activity found.
        </div>
      </section>

      <p class="text-center text-[11px] text-slate-600">
        Live data from monitored infrastructure
      </p>
    </template>
  </div>
</template>
