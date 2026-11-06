
<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('scan_runs', function (Blueprint $table) {
            $table->id();

            $table->foreignId('project_id')
                ->constrained('projects')
                ->cascadeOnDelete();

            // manual, scheduled, image_updated, remediation
            $table->string('trigger_type')->default('manual');

            // queued, preparing, scanning, processing,
            // completed, partial, failed, cancelled
            $table->string('status')->default('queued');

            $table->timestamp('started_at')->nullable();
            $table->timestamp('completed_at')->nullable();

            $table->string('scanner_version')->nullable();
            $table->string('source_commit_sha')->nullable();

            $table->text('error_message')->nullable();

            $table->timestamps();

            $table->index(['project_id', 'status']);
            $table->index(['project_id', 'created_at']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('scan_runs');
    }
};
