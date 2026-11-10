<?php

namespace App\Http\Controllers\Api\Agent;

use App\Http\Controllers\Controller;
use App\Models\ScanFinding;
use App\Models\ScanRun;
use App\Services\NotificationService;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\DB;
use Illuminate\Validation\Rule;

class ScanController extends Controller
{
    public function __invoke(Request $request)
    {
        $validated = $request->validate([
            'trigger_type' => [
                'required',
                'string',
                Rule::in([
                    'manual',
                    'discovery',
                    'scheduled',
                    'image_updated',
                    'remediation',
                ]),
            ],
            'scanner_version' => ['required', 'string', 'max:50'],

            'projects' => ['required', 'array', 'min:1'],
            'projects.*.docker_project_name' => [
                'required',
                'string',
                'max:255',
            ],
            'projects.*.ecosystems' => [
                'required',
                'array',
                'min:1',
            ],

            'projects.*.ecosystems.*.ecosystem' => [
                'required',
                'string',
                'max:50',
            ],
            'projects.*.ecosystems.*.manifest_file' => [
                'required',
                'string',
                'max:255',
            ],
            'projects.*.ecosystems.*.status' => [
                'required',
                'string',
                Rule::in(['completed', 'failed', 'unsupported']),
            ],
            'projects.*.ecosystems.*.error_message' => [
                'nullable',
                'string',
                'max:5000',
            ],
            'projects.*.ecosystems.*.findings' => [
                'present',
                'array',
            ],

            'projects.*.ecosystems.*.findings.*.finding_type' => [
                'sometimes',
                'required',
                'string',
                Rule::in([
                    'vulnerability',
                    'misconfiguration',
                    'secret',
                    'license',
                ]),
            ],
            'projects.*.ecosystems.*.findings.*.finding_id' => [
                'required',
                'string',
                'max:255',
            ],
            'projects.*.ecosystems.*.findings.*.vulnerability_id' => [
                'nullable',
                'string',
                'max:255',
            ],
            'projects.*.ecosystems.*.findings.*.package_name' => [
                'nullable',
                'string',
                'max:255',
            ],
            'projects.*.ecosystems.*.findings.*.installed_version' => [
                'nullable',
                'string',
                'max:255',
            ],
            'projects.*.ecosystems.*.findings.*.fixed_version' => [
                'nullable',
                'string',
                'max:255',
            ],
            'projects.*.ecosystems.*.findings.*.severity' => [
                'required',
                'string',
                Rule::in([
                    'critical',
                    'high',
                    'medium',
                    'low',
                    'unknown',
                ]),
            ],
            'projects.*.ecosystems.*.findings.*.title' => [
                'nullable',
                'string',
                'max:255',
            ],
            'projects.*.ecosystems.*.findings.*.description' => [
                'nullable',
                'string',
            ],
            'projects.*.ecosystems.*.findings.*.primary_url' => [
                'nullable',
                'url',
                'max:2000',
            ],
            'projects.*.ecosystems.*.findings.*.metadata' => [
                'nullable',
                'array',
            ],
        ]);

        $host = $request->user();
        $results = [];
        // Resolve all projects before processing any scan results.
        $projectNames = collect($validated['projects'])
            ->pluck('docker_project_name')
            ->unique()
            ->values();

        $projects = $host->projects()
            ->whereIn('docker_project_name', $projectNames)
            ->get()
            ->keyBy('docker_project_name');

        // Preflight: validate every project before creating ScanRun.
        foreach ($validated['projects'] as $projectData) {
            $projectName = $projectData['docker_project_name'];
            $project = $projects->get($projectName);

            if (!$project) {
                abort(404, "Project not found: {$projectName}");
            }

            if (
                !$project->scan_source ||
                !in_array($project->status, ['active', 'degraded'], true)
            ) {
                return response()->json([
                    'message' => "Scanning is not allowed for project {$projectName}.",
                    'project' => $projectName,
                    'status' => $project->status,
                    'scan_source' => (bool) $project->scan_source,
                ], 403);
            }

            if (
                $validated['trigger_type'] === 'scheduled' &&
                !$project->weekly_scan_enabled
            ) {
                return response()->json([
                    'message' => "Weekly scanning is disabled for project {$projectName}.",
                    'project' => $projectName,
                    'weekly_scan_enabled' => false,
                ], 403);
            }
        }

        // Resolve all projects belonging to this Docker host first.
        $projectNames = collect($validated['projects'])
            ->pluck('docker_project_name')
            ->unique()
            ->values();

        $projects = $host->projects()
            ->whereIn('docker_project_name', $projectNames)
            ->get()
            ->keyBy('docker_project_name');

        foreach ($validated['projects'] as $projectData) {
            $project = $projects->get(
                $projectData['docker_project_name']
            );
            $result = DB::transaction(function () use (
                $project,
                $projectData,
                $validated
            ) {
                $ecosystems = $projectData['ecosystems'];

                $completedEcosystems = collect($ecosystems)
                    ->where('status', 'completed')
                    ->count();

                $failedEcosystems = collect($ecosystems)
                    ->where('status', 'failed')
                    ->count();

                $unsupportedEcosystems = collect($ecosystems)
                    ->where('status', 'unsupported')
                    ->count();

                $totalEcosystems = count($ecosystems);

                $status = match (true) {
                    $completedEcosystems === $totalEcosystems => 'completed',
                    $completedEcosystems === 0
                        && $unsupportedEcosystems === $totalEcosystems
                    => 'unsupported',
                    $completedEcosystems === 0 => 'failed',
                    default => 'partial',
                };


                $errors = collect($ecosystems)
                    ->filter(
                        fn($ecosystem) =>
                        $ecosystem['status'] !== 'completed'
                    )
                    ->map(
                        fn($ecosystem) =>
                        $ecosystem['ecosystem']
                            . ' (' . $ecosystem['manifest_file'] . '): '
                            . (
                                $ecosystem['error_message']
                                ?? 'Ecosystem scan '
                                . $ecosystem['status']
                            )
                    )
                    ->values();

                $scanRun = ScanRun::create([
                    'project_id' => $project->id,
                    'trigger_type' => $validated['trigger_type'],
                    'status' => 'processing',
                    'started_at' => now(),
                    'scanner_version' => $validated['scanner_version'],
                ]);
                foreach ($ecosystems as $ecosystem) {
                    $scanRun->ecosystems()->updateOrCreate(
                        [
                            'ecosystem' => strtolower(trim($ecosystem['ecosystem'])),
                            'manifest_file' => $ecosystem['manifest_file'],
                        ],
                        [
                            'status' => $ecosystem['status'],
                            'error_message' => $ecosystem['error_message'] ?? null,
                        ]
                    );
                }

                $findingCount = 0;
                $processedFingerprints = [];

                foreach ($ecosystems as $ecosystem) {
                    if ($ecosystem['status'] !== 'completed') {
                        continue;
                    }

                    $ecosystemName = strtolower($ecosystem['ecosystem']);
                    $manifestFile = $ecosystem['manifest_file'];

                    $targetType = 'source_dependency';
                    $targetIdentifier = $ecosystemName . ':' . $manifestFile;

                    foreach ($ecosystem['findings'] as $finding) {
                        $findingType = $finding['finding_type']
                            ?? 'vulnerability';

                        $findingId = $finding['finding_id'];

                        $metadata = $finding['metadata'] ?? [];

                        // Simpan konteks ecosystem dan manifest.
                        $metadata['ecosystem'] = $ecosystemName;
                        $metadata['manifest_file'] = $manifestFile;

                        $location = $metadata['location']
                            ?? $metadata['primary_location']
                            ?? '';

                        if (! is_string($location)) {
                            $location = json_encode(
                                $location,
                                JSON_UNESCAPED_SLASHES
                                    | JSON_UNESCAPED_UNICODE
                            ) ?: '';
                        }

                        /*
                         * Versi package sengaja tidak dimasukkan ke
                         * fingerprint agar upgrade package tidak
                         * menghasilkan finding baru untuk advisory sama.
                         */
                        $fingerprintData = [
                            $targetType,
                            $targetIdentifier,
                            $ecosystemName,
                            $findingType,
                            $findingId,
                            $finding['package_name'] ?? '',
                            $location,
                        ];

                        $fingerprint = hash(
                            'sha256',
                            implode("\0", $fingerprintData)
                        );
                        if (isset($processedFingerprints[$fingerprint])) {
                            continue;
                        }

                        $processedFingerprints[$fingerprint] = true;
                        $existingFinding = ScanFinding::where(
                            'project_id',
                            $project->id
                        )
                            ->where('finding_fingerprint', $fingerprint)
                            ->first();

                        $now = now();

                        $scanFinding = ScanFinding::updateOrCreate(
                            [
                                'project_id' => $project->id,
                                'finding_fingerprint' => $fingerprint,
                            ],
                            [
                                'last_scan_run_id' => $scanRun->id,
                                'target_type' => $targetType,
                                'target_identifier' => $targetIdentifier,
                                'finding_type' => $findingType,
                                'finding_id' => $findingId,
                                'vulnerability_id' => $finding['vulnerability_id']
                                    ?? (
                                        $findingType === 'vulnerability'
                                        ? $findingId
                                        : null
                                    ),
                                'package_name' => $finding['package_name'] ?? null,
                                'installed_version' => $finding['installed_version'] ?? null,
                                'fixed_version' => $finding['fixed_version'] ?? null,
                                'severity' => strtolower($finding['severity']),
                                'title' => $finding['title'] ?? null,
                                'description' => $finding['description'] ?? null,
                                'primary_url' => $finding['primary_url'] ?? null,
                                'metadata' => $metadata ?: null,
                                'status' => ! $existingFinding
                                    ? 'new'
                                    : (
                                        $existingFinding->status === 'resolved'
                                        ? 'reopened'
                                        : 'existing'
                                    ),
                                'first_seen_at' => $existingFinding?->first_seen_at
                                    ?? $now,
                                'last_seen_at' => $now,
                            ]
                        );

                        $scanRun->findings()->syncWithoutDetaching([
                            $scanFinding->id,
                        ]);
                        if (in_array($scanFinding->status, ['new', 'reopened'], true)) {
                            app(NotificationService::class)->notifyFinding(
                                $scanFinding,
                                $scanRun,
                                $scanFinding->status
                            );
                        }
                        $findingCount++;
                    }
                }

                /*
                 * Resolve findings independently for each successfully
                 * completed ecosystem and manifest.
                 *
                 * Failed or unsupported ecosystems must never resolve
                 * their previous findings.
                 */

                $completedTargets = collect($ecosystems)
                    ->where('status', 'completed')
                    ->map(function ($ecosystem) {
                        return strtolower(trim($ecosystem['ecosystem']))
                            . ':'
                            . $ecosystem['manifest_file'];
                    })
                    ->unique()
                    ->values()
                    ->all();

                if ($completedTargets !== []) {
                    ScanFinding::where('project_id', $project->id)
                        ->where('target_type', 'source_dependency')
                        ->whereIn('target_identifier', $completedTargets)
                        ->whereIn('status', [
                            'new',
                            'existing',
                            'reopened',
                        ])
                        ->whereDoesntHave(
                            'scanRuns',
                            function ($query) use ($scanRun) {
                                $query->where(
                                    'scan_runs.id',
                                    $scanRun->id
                                );
                            }
                        )
                        ->update([
                            'status' => 'resolved',
                        ]);
                }

                $scanRun->update([
                    'status' => $status,
                    'completed_at' => now(),
                    'error_message' => $errors->isNotEmpty()
                        ? $errors->implode("\n")
                        : null,
                ]);
                if ($status === 'failed') {
                    app(NotificationService::class)->notifyScanFailed($scanRun);
                }
                $project->update([
                    'last_scanned_at' => now(),
                ]);

                return [
                    'project' => $project->docker_project_name,
                    'scan_run_id' => $scanRun->id,
                    'status' => $status,
                    'ecosystems_scanned' => $completedEcosystems,
                    'ecosystems_failed' => $failedEcosystems,
                    'ecosystems_unsupported' => $unsupportedEcosystems,
                    'findings_received' => $findingCount,
                ];
            });

            $results[] = $result;
        }

        return response()->json([
            'message' => 'Scan results processed successfully.',
            'host_id' => $host->id,
            'projects' => $results,
        ]);
    }
}
