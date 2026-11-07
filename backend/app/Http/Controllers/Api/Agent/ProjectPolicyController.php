<?php

namespace App\Http\Controllers\Api\Agent;

use App\Http\Controllers\Controller;
use App\Models\Project;
use Illuminate\Http\Request;

class ProjectPolicyController extends Controller
{
    public function __invoke(Request $request)
    {
        $host = $request->user();

        $policies = Project::query()
            ->where('docker_host_id', $host->id)
            ->orderBy('docker_project_name')
            ->get([
                'docker_project_name',
                'status',
                'scan_source',
                'weekly_scan_enabled',
            ])
            ->map(function (Project $project) {
                return [
                    'docker_project_name' => $project->docker_project_name,
                    'status' => $project->status,
                    'scan_source' => (bool) $project->scan_source,
                    'weekly_scan_enabled' => (bool) $project->weekly_scan_enabled,
                ];
            })
            ->values();

        return response()->json([
            'host_id' => $host->id,
            'policies' => $policies,
            'count' => $policies->count(),
        ]);
    }
}