<?php

namespace Tests\Feature;

use App\Models\DockerHost;
use App\Models\Project;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\DB;
use Tests\TestCase;

class AgentInventoryTest extends TestCase
{
    use RefreshDatabase;

    private function createHost(): DockerHost
    {
        return DockerHost::create([
            'name' => 'Test Host',
            'hostname' => 'test-host',
            'status' => 'online',
        ]);
    }

    private function container(
        string $name,
        string $service,
        string $state = 'running',
    ): array {
        return [
            'container_id' => hash('sha256', $name),
            'container_name' => $name,
            'service_name' => $service,
            'image_reference' => $name . ':latest',
            'image_digest' => null,
            'container_state' => $state,
        ];
    }

    private function projectPayload(
        string $name,
        array $containers,
    ): array {
        return [
            'docker_project_name' => $name,
            'containers' => $containers,
        ];
    }

    private function submitInventory(
        DockerHost $host,
        array $projects,
        bool $complete = true,
    ) {
        $token = $host->createToken(
            'test-agent',
            ['inventory:write']
        );

        return $this
            ->withToken($token->plainTextToken)
            ->postJson('/api/agent/inventory', [
                'snapshot_complete' => $complete,
                'projects' => $projects,
            ]);
    }

    private function submitEmptySnapshot(DockerHost $host): void
    {
        $this->submitInventory($host, [])
            ->assertOk();
    }

    public function test_new_project_with_running_containers_becomes_active(): void
    {
        $host = $this->createHost();

        $response = $this->submitInventory($host, [
            $this->projectPayload('deployment', [
                $this->container(
                    'deployment-api',
                    'api',
                    'running'
                ),
                $this->container(
                    'deployment-postgres',
                    'postgres',
                    'running'
                ),
            ]),
        ]);

        $response
            ->assertOk()
            ->assertJsonPath('projects_received', 1)
            ->assertJsonPath('containers_received', 2);

        $project = Project::where(
            'docker_host_id',
            $host->id
        )->where(
            'docker_project_name',
            'deployment'
        )->firstOrFail();

        $this->assertSame('active', $project->status);
        $this->assertSame(0, $project->missing_snapshots);
        $this->assertNotNull($project->last_seen_at);

        $this->assertDatabaseHas('project_containers', [
            'project_id' => $project->id,
            'container_name' => 'deployment-api',
            'container_state' => 'running',
        ]);
        $this->assertDatabaseHas('projects', [
            'docker_host_id' => $host->id,
            'docker_project_name' => 'deployment',
            'scan_source' => true,
            'weekly_scan_enabled' => true,
        ]);
    }

    public function test_project_with_all_stopped_containers_becomes_stopped(): void
    {
        $host = $this->createHost();

        $this->submitInventory($host, [
            $this->projectPayload('deployment', [
                $this->container(
                    'deployment-api',
                    'api',
                    'exited'
                ),
                $this->container(
                    'deployment-postgres',
                    'postgres',
                    'exited'
                ),
            ]),
        ])->assertOk();

        $this->assertDatabaseHas('projects', [
            'docker_host_id' => $host->id,
            'docker_project_name' => 'deployment',
            'status' => 'stopped',
        ]);
    }

    public function test_project_with_partially_running_containers_becomes_degraded(): void
    {
        $host = $this->createHost();

        $this->submitInventory($host, [
            $this->projectPayload('deployment', [
                $this->container(
                    'deployment-api',
                    'api',
                    'running'
                ),
                $this->container(
                    'deployment-postgres',
                    'postgres',
                    'exited'
                ),
            ]),
        ])->assertOk();

        $this->assertDatabaseHas('projects', [
            'docker_host_id' => $host->id,
            'docker_project_name' => 'deployment',
            'status' => 'degraded',
        ]);
    }

    public function test_missing_project_requires_three_complete_snapshots(): void
    {
        $host = $this->createHost();

        $this->submitInventory($host, [
            $this->projectPayload('deployment', [
                $this->container(
                    'deployment-api',
                    'api',
                    'running'
                ),
            ]),
        ])->assertOk();

        $project = Project::where(
            'docker_host_id',
            $host->id
        )->where(
            'docker_project_name',
            'deployment'
        )->firstOrFail();

        // Snapshot 1: project not detected.
        $this->submitEmptySnapshot($host);

        $project->refresh();

        $this->assertSame('active', $project->status);
        $this->assertSame(1, $project->missing_snapshots);
        $this->assertNotNull($project->missing_since_at);

        // Container can be marked missing immediately.
        $this->assertDatabaseHas('project_containers', [
            'project_id' => $project->id,
            'container_state' => 'missing',
        ]);

        // Snapshot 2.
        $this->submitEmptySnapshot($host);

        $project->refresh();

        $this->assertSame('active', $project->status);
        $this->assertSame(2, $project->missing_snapshots);

        // Snapshot 3: project becomes missing.
        $response = $this->submitInventory($host, []);

        $response
            ->assertOk()
            ->assertJsonPath('projects_marked_missing', 1);

        $project->refresh();

        $this->assertSame('missing', $project->status);
        $this->assertSame(3, $project->missing_snapshots);
    }

    public function test_missing_project_is_reactivated_without_losing_history(): void
    {
        $host = $this->createHost();

        $this->submitInventory($host, [
            $this->projectPayload('deployment', [
                $this->container(
                    'deployment-api',
                    'api',
                    'running'
                ),
            ]),
        ])->assertOk();

        $project = Project::where(
            'docker_host_id',
            $host->id
        )->where(
            'docker_project_name',
            'deployment'
        )->firstOrFail();

        // Create an existing scan history record.
        $scanRunId = DB::table('scan_runs')->insertGetId([
            'project_id' => $project->id,
            'trigger_type' => 'manual',
            'status' => 'completed',
            'scanner_version' => 'test',
            'created_at' => now(),
            'updated_at' => now(),
        ]);

        // Remove project from three complete snapshots.
        for ($i = 0; $i < 3; $i++) {
            $this->submitEmptySnapshot($host);
        }

        $project->refresh();

        $this->assertSame('missing', $project->status);

        // Project appears again with its original container.
        $response = $this->submitInventory($host, [
            $this->projectPayload('deployment', [
                $this->container(
                    'deployment-api',
                    'api',
                    'running'
                ),
            ]),
        ]);

        $response
            ->assertOk()
            ->assertJsonPath('projects_reactivated', 1);

        $project->refresh();

        $this->assertSame('active', $project->status);
        $this->assertSame(0, $project->missing_snapshots);
        $this->assertNull($project->missing_since_at);

        // Original project identity must remain unchanged.
        $this->assertSame(
            $project->id,
            Project::where(
                'docker_host_id',
                $host->id
            )->where(
                'docker_project_name',
                'deployment'
            )->value('id')
        );

        // Previous scan history must not be deleted.
        $this->assertDatabaseHas('scan_runs', [
            'id' => $scanRunId,
            'project_id' => $project->id,
        ]);
    }

    public function test_archived_project_stays_archived_when_detected(): void
    {
        $host = $this->createHost();

        $this->submitInventory($host, [
            $this->projectPayload('deployment', [
                $this->container(
                    'deployment-api',
                    'api',
                    'running'
                ),
            ]),
        ])->assertOk();

        $project = Project::where(
            'docker_host_id',
            $host->id
        )->where(
            'docker_project_name',
            'deployment'
        )->firstOrFail();

        $project->update([
            'status' => 'archived',
        ]);

        $this->submitInventory($host, [
            $this->projectPayload('deployment', [
                $this->container(
                    'deployment-api',
                    'api',
                    'running'
                ),
            ]),
        ])->assertOk();

        $project->refresh();

        $this->assertSame('archived', $project->status);
        $this->assertNotNull($project->last_seen_at);
    }

    public function test_incomplete_inventory_snapshot_is_rejected(): void
    {
        $host = $this->createHost();

        $this->submitInventory($host, [
            $this->projectPayload('deployment', [
                $this->container(
                    'deployment-api',
                    'api',
                    'running'
                ),
            ]),
        ])->assertOk();

        $response = $this->submitInventory(
            $host,
            [],
            false
        );

        $response->assertUnprocessable();

        $this->assertDatabaseHas('projects', [
            'docker_host_id' => $host->id,
            'docker_project_name' => 'deployment',
            'status' => 'active',
            'missing_snapshots' => 0,
        ]);
    }
}
