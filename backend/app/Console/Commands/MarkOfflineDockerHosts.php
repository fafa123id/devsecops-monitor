<?php

namespace App\Console\Commands;

use App\Models\DockerHost;
use Illuminate\Console\Command;

class MarkOfflineDockerHosts extends Command
{
    protected $signature = 'hosts:mark-offline';

    protected $description = 'Mark Docker hosts as offline when heartbeat expires';

    public function handle(): int
    {
        $updated = DockerHost::query()
            ->where('status', 'online')
            ->whereNotNull('last_heartbeat_at')
            ->where('last_heartbeat_at', '<', now()->subMinutes(2))
            ->update([
                'status' => 'offline',
            ]);

        $this->info("Marked {$updated} Docker host(s) as offline.");

        return self::SUCCESS;
    }
}
