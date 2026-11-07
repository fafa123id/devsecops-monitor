<?php

namespace App\Http\Controllers\Api\Agent;

use App\Http\Controllers\Controller;
use App\Models\Project;
use App\Models\ProjectContainer;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\DB;
use Illuminate\Validation\ValidationException;

class InventoryController extends Controller
{
    private const MISSING_SNAPSHOT_THRESHOLD = 3;

    public function __invoke(Request $request)
    {
        $validated = $request->validate([
            'snapshot_complete' => ['required', 'accepted'],
            'projects' => ['present', 'array'],

            'projects.*.docker_project_name' => [
                'required',
                'string',
                'max:255',
            ],
            'projects.*.containers' => [
                'required',
                'array',
            ],
            'projects.*.containers.*.container_id' => [
                'required',
                'string',
                'max:64',
            ],
            'projects.*.containers.*.container_name' => [
                'required',
                'string',
                'max:255',
            ],
            'projects.*.containers.*.service_name' => [
                'nullable',
                'string',
                'max:255',
            ],
            'projects.*.containers.*.image_reference' => [
                'nullable',
                'string',
                'max:500',
            ],
            'projects.*.containers.*.image_digest' => [
                'nullable',
                'string',
                'max:255',
            ],
            'projects.*.containers.*.container_state' => [
                'required',
                'string',
                'max:50',
            ],
        ]);

        $host = $request->user();

        $containerIds = [];
        $projectNames = [];

        foreach ($validated['projects'] as $projectData) {
            $projectName = $projectData['docker_project_name'];

            if (isset($projectNames[$projectName])) {
                throw ValidationException::withMessages([
                    'projects' => [
                        "Duplicate Docker project detected: {$projectName}",
                    ],
                ]);
            }

            $projectNames[$projectName] = true;
            $projectContainerNames = [];

            foreach ($projectData['containers'] as $containerData) {
                $containerId = $containerData['container_id'];
                $containerName = strtolower(
                    trim($containerData['container_name'])
                );

                if (isset($containerIds[$containerId])) {
                    throw ValidationException::withMessages([
                        'projects' => [
                            "Duplicate container ID detected: {$containerId}",
                        ],
                    ]);
                }

                if (isset($projectContainerNames[$containerName])) {
                    throw ValidationException::withMessages([
                        'projects' => [
                            "Duplicate container name detected in project {$projectName}: {$containerName}",
                        ],
                    ]);
                }

                $containerIds[$containerId] = true;
                $projectContainerNames[$containerName] = true;
            }
        }

        $receivedContainerIds = array_keys($containerIds);
        $receivedProjectNames = array_keys($projectNames);

        $result = DB::transaction(function () use (
            $validated,
            $host,
            $receivedContainerIds,
            $receivedProjectNames
        ) {
            $now = now();

            $projectCount = 0;
            $containerCount = 0;
            $reactivatedCount = 0;
            $pendingMissingCount = 0;
            $markedMissingCount = 0;

            foreach ($validated['projects'] as $item) {
                $project = Project::query()
                    ->where('docker_host_id', $host->id)
                    ->where(
                        'docker_project_name',
                        $item['docker_project_name']
                    )
                    ->lockForUpdate()
                    ->first();

                if (!$project) {
                    $project = new Project([
                        'docker_host_id' => $host->id,
                        'docker_project_name' => $item['docker_project_name'],
                        'name' => $item['docker_project_name'],
                        'status' => 'active',

                        // Automatically enable scanning for newly discovered projects.
                        'scan_source' => true,
                        'weekly_scan_enabled' => true,
                    ]);
                }
                $project->scan_source = true;
                $project->weekly_scan_enabled = true;
                $previousStatus = $project->status;
                $wasArchived = $previousStatus === 'archived';

                $project->name = $item['docker_project_name'];
                $project->last_seen_at = $now;
                $project->missing_since_at = null;
                $project->missing_snapshots = 0;
                // Archived projects stay archived until explicitly restored.
                if (!$wasArchived) {
                    $project->status = $this->determineProjectStatus(
                        $item['containers']
                    );
                }

                $project->save();
                foreach ($item['containers'] as $containerData) {
                    ProjectContainer::updateOrCreate(
                        [
                            'project_id' => $project->id,
                            'container_name' => $containerData['container_name'],
                        ],
                        [
                            'container_id' => $containerData['container_id'],
                            'service_name' => $containerData['service_name'] ?? null,
                            'image_reference' => $containerData['image_reference'] ?? null,
                            'image_digest' => $containerData['image_digest'] ?? null,
                            'container_state' => strtolower(
                                trim($containerData['container_state'])
                            ),
                            'last_seen_at' => $now,
                        ]
                    );

                    $containerCount++;
                }



                if (
                    $previousStatus === 'missing' &&
                    $project->status !== 'missing' &&
                    !$wasArchived
                ) {
                    $reactivatedCount++;
                }

                $projectCount++;
            }

            // Mark containers absent from this complete host snapshot.
            $staleContainers = ProjectContainer::query()
                ->whereHas('project', function ($query) use ($host) {
                    $query->where('docker_host_id', $host->id);
                })
                ->where('container_state', '!=', 'missing');
            if ($receivedContainerIds === []) {
                $containerMissingCount = $staleContainers->update([
                    'container_state' => 'missing',
                ]);
            } else {
                $containerMissingCount = $staleContainers
                    ->whereNotIn('container_id', $receivedContainerIds)
                    ->update([
                        'container_state' => 'missing',
                    ]);
            }
            // Recalculate project status after marking stale containers.
            $projectsToRecalculate = Project::query()
                ->where('docker_host_id', $host->id)
                ->whereIn('docker_project_name', $receivedProjectNames)
                ->where('status', '!=', 'archived')
                ->with('containers:id,project_id,container_state')
                ->get();

            foreach ($projectsToRecalculate as $project) {
                $states = $project->containers
                    ->pluck('container_state')
                    ->all();

                $project->status = $this->determineProjectStatusFromStoredContainers(
                    $states
                );

                $project->save();
            }
            // Reconcile projects absent from this complete snapshot.
            $missingProjects = Project::query()
                ->where('docker_host_id', $host->id)
                ->where('status', '!=', 'archived')
                ->when(
                    $receivedProjectNames !== [],
                    fn($query) => $query->whereNotIn(
                        'docker_project_name',
                        $receivedProjectNames
                    )
                )
                ->lockForUpdate()
                ->get();

            foreach ($missingProjects as $project) {
                if ($project->missing_since_at === null) {
                    $project->missing_since_at = $now;
                }

                $project->missing_snapshots++;

                if (
                    $project->missing_snapshots >=
                    self::MISSING_SNAPSHOT_THRESHOLD
                ) {
                    if ($project->status !== 'missing') {
                        $markedMissingCount++;
                    }

                    $project->status = 'missing';
                } else {
                    $pendingMissingCount++;
                }

                $project->save();
            }

            return [
                'projectCount' => $projectCount,
                'containerCount' => $containerCount,
                'containerMissingCount' => $containerMissingCount,
                'reactivatedCount' => $reactivatedCount,
                'pendingMissingCount' => $pendingMissingCount,
                'markedMissingCount' => $markedMissingCount,
            ];
        });

        $host->update([
            'last_heartbeat_at' => now(),
            'status' => 'online',
        ]);

        return response()->json([
            'message' => 'Inventory synchronized successfully.',
            'host_id' => $host->id,
            'projects_received' => $result['projectCount'],
            'containers_received' => $result['containerCount'],
            'containers_marked_missing' => $result['containerMissingCount'],
            'projects_pending_missing' => $result['pendingMissingCount'],
            'projects_marked_missing' => $result['markedMissingCount'],
            'projects_reactivated' => $result['reactivatedCount'],
        ]);
    }
    private function determineProjectStatusFromStoredContainers(
        array $states
    ): string {
        if ($states === []) {
            return 'stopped';
        }

        $total = count($states);
        $running = 0;
        $paused = 0;
        $missing = 0;

        foreach ($states as $state) {
            $state = strtolower(trim($state));

            if ($state === 'running') {
                $running++;
            }

            if ($state === 'paused' || $state === 'restarting') {
                $paused++;
            }

            if ($state === 'missing') {
                $missing++;
            }
        }

        // No active or paused containers means the project is stopped.
        if ($running === 0 && $paused === 0) {
            return 'stopped';
        }

        // Some containers are missing while others remain active.
        if ($missing > 0) {
            return 'degraded';
        }

        if ($running === $total) {
            return 'active';
        }

        return 'degraded';
    }
    private function determineProjectStatus(array $containers): string
    {
        if ($containers === []) {
            return 'stopped';
        }

        $total = count($containers);
        $running = 0;
        $paused = 0;

        foreach ($containers as $container) {
            $state = strtolower(
                trim($container['container_state'])
            );

            if ($state === 'running') {
                $running++;
            }

            if ($state === 'paused' || $state === 'restarting') {
                $paused++;
            }
        }

        if ($running === $total) {
            return 'active';
        }

        if ($running > 0 || $paused > 0) {
            return 'degraded';
        }

        return 'stopped';
    }
}
