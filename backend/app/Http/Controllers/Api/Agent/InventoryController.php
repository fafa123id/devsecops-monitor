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

        // Reject duplicate container IDs in one snapshot.
        $containerIds = [];

        foreach ($validated['projects'] as $projectData) {
            foreach ($projectData['containers'] as $containerData) {
                $containerId = $containerData['container_id'];

                if (isset($containerIds[$containerId])) {
                    throw ValidationException::withMessages([
                        'projects' => [
                            "Duplicate container ID detected: {$containerId}",
                        ],
                    ]);
                }

                $containerIds[$containerId] = true;
            }
        }

        $receivedContainerIds = array_keys($containerIds);

        $result = DB::transaction(function () use (
            $validated,
            $host,
            $receivedContainerIds
        ) {
            $projectCount = 0;
            $containerCount = 0;

            foreach ($validated['projects'] as $item) {
                $project = Project::updateOrCreate(
                    [
                        'docker_host_id' => $host->id,
                        'docker_project_name' => $item['docker_project_name'],
                    ],
                    [
                        'name' => $item['docker_project_name'],
                        'status' => 'active',
                    ]
                );

                $projectCount++;

                foreach ($item['containers'] as $containerData) {
                    ProjectContainer::updateOrCreate(
                        [
                            'project_id' => $project->id,
                            'container_id' => $containerData['container_id'],
                        ],
                        [
                            'container_name' => $containerData['container_name'],
                            'service_name' => $containerData['service_name'] ?? null,
                            'image_reference' => $containerData['image_reference'] ?? null,
                            'image_digest' => $containerData['image_digest'] ?? null,
                            'container_state' => $containerData['container_state'],
                        ]
                    );

                    $containerCount++;
                }
            }

            // Reconcile only after a complete snapshot has been validated.
            $staleContainers = ProjectContainer::whereHas(
                'project',
                function ($query) use ($host) {
                    $query->where('docker_host_id', $host->id);
                }
            );

            if ($receivedContainerIds === []) {
                // An empty snapshot means no Compose containers remain.
                $staleCount = $staleContainers->update([
                    'container_state' => 'missing',
                ]);
            } else {
                $staleCount = $staleContainers
                    ->whereNotIn('container_id', $receivedContainerIds)
                    ->update([
                        'container_state' => 'missing',
                    ]);
            }

            return compact(
                'projectCount',
                'containerCount',
                'staleCount'
            );
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
            'containers_marked_missing' => $result['staleCount'],
        ]);
    }
}
