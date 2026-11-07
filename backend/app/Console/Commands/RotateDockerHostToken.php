<?php

namespace App\Console\Commands;

use App\Models\DockerHost;
use Illuminate\Console\Command;

class RotateDockerHostToken extends Command
{
    protected $signature = 'agent:rotate-token
        {host_id : ID Docker Host}';

    protected $description = 'Rotate an existing Docker Host agent token';

    public function handle(): int
    {
        $host = DockerHost::find($this->argument('host_id'));

        if (! $host) {
            $this->error('Docker Host not found.');

            return self::FAILURE;
        }

        if (! $this->confirm(
            "Rotate token for {$host->name} ({$host->hostname})?"
        )) {
            return self::SUCCESS;
        }

        $token = $host->createToken(
            'docker-agent',
            [
                'inventory:read',
                'inventory:write',
                'heartbeat:write',
                'scan:write',
            ]
        );

        $this->newLine();
        $this->info('New agent token generated.');
        $this->line('Host ID: '.$host->id);
        $this->warn('Copy this token now. It will not be shown again.');
        $this->line($token->plainTextToken);

        return self::SUCCESS;
    }
}