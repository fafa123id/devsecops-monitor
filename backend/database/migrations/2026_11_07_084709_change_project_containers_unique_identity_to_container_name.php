<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        // Find duplicate logical containers within each project.
        $duplicates = DB::table('project_containers')
            ->select('project_id', 'container_name')
            ->groupBy('project_id', 'container_name')
            ->havingRaw('COUNT(*) > 1')
            ->get();

        foreach ($duplicates as $duplicate) {
            // Keep the newest record. When timestamps are equal,
            // prefer a running container, then the highest ID.
            $rows = DB::table('project_containers')
                ->where('project_id', $duplicate->project_id)
                ->where('container_name', $duplicate->container_name)
                ->orderByRaw(
                    'CASE WHEN last_seen_at IS NULL THEN 1 ELSE 0 END ASC'
                )
                ->orderByDesc('last_seen_at')
                ->orderByRaw("
                    CASE
                        WHEN container_state = 'running' THEN 0
                        WHEN container_state IN ('paused', 'restarting') THEN 1
                        ELSE 2
                    END ASC
                ")
                ->orderByDesc('id')
                ->get(['id']);

            $deleteIds = $rows
                ->skip(1)
                ->pluck('id')
                ->all();

            if ($deleteIds !== []) {
                DB::table('project_containers')
                    ->whereIn('id', $deleteIds)
                    ->delete();
            }
        }

        // A recreated Docker container updates the same logical record.
        Schema::table('project_containers', function (Blueprint $table) {
            $table->unique(
                ['project_id', 'container_name'],
                'project_containers_project_name_unique'
            );
        });
    }

    public function down(): void
    {
        Schema::table('project_containers', function (Blueprint $table) {
            $table->dropUnique(
                'project_containers_project_name_unique'
            );
        });
    }
};