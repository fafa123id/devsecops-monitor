
<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('scan_findings', function (Blueprint $table) {
            $table->id();

            $table->foreignId('scan_run_id')
                ->constrained('scan_runs')
                ->cascadeOnDelete();

            $table->foreignId('project_id')
                ->constrained('projects')
                ->cascadeOnDelete();

            // source_dependency, container_image, os_package, secret
            $table->string('target_type');
            $table->string('target_identifier')->nullable();

            // Informasi vulnerability
            $table->string('vulnerability_id');
            $table->string('package_name')->nullable();
            $table->string('installed_version')->nullable();
            $table->string('fixed_version')->nullable();

            $table->string('severity')->default('unknown');
            $table->string('title')->nullable();
            $table->text('description')->nullable();
            $table->text('primary_url')->nullable();

            // new, existing, resolved
            $table->string('status')->default('new');

            // Identitas unik temuan lintas scan
            $table->string('finding_fingerprint', 64);

            $table->timestamps();

            $table->unique(
                ['scan_run_id', 'finding_fingerprint'],
                'scan_findings_run_fingerprint_unique'
            );

            $table->index(['project_id', 'status']);
            $table->index(['severity', 'status']);
            $table->index('vulnerability_id');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('scan_findings');
    }
};
