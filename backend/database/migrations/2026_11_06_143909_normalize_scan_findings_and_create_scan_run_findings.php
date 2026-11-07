
<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        /*
         * 1. Create pivot table first.
         */
        Schema::create('scan_run_findings', function (Blueprint $table) {
            $table->id();

            $table->foreignId('scan_run_id')
                ->constrained('scan_runs')
                ->cascadeOnDelete();

            $table->foreignId('scan_finding_id')
                ->constrained('scan_findings')
                ->cascadeOnDelete();

            $table->timestamps();

            $table->unique(
                ['scan_run_id', 'scan_finding_id'],
                'scan_run_findings_unique'
            );
        });

        /*
         * 2. Add canonical finding lifecycle columns.
         */
        Schema::table('scan_findings', function (Blueprint $table) {
            $table->foreignId('last_scan_run_id')
                ->nullable()
                ->after('scan_run_id')
                ->constrained('scan_runs')
                ->nullOnDelete();

            $table->timestamp('first_seen_at')
                ->nullable()
                ->after('finding_fingerprint');

            $table->timestamp('last_seen_at')
                ->nullable()
                ->after('first_seen_at');
        });

        /*
         * 3. Create pivot relationships from existing records.
         */
        DB::table('scan_findings')
            ->select('id', 'scan_run_id')
            ->orderBy('id')
            ->chunk(1000, function ($findings) {
                $now = now();

                $rows = $findings->map(fn($finding) => [
                    'scan_run_id' => $finding->scan_run_id,
                    'scan_finding_id' => $finding->id,
                    'created_at' => $now,
                    'updated_at' => $now,
                ])->all();

                DB::table('scan_run_findings')->insert($rows);
            });

        /*
         * 4. Populate lifecycle timestamps and latest scan.
         */
        DB::statement("
            UPDATE scan_findings sf
            SET
                first_seen_at = sr.started_at,
                last_seen_at = sr.completed_at,
                last_scan_run_id = sf.scan_run_id
            FROM scan_runs sr
            WHERE sr.id = sf.scan_run_id
        ");

        /*
         * 5. Keep one canonical row for each project + fingerprint.
         * Prefer the newest scan record.
         */
        $duplicates = DB::table('scan_findings')
            ->select('project_id', 'finding_fingerprint')
            ->groupBy('project_id', 'finding_fingerprint')
            ->havingRaw('COUNT(*) > 1')
            ->get();

        foreach ($duplicates as $duplicate) {
            $records = DB::table('scan_findings')
                ->where('project_id', $duplicate->project_id)
                ->where('finding_fingerprint', $duplicate->finding_fingerprint)
                ->join('scan_runs', 'scan_runs.id', '=', 'scan_findings.scan_run_id')
                ->orderByRaw('scan_runs.completed_at DESC NULLS LAST')
                ->orderByDesc('scan_findings.id')
                ->select('scan_findings.*')
                ->get();

            $canonical = $records->first();
            $duplicateIds = $records->pluck('id')->all();

            $allScanIds = DB::table('scan_run_findings')
                ->whereIn('scan_finding_id', $duplicateIds)
                ->pluck('scan_run_id')
                ->unique();

            $now = now();

            foreach ($allScanIds as $scanRunId) {
                DB::table('scan_run_findings')->updateOrInsert(
                    [
                        'scan_run_id' => $scanRunId,
                        'scan_finding_id' => $canonical->id,
                    ],
                    [
                        'created_at' => $now,
                        'updated_at' => $now,
                    ]
                );
            }

            DB::table('scan_run_findings')
                ->whereIn('scan_finding_id', $duplicateIds)
                ->where('scan_finding_id', '!=', $canonical->id)
                ->delete();

            DB::table('scan_findings')
                ->whereIn('id', $duplicateIds)
                ->where('id', '!=', $canonical->id)
                ->delete();

            $firstSeen = $records->min('first_seen_at');
            $lastSeen = $records->max('last_seen_at');

            DB::table('scan_findings')
                ->where('id', $canonical->id)
                ->update([
                    'first_seen_at' => $firstSeen,
                    'last_seen_at' => $lastSeen,
                ]);
        }

        /*
         * 6. Replace the old unique constraint.
         */
        Schema::table('scan_findings', function (Blueprint $table) {
            $table->dropUnique('scan_findings_run_fingerprint_unique');

            $table->unique(
                ['project_id', 'finding_fingerprint'],
                'scan_findings_project_fingerprint_unique'
            );
        });

        /*
         * 7. Remove the old direct scan relationship.
         */
        Schema::table('scan_findings', function (Blueprint $table) {
            $table->dropConstrainedForeignId('scan_run_id');
        });
    }

    public function down(): void
    {
        throw new RuntimeException(
            'This migration is intentionally irreversible because it consolidates historical findings.'
        );
    }
};
