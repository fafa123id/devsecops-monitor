
<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('repository_branches', function (Blueprint $table) {
            $table->id();

            $table->foreignId('repository_id')
                ->constrained('repositories')
                ->cascadeOnDelete();

            $table->string('name');
            $table->string('last_commit_sha', 64)->nullable();

            $table->boolean('is_default')->default(false);
            $table->boolean('is_deployed')->default(false);

            $table->timestamp('last_synced_at')->nullable();

            $table->timestamps();

            $table->unique(
                ['repository_id', 'name'],
                'repository_branches_repository_name_unique'
            );
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('repository_branches');
    }
};
