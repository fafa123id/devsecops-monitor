<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Models\ScanRun;
use Illuminate\Http\JsonResponse;

class DashboardController extends Controller
{
    public function vulnerabilityTrend(): JsonResponse
    {
        $scans = ScanRun::query()
            ->whereIn('status', ['completed', 'partial'])
            ->whereNotNull('completed_at')
            ->with('project:id,name')
            ->withCount([
                'findings as vulnerability_findings_count' => function ($query) {
                    $query->where('finding_type', 'vulnerability');
                },
            ])
            ->orderByDesc('completed_at')
            ->limit(10)
            ->get()
            ->reverse()
            ->values()
            ->map(fn ($scan) => [
                'id' => $scan->id,
                'date' => $scan->completed_at,
                'project' => $scan->project?->name,
                'findings_count' => $scan->vulnerability_findings_count,
                'status' => $scan->status,
            ]);

        return response()->json([
            'data' => $scans,
        ]);
    }
}