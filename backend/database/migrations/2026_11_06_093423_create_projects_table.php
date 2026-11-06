
<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('projects', function (Blueprint $table) {
            $table->id();

            $table->foreignId('docker_host_id')
                ->constrained('docker_hosts')
                ->cascadeOnDelete();

            $table->string('name');
            $table->string('docker_project_name');

            // Referensi source code di host, bukan path bebas untuk eksekusi
            $table->string('source_path_reference')->nullable();

            $table->boolean('scan_source')->default(false);
            $table->boolean('weekly_scan_enabled')->default(true);

            $table->string('status')->default('active');

            $table->timestamp('last_scanned_at')->nullable();

            $table->timestamps();

            $table->unique(
                ['docker_host_id', 'docker_project_name'],
                'projects_host_docker_name_unique'
            );

            $table->index('status');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('projects');
    }
};
