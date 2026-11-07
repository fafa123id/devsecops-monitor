<?php

namespace Tests\Feature;

use App\Models\DockerHost;
use App\Models\Project;
use App\Models\ScanFinding;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class AgentScanTest extends TestCase
{
    use RefreshDatabase;

    private function createHost(string $name = 'Test Host'): DockerHost
    {
        return DockerHost::create([
            'name' => $name,
            'hostname' => strtolower(str_replace(' ', '-', $name)),
            'status' => 'online',
        ]);
    }

    private function createProject(DockerHost $host): Project
    {
        return Project::create([
            'docker_host_id' => $host->id,
            'name' => 'deployment',
            'docker_project_name' => 'deployment',
            'status' => 'active',
            'scan_source' => true,
            'weekly_scan_enabled' => true,
        ]);
    }

    private function scanPayload(): array
    {
        return [
            'trigger_type' => 'manual',
            'scanner_version' => '1.0.0',
            'projects' => [
                [
                    'docker_project_name' => 'deployment',
                    'ecosystems' => [
                        [
                            'ecosystem' => 'composer',
                            'manifest_file' => 'composer.lock',
                            'status' => 'completed',
                            'findings' => [
                                [
                                    'finding_type' => 'vulnerability',
                                    'finding_id' => 'GHSA-xxxx-yyyy-zzzz',
                                    'vulnerability_id' => 'CVE-2026-12345',
                                    'package_name' => 'vendor/package',
                                    'installed_version' => '1.0.0',
                                    'fixed_version' => '1.0.1',
                                    'severity' => 'high',
                                    'title' => 'Test dependency vulnerability',
                                    'description' => 'Test advisory description',
                                    'primary_url' => 'https://example.com/advisory',
                                    'metadata' => [
                                        'dependency_type' => 'direct',
                                    ],
                                ],
                            ],
                        ],
                    ],
                ],
            ],
        ];
    }

    private function submitScan(DockerHost $host, array $payload)
    {
        $token = $host->createToken('test-agent', ['scan:write']);

        return $this
            ->withToken($token->plainTextToken)
            ->postJson('/api/agent/scans', $payload);
    }

    public function test_agent_can_submit_dependency_scan_results(): void
    {
        $host = $this->createHost();
        $project = $this->createProject($host);

        $response = $this->submitScan($host, $this->scanPayload());

        $response
            ->assertOk()
            ->assertJsonPath('projects.0.status', 'completed')
            ->assertJsonPath('projects.0.ecosystems_scanned', 1)
            ->assertJsonPath('projects.0.findings_received', 1);

        $this->assertDatabaseHas('scan_runs', [
            'project_id' => $project->id,
            'status' => 'completed',
            'scanner_version' => '1.0.0',
        ]);

        $this->assertDatabaseHas('scan_findings', [
            'project_id' => $project->id,
            'target_type' => 'source_dependency',
            'target_identifier' => 'composer:composer.lock',
            'finding_type' => 'vulnerability',
            'finding_id' => 'GHSA-xxxx-yyyy-zzzz',
            'vulnerability_id' => 'CVE-2026-12345',
            'package_name' => 'vendor/package',
            'installed_version' => '1.0.0',
            'severity' => 'high',
            'status' => 'new',
        ]);

        $this->assertDatabaseCount('scan_run_findings', 1);
    }

    public function test_existing_finding_is_reused_on_next_scan(): void
    {
        $host = $this->createHost();
        $project = $this->createProject($host);

        $this->submitScan($host, $this->scanPayload())
            ->assertOk();

        $finding = ScanFinding::where('project_id', $project->id)
            ->firstOrFail();

        $firstSeenAt = $finding->first_seen_at;

        $payload = $this->scanPayload();

        // Simulate a package version change without changing the advisory.
        $payload['projects'][0]['ecosystems'][0]['findings'][0]['installed_version'] = '1.0.1';

        $this->submitScan($host, $payload)
            ->assertOk()
            ->assertJsonPath('projects.0.findings_received', 1);

        $this->assertDatabaseCount('scan_findings', 1);
        $this->assertDatabaseCount('scan_runs', 2);
        $this->assertDatabaseCount('scan_run_findings', 2);

        $finding->refresh();

        $this->assertSame('existing', $finding->status);
        $this->assertEquals($firstSeenAt, $finding->first_seen_at);
    }

    public function test_failed_ecosystem_does_not_resolve_existing_findings(): void
    {
        $host = $this->createHost();
        $project = $this->createProject($host);

        $this->submitScan($host, $this->scanPayload())
            ->assertOk();

        $finding = ScanFinding::where('project_id', $project->id)
            ->firstOrFail();

        $payload = $this->scanPayload();

        $payload['projects'][0]['ecosystems'][0] = [
            'ecosystem' => 'composer',
            'manifest_file' => 'composer.lock',
            'status' => 'failed',
            'error_message' => 'Composer audit failed.',
            'findings' => [],
        ];

        $this->submitScan($host, $payload)
            ->assertOk()
            ->assertJsonPath('projects.0.status', 'failed');

        $this->assertSame('new', $finding->fresh()->status);
    }

    public function test_completed_empty_scan_resolves_old_findings(): void
    {
        $host = $this->createHost();
        $project = $this->createProject($host);

        $this->submitScan($host, $this->scanPayload())
            ->assertOk();

        $finding = ScanFinding::where('project_id', $project->id)
            ->firstOrFail();

        $payload = $this->scanPayload();

        $payload['projects'][0]['ecosystems'][0]['findings'] = [];

        $this->submitScan($host, $payload)
            ->assertOk()
            ->assertJsonPath('projects.0.status', 'completed')
            ->assertJsonPath('projects.0.findings_received', 0);

        $this->assertSame('resolved', $finding->fresh()->status);
    }

    public function test_agent_cannot_submit_scan_for_another_host_project(): void
    {
        $host = $this->createHost('Host A');
        $otherHost = $this->createHost('Host B');

        $this->createProject($otherHost);

        $response = $this->submitScan($host, $this->scanPayload());

        $response->assertNotFound();

        $this->assertDatabaseCount('scan_runs', 0);
        $this->assertDatabaseCount('scan_findings', 0);
    }

    public function test_agent_without_scan_ability_is_forbidden(): void
    {
        $host = $this->createHost();

        $token = $host->createToken(
            'test-agent',
            ['inventory:write', 'heartbeat:write']
        );

        $response = $this
            ->withToken($token->plainTextToken)
            ->postJson('/api/agent/scans', $this->scanPayload());

        $response->assertForbidden();

        $this->assertDatabaseCount('scan_runs', 0);
    }
    public function test_agent_cannot_scan_project_when_source_scanning_is_disabled(): void
    {
        $host = $this->createHost();
        $project = $this->createProject($host);

        $project->update([
            'scan_source' => false,
        ]);

        $this->submitScan($host, $this->scanPayload())
            ->assertForbidden();

        $this->assertDatabaseCount('scan_runs', 0);
        $this->assertDatabaseCount('scan_findings', 0);
    }

    public function test_agent_cannot_scan_archived_project(): void
    {
        $host = $this->createHost();
        $project = $this->createProject($host);

        $project->update([
            'status' => 'archived',
        ]);

        $this->submitScan($host, $this->scanPayload())
            ->assertForbidden();

        $this->assertDatabaseCount('scan_runs', 0);
        $this->assertDatabaseCount('scan_findings', 0);
    }

    public function test_agent_cannot_run_weekly_scan_when_weekly_policy_is_disabled(): void
    {
        $host = $this->createHost();
        $project = $this->createProject($host);

        $project->update([
            'weekly_scan_enabled' => false,
        ]);

        $payload = $this->scanPayload();
        $payload['trigger_type'] = 'scheduled';

        $this->submitScan($host, $payload)
            ->assertForbidden();

        $this->assertDatabaseCount('scan_runs', 0);
    }

    public function test_agent_can_submit_discovery_scan(): void
    {
        $host = $this->createHost();
        $this->createProject($host);

        $payload = $this->scanPayload();
        $payload['trigger_type'] = 'discovery';

        $this->submitScan($host, $payload)
            ->assertOk()
            ->assertJsonPath('projects.0.status', 'completed');

        $this->assertDatabaseCount('scan_runs', 1);
    }

    public function test_unauthorized_project_prevents_entire_batch_from_being_saved(): void
    {
        $host = $this->createHost();
        $this->createProject($host);

        Project::create([
            'docker_host_id' => $host->id,
            'name' => 'archived-project',
            'docker_project_name' => 'archived-project',
            'status' => 'archived',
            'scan_source' => true,
            'weekly_scan_enabled' => true,
        ]);

        $payload = $this->scanPayload();

        $secondProject = $payload['projects'][0];
        $secondProject['docker_project_name'] = 'archived-project';

        $payload['projects'][] = $secondProject;

        $this->submitScan($host, $payload)
            ->assertForbidden();

        $this->assertDatabaseCount('scan_runs', 0);
        $this->assertDatabaseCount('scan_findings', 0);
    }
    public function test_resolved_finding_is_reopened_when_detected_again(): void
    {
        $host = $this->createHost();
        $project = $this->createProject($host);

        // First scan: finding is new.
        $this->submitScan($host, $this->scanPayload())
            ->assertOk();

        $finding = ScanFinding::where('project_id', $project->id)
            ->firstOrFail();

        $firstSeenAt = $finding->first_seen_at;

        $this->assertSame('new', $finding->status);

        // Second scan: finding disappears from a complete scan.
        $emptyPayload = $this->scanPayload();

        $emptyPayload['projects'][0]['ecosystems'][0]['findings'] = [];

        $this->submitScan($host, $emptyPayload)
            ->assertOk();

        $this->assertSame(
            'resolved',
            $finding->fresh()->status
        );

        // Third scan: finding appears again.
        $reopenedPayload = $this->scanPayload();

        $reopenedPayload['projects'][0]['ecosystems'][0]['findings'][0]['installed_version'] = '1.0.1';

        $this->submitScan($host, $reopenedPayload)
            ->assertOk();

        $finding->refresh();
        $this->assertSame(
            '1.0.1',
            $finding->installed_version
        );
        $this->assertSame('reopened', $finding->status);

        // first_seen_at must not be reset.
        $this->assertEquals(
            $firstSeenAt,
            $finding->first_seen_at
        );

        $this->assertDatabaseCount('scan_runs', 3);
        $this->assertDatabaseCount('scan_findings', 1);
        $this->assertDatabaseCount('scan_run_findings', 2);
    }
    public function test_completed_ecosystem_resolves_findings_even_when_another_is_unsupported(): void
    {
        $host = $this->createHost();
        $project = $this->createProject($host);

        // Initial scan: Composer and npm both completed.
        $initialPayload = $this->scanPayload();

        $initialPayload['projects'][0]['ecosystems'][] = [
            'ecosystem' => 'npm',
            'manifest_file' => 'package-lock.json',
            'status' => 'completed',
            'findings' => [
                [
                    'finding_type' => 'vulnerability',
                    'finding_id' => 'GHSA-npm-xxxx-yyyy',
                    'package_name' => 'axios',
                    'installed_version' => '1.0.0',
                    'severity' => 'high',
                    'title' => 'Test npm vulnerability',
                ],
            ],
        ];

        $this->submitScan($host, $initialPayload)
            ->assertOk()
            ->assertJsonPath('projects.0.status', 'completed');

        $composerFinding = ScanFinding::where(
            'finding_id',
            'GHSA-xxxx-yyyy-zzzz'
        )->firstOrFail();

        $npmFinding = ScanFinding::where(
            'finding_id',
            'GHSA-npm-xxxx-yyyy'
        )->firstOrFail();

        // Second scan:
        // Composer completed with zero findings.
        // npm is unsupported.
        $nextPayload = $this->scanPayload();

        $nextPayload['projects'][0]['ecosystems'][0]['findings'] = [];

        $nextPayload['projects'][0]['ecosystems'][] = [
            'ecosystem' => 'npm',
            'manifest_file' => 'package-lock.json',
            'status' => 'unsupported',
            'error_message' => 'npm lockfile not found.',
            'findings' => [],
        ];

        $this->submitScan($host, $nextPayload)
            ->assertOk()
            ->assertJsonPath('projects.0.status', 'partial');

        // Composer was successfully audited.
        $this->assertSame(
            'resolved',
            $composerFinding->fresh()->status
        );

        // npm was unsupported, so its previous finding remains.
        $this->assertSame(
            'new',
            $npmFinding->fresh()->status
        );
    }
    public function test_duplicate_finding_in_same_scan_payload_is_processed_once(): void
    {
        $host = $this->createHost();
        $project = $this->createProject($host);

        $payload = $this->scanPayload();

        $finding = $payload['projects'][0]['ecosystems'][0]['findings'][0];

        // Simulate duplicated finding from scanner payload.
        $payload['projects'][0]['ecosystems'][0]['findings'][] = $finding;

        $this->submitScan($host, $payload)
            ->assertOk()
            ->assertJsonPath('projects.0.findings_received', 1);

        $this->assertDatabaseCount('scan_runs', 1);
        $this->assertDatabaseCount('scan_findings', 1);
        $this->assertDatabaseCount('scan_run_findings', 1);

        $this->assertDatabaseHas('scan_findings', [
            'project_id' => $project->id,
            'finding_id' => 'GHSA-xxxx-yyyy-zzzz',
        ]);
    }
}
