
<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('project_containers', function (Blueprint $table) {
            $table->id();

            $table->foreignId('project_id')
                ->constrained('projects')
                ->cascadeOnDelete();

            // Identitas container Docker
            $table->string('container_id', 64);
            $table->string('container_name');
            $table->string('service_name')->nullable();

            // Informasi image
            $table->string('image_reference')->nullable();
            $table->string('image_digest')->nullable();

            // Status container: running, exited, paused, dll.
            $table->string('container_state')->default('unknown');

            $table->timestamps();

            // Satu container hanya tercatat sekali dalam satu project
            $table->unique(
                ['project_id', 'container_id'],
                'project_containers_project_container_unique'
            );

            $table->index('container_state');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('project_containers');
    }
};
