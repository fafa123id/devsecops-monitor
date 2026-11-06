
<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('repositories', function (Blueprint $table) {
            $table->id();

            $table->foreignId('project_id')
                ->constrained('projects')
                ->cascadeOnDelete();

            // GitHub untuk provider awal
            $table->string('provider')->default('github');
            $table->string('external_repository_id')->nullable();

            $table->string('owner');
            $table->string('name');
            $table->string('full_name');

            $table->text('html_url')->nullable();
            $table->text('clone_url')->nullable();

            $table->string('default_branch')->nullable();
            $table->boolean('is_private')->default(false);

            // ID instalasi GitHub App, bukan access token
            $table->string('installation_id')->nullable();

            $table->timestamp('last_synced_at')->nullable();

            $table->timestamps();

            $table->unique(
                ['provider', 'external_repository_id'],
                'repositories_provider_external_id_unique'
            );

            $table->index('project_id');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('repositories');
    }
};
