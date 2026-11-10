<?php

namespace App\Console\Commands;

use App\Models\DockerHost;
use App\Services\NotificationService;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\DB;

class MarkOfflineDockerHosts extends Command
{
    protected $signature = 'hosts:mark-offline';

    protected $description = 'Mark Docker hosts as offline when heartbeat expires';

    public function handle(NotificationService $notificationService): int
    {
        $threshold = now()->subMinutes(2);

        $hosts = DockerHost::query()
            ->where('status', 'online')
            ->whereNotNull('last_heartbeat_at')
            ->where('last_heartbeat_at', '<', $threshold)
            ->get();

        $updated = 0;

        foreach ($hosts as $host) {
            $markedOffline = DB::transaction(function () use (
                $host,
                $threshold,
                $notificationService
            ) {
                $changed = DockerHost::query()
                    ->whereKey($host->id)
                    ->where('status', 'online')
                    ->whereNotNull('last_heartbeat_at')
                    ->where('last_heartbeat_at', '<', $threshold)
                    ->update([
                        'status' => 'offline',
                        'updated_at' => now(),
                    ]);

                if (!$changed) {
                    return false;
                }

                $notificationService->notifyAgentOffline($host);

                return true;
            });

            if ($markedOffline) {
                $updated++;
            }
        }

        $this->info("Marked {$updated} Docker host(s) as offline.");

        return self::SUCCESS;
    }
}