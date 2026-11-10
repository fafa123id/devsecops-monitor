<?php

namespace Tests\Feature;

use App\Models\DockerHost;
use App\Models\Notification;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class MarkOfflineDockerHostsTest extends TestCase
{
    use RefreshDatabase;

    private function createHost(
        string $name,
        string $status,
        mixed $lastHeartbeatAt
    ): DockerHost {
        return DockerHost::create([
            'name' => $name,
            'hostname' => strtolower(str_replace(' ', '-', $name)),
            'status' => $status,
            'last_heartbeat_at' => $lastHeartbeatAt,
        ]);
    }

    public function test_stale_online_agent_is_marked_offline_and_notified(): void
    {
        $user = User::factory()->create();

        $host = $this->createHost(
            'Production Agent',
            'online',
            now()->subMinutes(3)
        );

        $this->artisan('hosts:mark-offline')
            ->assertSuccessful();

        $this->assertDatabaseHas('docker_hosts', [
            'id' => $host->id,
            'status' => 'offline',
        ]);

        $this->assertDatabaseHas('notifications', [
            'user_id' => $user->id,
            'type' => 'agent_offline',
            'title' => 'Docker agent went offline',
        ]);

        $this->assertDatabaseCount('notifications', 1);
    }

    public function test_repeated_command_does_not_create_duplicate_notifications(): void
    {
        User::factory()->create();

        $host = $this->createHost(
            'Production Agent',
            'online',
            now()->subMinutes(3)
        );

        $this->artisan('hosts:mark-offline')
            ->assertSuccessful();

        $this->artisan('hosts:mark-offline')
            ->assertSuccessful();

        $this->assertDatabaseHas('docker_hosts', [
            'id' => $host->id,
            'status' => 'offline',
        ]);

        $this->assertDatabaseCount('notifications', 1);
    }

    public function test_agent_with_recent_heartbeat_remains_online(): void
    {
        User::factory()->create();

        $host = $this->createHost(
            'Active Agent',
            'online',
            now()->subMinute()
        );

        $this->artisan('hosts:mark-offline')
            ->assertSuccessful();

        $this->assertDatabaseHas('docker_hosts', [
            'id' => $host->id,
            'status' => 'online',
        ]);

        $this->assertDatabaseCount('notifications', 0);
    }

    public function test_agent_without_heartbeat_is_not_marked_offline(): void
    {
        User::factory()->create();

        $host = $this->createHost(
            'New Agent',
            'online',
            null
        );

        $this->artisan('hosts:mark-offline')
            ->assertSuccessful();

        $this->assertDatabaseHas('docker_hosts', [
            'id' => $host->id,
            'status' => 'online',
        ]);

        $this->assertDatabaseCount('notifications', 0);
    }
}