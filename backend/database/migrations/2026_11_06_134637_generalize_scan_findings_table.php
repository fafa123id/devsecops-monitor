
<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('scan_findings', function (Blueprint $table) {
            $table->string('finding_type')
                ->default('vulnerability')
                ->after('target_identifier');

            $table->string('finding_id')
                ->nullable()
                ->after('finding_type');

            $table->json('metadata')
                ->nullable()
                ->after('primary_url');
        });

        // Preserve identifiers from existing vulnerability findings.
        DB::table('scan_findings')
            ->whereNull('finding_id')
            ->update([
                'finding_id' => DB::raw('vulnerability_id'),
            ]);

        Schema::table('scan_findings', function (Blueprint $table) {
            $table->string('vulnerability_id')
                ->nullable()
                ->change();

            $table->index(
                ['finding_type', 'severity'],
                'scan_findings_type_severity_index'
            );

            $table->index(
                'finding_id',
                'scan_findings_finding_id_index'
            );
        });
    }

    public function down(): void
    {
        // Refuse rollback if non-vulnerability findings exist.
        if (DB::table('scan_findings')
            ->where('finding_type', '!=', 'vulnerability')
            ->exists()) {
            throw new RuntimeException(
                'Cannot rollback: non-vulnerability findings exist.'
            );
        }

        Schema::table('scan_findings', function (Blueprint $table) {
            $table->dropIndex('scan_findings_type_severity_index');
            $table->dropIndex('scan_findings_finding_id_index');
            $table->dropColumn([
                'finding_type',
                'finding_id',
                'metadata',
            ]);
        });

        // Existing vulnerability records already have this value.
        Schema::table('scan_findings', function (Blueprint $table) {
            $table->string('vulnerability_id')
                ->nullable(false)
                ->change();
        });
    }
};
