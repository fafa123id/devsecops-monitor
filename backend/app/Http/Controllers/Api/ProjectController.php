<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Models\Project;
use App\Models\ScanFinding;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Http\JsonResponse;

class ProjectController extends Controller
{
    public function index(): JsonResponse
    {
        // Active vulnerability findings only.
        $activeVulnerabilities = static function (Builder $query): Builder {
            return $query
                ->where('finding_type', 'vulnerability')
                ->whereIn('status', [
                    'new',
                    'existing',
                    'reopened',
                ]);
        };

        $projects = Project::query()
            ->with([
                'dockerHost:id,name,hostname,status,last_heartbeat_at',
                'latestScanRun:id,project_id,trigger_type,status,started_at,completed_at,scanner_version',
            ])
            ->withCount([
                // Containers currently detected by the agent.
                'containers as containers_total' => static fn(Builder $query) =>
                $query->where('container_state', '!=', 'missing'),

                'containers as containers_running' => static fn(Builder $query) =>
                $query->where('container_state', 'running'),

                'containers as containers_missing' => static fn(Builder $query) =>
                $query->where('container_state', 'missing'),

                // Active vulnerability totals.
                'findings as findings_total' => $activeVulnerabilities,

                'findings as findings_critical' => static fn(Builder $query) =>
                $activeVulnerabilities($query)->where('severity', 'critical'),

                'findings as findings_high' => static fn(Builder $query) =>
                $activeVulnerabilities($query)->where('severity', 'high'),

                'findings as findings_medium' => static fn(Builder $query) =>
                $activeVulnerabilities($query)->where('severity', 'medium'),

                'findings as findings_low' => static fn(Builder $query) =>
                $activeVulnerabilities($query)->where('severity', 'low'),

                'findings as findings_unknown' => static fn(Builder $query) =>
                $activeVulnerabilities($query)->where('severity', 'unknown'),
            ])
            ->orderBy('name')
            ->get();

        // Retrieve ecosystems recorded in vulnerability metadata.
        // Include resolved findings so previously detected ecosystems
        // remain visible even when their findings have been resolved.
        $ecosystemsByProject = collect();

        if ($projects->isNotEmpty()) {
            $ecosystemsByProject = ScanFinding::query()
                ->whereIn('project_id', $projects->modelKeys())
                ->where('finding_type', 'vulnerability')
                ->get(['project_id', 'metadata'])
                ->groupBy('project_id')
                ->map(function ($findings) {
                    return $findings
                        ->map(
                            fn($finding) =>
                            $finding->metadata['ecosystem'] ?? null
                        )
                        ->filter(
                            fn($ecosystem) =>
                            is_string($ecosystem) && trim($ecosystem) !== ''
                        )
                        ->map(fn($ecosystem) => ucfirst(strtolower(trim($ecosystem))))
                        ->unique()
                        ->sort()
                        ->values();
                });
        }

        $data = $projects->map(function (Project $project) use ($ecosystemsByProject) {
            $latestScan = $project->latestScanRun;
            $host = $project->dockerHost;

            $observedContainers = $project->containers_total
                + $project->containers_missing;

            return [
                'id' => $project->id,
                'name' => $project->name,
                'docker_project_name' => $project->docker_project_name,
                'status' => $project->status,

                'scan_source' => $project->scan_source,
                'weekly_scan_enabled' => $project->weekly_scan_enabled,

                'source_path_reference' => $project->source_path_reference,

                'host' => $host ? [
                    'id' => $host->id,
                    'name' => $host->name,
                    'hostname' => $host->hostname,
                    'status' => $host->status,
                    'last_heartbeat_at' => $host->last_heartbeat_at,
                ] : null,

                'containers' => [
                    'running' => (int) $project->containers_running,
                    'total' => (int) $project->containers_total,
                    'missing' => (int) $project->containers_missing,
                    'observed_total' => (int) $observedContainers,
                ],

                'ecosystems' => $ecosystemsByProject
                    ->get($project->id, collect())
                    ->values(),

                'findings' => [
                    'total' => (int) $project->findings_total,
                    'critical' => (int) $project->findings_critical,
                    'high' => (int) $project->findings_high,
                    'medium' => (int) $project->findings_medium,
                    'low' => (int) $project->findings_low,
                    'unknown' => (int) $project->findings_unknown,
                ],

                'last_scan' => $latestScan ? [
                    'id' => $latestScan->id,
                    'trigger_type' => $latestScan->trigger_type,
                    'status' => $latestScan->status,
                    'started_at' => $latestScan->started_at,
                    'completed_at' => $latestScan->completed_at,
                    'scanner_version' => $latestScan->scanner_version,
                ] : null,

                'last_scanned_at' => $project->last_scanned_at,
                'last_seen_at' => $project->last_seen_at,
                'missing_since_at' => $project->missing_since_at,
            ];
        });

        return response()->json([
            'message' => 'Projects retrieved successfully.',
            'data' => $data,
            'meta' => [
                'total' => $data->count(),
            ],
        ]);
    }

    public function show(Project $project): JsonResponse
    {
        $project->load([
            'dockerHost:id,name,hostname,status,last_heartbeat_at',
            'containers:id,project_id,container_id,container_name,service_name,image_reference,container_state,last_seen_at',
            'latestScanRun:id,project_id,trigger_type,status,started_at,completed_at,scanner_version',
        ]);

        $activeStatuses = [
            'new',
            'existing',
            'reopened',
        ];

        // Aggregate active vulnerability findings by severity.
        $severityCounts = $project->findings()
            ->where('finding_type', 'vulnerability')
            ->whereIn('status', $activeStatuses)
            ->selectRaw('LOWER(severity) AS severity, COUNT(*) AS aggregate')
            ->groupByRaw('LOWER(severity)')
            ->pluck('aggregate', 'severity');

        $findings = [
            'total' => (int) $severityCounts->sum(),
            'critical' => (int) ($severityCounts['critical'] ?? 0),
            'high' => (int) ($severityCounts['high'] ?? 0),
            'medium' => (int) ($severityCounts['medium'] ?? 0),
            'low' => (int) ($severityCounts['low'] ?? 0),
            'unknown' => (int) ($severityCounts['unknown'] ?? 0),
        ];

        // Retrieve detected ecosystems from finding metadata.
        $ecosystems = $project->findings()
            ->where('finding_type', 'vulnerability')
            ->get(['metadata'])
            ->map(fn($finding) => $finding->metadata['ecosystem'] ?? null)
            ->filter(
                fn($ecosystem) =>
                is_string($ecosystem) && trim($ecosystem) !== ''
            )
            ->map(
                fn($ecosystem) =>
                ucfirst(strtolower(trim($ecosystem)))
            )
            ->unique()
            ->sort()
            ->values();

        $latestScan = $project->latestScanRun;
        $host = $project->dockerHost;

        $containers = $project->containers
            ->sortBy('container_name')
            ->values()
            ->map(fn($container) => [
                'id' => $container->id,
                'container_id' => $container->container_id,
                'name' => $container->container_name,
                'service_name' => $container->service_name,
                'image_reference' => $container->image_reference,
                'state' => $container->container_state,
                'last_seen_at' => $container->last_seen_at,
            ]);

        return response()->json([
            'message' => 'Project retrieved successfully.',
            'data' => [
                'id' => $project->id,
                'name' => $project->name,
                'docker_project_name' => $project->docker_project_name,
                'status' => $project->status,
                'scan_source' => $project->scan_source,
                'weekly_scan_enabled' => $project->weekly_scan_enabled,

                'host' => $host ? [
                    'id' => $host->id,
                    'name' => $host->name,
                    'hostname' => $host->hostname,
                    'status' => $host->status,
                    'last_heartbeat_at' => $host->last_heartbeat_at,
                ] : null,

                'containers' => $containers,
                'ecosystems' => $ecosystems,
                'findings' => $findings,

                'last_scan' => $latestScan ? [
                    'id' => $latestScan->id,
                    'trigger_type' => $latestScan->trigger_type,
                    'status' => $latestScan->status,
                    'started_at' => $latestScan->started_at,
                    'completed_at' => $latestScan->completed_at,
                    'scanner_version' => $latestScan->scanner_version,
                ] : null,

                'last_scanned_at' => $project->last_scanned_at,
                'last_seen_at' => $project->last_seen_at,
                'missing_since_at' => $project->missing_since_at,
            ],
        ]);
    }
}
