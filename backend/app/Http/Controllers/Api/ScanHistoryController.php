<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Models\ScanRun;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\DB;
use Illuminate\Validation\Rule;

class ScanHistoryController extends Controller
{
    public function index(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'q' => ['nullable', 'string', 'max:255'],

            'status' => [
                'nullable',
                Rule::in([
                    'queued',
                    'preparing',
                    'scanning',
                    'processing',
                    'completed',
                    'partial',
                    'failed',
                    'cancelled',
                    'unsupported',
                ]),
            ],

            'trigger_type' => [
                'nullable',
                Rule::in([
                    'manual',
                    'discovery',
                    'scheduled',
                    'image_updated',
                    'remediation',
                ]),
            ],

            'project_id' => [
                'nullable',
                'integer',
                'exists:projects,id',
            ],

            'per_page' => ['nullable', 'integer', 'min:1', 'max:100'],
        ]);

        // Global summary, independent of active filters.
        $totalScans = ScanRun::count();

        $completedCount = ScanRun::where('status', 'completed')->count();
        $partialCount = ScanRun::where('status', 'partial')->count();
        $failedCount = ScanRun::where('status', 'failed')->count();

        // Count scan-finding relationships, not unique canonical findings.
        $findingsDetected = DB::table('scan_run_findings')->count();

        $summary = [
            'total_scans' => $totalScans,
            'completed' => $completedCount,
            'partial' => $partialCount,
            'failed' => $failedCount,
            'findings_detected' => $findingsDetected,
        ];

        // Build paginated scan history.
        $query = ScanRun::query()
            ->with([
                'project:id,name,docker_project_name',

                'ecosystems:id,scan_run_id,ecosystem,manifest_file,status,error_message',

                // Used as a fallback for historical scans created before
                // scan_run_ecosystems was implemented.
                'findings:id,metadata',
            ])
            ->withCount('findings');

        if (!empty($validated['status'])) {
            $query->where('status', $validated['status']);
        }

        if (!empty($validated['trigger_type'])) {
            $query->where('trigger_type', $validated['trigger_type']);
        }

        if (!empty($validated['project_id'])) {
            $query->where('project_id', $validated['project_id']);
        }

        if (!empty($validated['q'])) {
            $search = trim($validated['q']);

            $query->where(function (Builder $searchQuery) use ($search) {
                $searchQuery->whereHas(
                    'project',
                    function (Builder $projectQuery) use ($search) {
                        $projectQuery->where('name', 'ILIKE', "%{$search}%")
                            ->orWhere(
                                'docker_project_name',
                                'ILIKE',
                                "%{$search}%"
                            );
                    }
                );

                if (ctype_digit($search)) {
                    $searchQuery->orWhere('id', (int) $search);
                }
            });
        }

        $perPage = $validated['per_page'] ?? 15;

        $paginator = $query
            ->orderByDesc('started_at')
            ->orderByDesc('id')
            ->paginate($perPage)
            ->withQueryString();

        $data = $paginator->getCollection()->map(
            function (ScanRun $scan) {
                $ecosystemRecords = $scan->ecosystems;

                $ecosystemNames = $ecosystemRecords
                    ->pluck('ecosystem')
                    ->filter()
                    ->unique(fn ($value) => strtolower($value))
                    ->values();

                // Legacy fallback: derive ecosystem information from
                // finding metadata when no explicit records exist.
                if ($ecosystemNames->isEmpty()) {
                    $ecosystemNames = $scan->findings
                        ->map(fn ($finding) =>
                            $finding->metadata['ecosystem'] ?? null
                        )
                        ->filter(fn ($value) =>
                            is_string($value) && trim($value) !== ''
                        )
                        ->unique(fn ($value) => strtolower($value))
                        ->values();
                }

                $formatEcosystem = static function (string $value): string {
                    return match (strtolower(trim($value))) {
                        'composer' => 'Composer',
                        'npm' => 'npm',
                        'go' => 'Go',
                        'python' => 'Python',
                        default => ucfirst(trim($value)),
                    };
                };

                $ecosystemLabels = $ecosystemNames
                    ->map($formatEcosystem)
                    ->values();

                $startedAt = $scan->started_at;
                $completedAt = $scan->completed_at;

                $durationMs = (
                    $startedAt && $completedAt
                )
                    ? (int) $startedAt->diffInMilliseconds($completedAt)
                    : null;

                return [
                    'id' => $scan->id,

                    'project' => $scan->project ? [
                        'id' => $scan->project->id,
                        'name' => $scan->project->name,
                        'docker_project_name' => $scan->project->docker_project_name,
                    ] : null,

                    'trigger_type' => $scan->trigger_type,
                    'status' => $scan->status,

                    'scanner_version' => $scan->scanner_version,
                    'source_commit_sha' => $scan->source_commit_sha,

                    'ecosystems' => $ecosystemLabels,
                    'ecosystems_count' => $ecosystemLabels->count(),

                    'ecosystem_results' => $ecosystemRecords
                        ->map(fn ($item) => [
                            'ecosystem' => $formatEcosystem($item->ecosystem),
                            'manifest_file' => $item->manifest_file,
                            'status' => $item->status,
                            'error_message' => $item->error_message,
                        ])
                        ->values(),

                    'findings_count' => (int) $scan->findings_count,

                    'started_at' => $startedAt,
                    'completed_at' => $completedAt,
                    'duration_ms' => $durationMs,

                    'error_message' => $scan->error_message,
                    'created_at' => $scan->created_at,
                ];
            }
        )->values();

        $projects = \App\Models\Project::query()
            ->whereHas('scanRuns')
            ->orderBy('name')
            ->get(['id', 'name']);

        return response()->json([
            'message' => 'Scan history retrieved successfully.',
            'data' => $data,
            'summary' => $summary,

            'filters' => [
                'projects' => $projects,
            ],

            'pagination' => [
                'current_page' => $paginator->currentPage(),
                'last_page' => $paginator->lastPage(),
                'per_page' => $paginator->perPage(),
                'total' => $paginator->total(),
            ],
        ]);
    }
}
