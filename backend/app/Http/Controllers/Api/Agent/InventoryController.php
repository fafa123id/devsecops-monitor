<?php

namespace App\Http\Controllers\Api\Agent;

use App\Http\Controllers\Controller;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\DB;
use App\Models\Project;
use App\Models\ProjectContainer;

class InventoryController extends Controller
{
    public function __invoke(Request $request)
    {
        $validated = $request->validate([
            'projects' => ['required', 'array'],
            'projects.*.docker_project_name' => ['required', 'string', 'max:255'],
            'projects.*.containers' => ['required', 'array'],

            'projects.*.containers.*.container_id' => [
                'required', 'string', 'max:64',
            ],
            'projects.*.containers.*.container_name' => [
                'required', 'string', 'max:255',
            ],
            'projects.*.containers.*.service_name' => [
                'nullable', 'string', 'max:255',
            ],
            'projects.*.containers.*.image_reference' => [
                'nullable', 'string', 'max:500',
            ],
            'projects.*.containers.*.image_digest' => [
                'nullable', 'string', 'max:255',
            ],
            'projects.*.containers.*.container_state' => [
                'required', 'string', 'max:50',
            ],
        ]);

        $host = $request->user();

        $result = DB::transaction(function () use ($validated, $host) {
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

            return compact('projectCount', 'containerCount');
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
        ]);
    }
}
