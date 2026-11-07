
<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('projects', function (Blueprint $table) {
            $table->timestamp('last_seen_at')->nullable();
            $table->timestamp('missing_since_at')->nullable();
            $table->unsignedSmallInteger('missing_snapshots')->default(0);

            $table->index(
                ['docker_host_id', 'status'],
                'projects_host_status_index'
            );
        });

        Schema::table('project_containers', function (Blueprint $table) {
            $table->timestamp('last_seen_at')->nullable();
        });
    }

    public function down(): void
    {
        Schema::table('project_containers', function (Blueprint $table) {
            $table->dropColumn('last_seen_at');
        });

        Schema::table('projects', function (Blueprint $table) {
            $table->dropIndex('projects_host_status_index');

            $table->dropColumn([
                'last_seen_at',
                'missing_since_at',
                'missing_snapshots',
            ]);
        });
    }
};
