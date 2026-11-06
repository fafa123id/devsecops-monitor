<?php

namespace App\Console\Commands;

use App\Models\DockerHost;
use Illuminate\Console\Command;
use Illuminate\Support\Str;

class RegisterDockerHost extends Command
{
    protected $signature = 'agent:register-host
        {name : Nama Docker Host}
        {--hostname= : Hostname VPS}
        {--ip= : IP Address VPS}';

    protected $description = 'Register a Docker Host and generate its agent token';

    public function handle(): int
    {
        $host = DockerHost::create([
            'name' => $this->argument('name'),
            'hostname' => $this->option('hostname'),
            'ip_address' => $this->option('ip'),
            'status' => 'pending',
        ]);

        $token = $host->createToken(
            'docker-agent',
            [
                'inventory:read',
                'inventory:write',
                'heartbeat:write',
            ]
        );

        $this->newLine();
        $this->info('Docker Host registered successfully.');
        $this->line('Host ID: '.$host->id);
        $this->line('Host Name: '.$host->name);
        $this->newLine();

        $this->warn('AGENT TOKEN - COPY IT NOW. IT WILL NOT BE SHOWN AGAIN.');
        $this->line($token->plainTextToken);

        $this->newLine();

        return self::SUCCESS;
    }
}
