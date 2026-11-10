<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('notifications', function (Blueprint $table) {
            $table->id();

            // Notification recipient
            $table->foreignId('user_id')
                ->constrained()
                ->cascadeOnDelete();

            // Related resources
            $table->foreignId('project_id')
                ->nullable()
                ->constrained()
                ->nullOnDelete();

            $table->foreignId('scan_run_id')
                ->nullable()
                ->constrained('scan_runs')
                ->nullOnDelete();

            $table->foreignId('scan_finding_id')
                ->nullable()
                ->constrained('scan_findings')
                ->nullOnDelete();

            // Notification information
            $table->string('type', 100);
            $table->string('severity', 20)->nullable();
            $table->string('title');
            $table->text('message');
            $table->json('data')->nullable();

            // Read status
            $table->timestamp('read_at')->nullable();

            $table->timestamps();

            // Indexes
            $table->index(['user_id', 'read_at']);
            $table->index(['user_id', 'created_at']);
            $table->index(['type', 'severity']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('notifications');
    }
};