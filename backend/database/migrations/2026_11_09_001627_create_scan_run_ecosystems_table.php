
<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('scan_run_ecosystems', function (Blueprint $table) {
            $table->id();

            $table->foreignId('scan_run_id')
                ->constrained('scan_runs')
                ->cascadeOnDelete();

            $table->string('ecosystem', 50);
            $table->string('manifest_file');
            $table->string('status', 20);
            $table->text('error_message')->nullable();

            $table->timestamps();

            $table->unique(
                ['scan_run_id', 'ecosystem', 'manifest_file'],
                'scan_run_ecosystems_unique'
            );

            $table->index(
                ['ecosystem', 'status'],
                'scan_run_ecosystems_status_index'
            );
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('scan_run_ecosystems');
    }
};
