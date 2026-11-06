
<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('docker_hosts', function (Blueprint $table) {
            $table->id();

            // Identitas host
            $table->string('name');
            $table->string('hostname')->nullable();
            $table->string('ip_address')->nullable();

            // Autentikasi agent
            $table->string('agent_token_hash')->unique();
            $table->string('agent_version')->nullable();

            // Status koneksi agent
            $table->timestamp('last_heartbeat_at')->nullable();
            $table->string('status')->default('pending');

            $table->timestamps();

            $table->index('status');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('docker_hosts');
    }
};
