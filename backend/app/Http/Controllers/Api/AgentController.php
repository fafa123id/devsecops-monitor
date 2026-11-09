<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Models\DockerHost;
use Illuminate\Http\JsonResponse;

class AgentController extends Controller
{
    public function index(): JsonResponse
    {
        $hosts = DockerHost::query()
            ->withCount('projects')
            ->with([
                'projects' => function ($query) {
                    $query->withCount([
                        'containers as containers_total' => function ($containerQuery) {
                            $containerQuery->where(
                                'container_state',
                                '!=',
                                'missing'
                            );
                        },
                    ]);
                },
            ])
            ->orderBy('name')
            ->get();

        $summary = [
            'total' => $hosts->count(),
            'online' => $hosts->where('status', 'online')->count(),
            'offline' => $hosts->where('status', 'offline')->count(),
            'pending' => $hosts->where('status', 'pending')->count(),
        ];

        $data = $hosts->map(function (DockerHost $host) {
            $containerCount = $host->projects->sum(
                'containers_total'
            );

            return [
                'id' => $host->id,
                'name' => $host->name,
                'hostname' => $host->hostname,
                'ip_address' => $host->ip_address,

                'status' => strtolower($host->status),

                'agent_version' => $host->agent_version,
                'last_heartbeat_at' => $host->last_heartbeat_at,
                'registered_at' => $host->created_at,

                'projects_count' => (int) $host->projects_count,
                'containers_count' => (int) $containerCount,
            ];
        })->values();

        return response()->json([
            'message' => 'Agents retrieved successfully.',
            'data' => $data,
            'summary' => $summary,
            'meta' => [
                'total' => $data->count(),
            ],
        ]);
    }
}
