
<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('docker_hosts', function (Blueprint $table) {
            $table->dropUnique(['agent_token_id']);

            $table->dropColumn([
                'agent_token_id',
                'agent_token_hash',
            ]);
        });
    }

    public function down(): void
    {
        Schema::table('docker_hosts', function (Blueprint $table) {
            $table->string('agent_token_id', 32)->nullable()->unique();
            $table->string('agent_token_hash');
        });
    }
};
