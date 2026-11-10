<?php

namespace App\Services;

use App\Models\DockerHost;
use App\Models\Notification;
use App\Models\ScanFinding;
use App\Models\ScanRun;
use App\Models\User;

class NotificationService
{
    public function notifyFinding(
        ScanFinding $finding,
        ScanRun $scanRun,
        string $event
    ): void {
        if ($finding->finding_type !== 'vulnerability') {
            return;
        }

        if (!in_array($event, ['new', 'reopened'], true)) {
            return;
        }

        $severity = strtolower($finding->severity);

        $isHighImpact = in_array(
            $severity,
            ['critical', 'high'],
            true
        );

        $title = match (true) {
            $event === 'reopened' && $isHighImpact =>
            strtoupper($severity) . ' vulnerability reopened',

            $event === 'reopened' =>
            'Vulnerability reopened',

            $isHighImpact =>
            strtoupper($severity) . ' vulnerability detected',

            default =>
            'New vulnerability detected',
        };

        $projectName = $scanRun->project?->name
            ?? $scanRun->project?->docker_project_name
            ?? 'Unknown project';

        $packageName = $finding->package_name
            ?? 'Unknown package';

        $vulnerabilityId = $finding->vulnerability_id
            ?? $finding->finding_id;

        $message = $event === 'reopened'
            ? "{$vulnerabilityId} in {$packageName} has been detected again in {$projectName}."
            : "{$vulnerabilityId} in {$packageName} was detected in {$projectName}.";

        $metadata = $finding->metadata ?? [];

        User::query()
            ->select('id')
            ->each(function (User $user) use (
                $finding,
                $scanRun,
                $severity,
                $title,
                $message,
                $event,
                $metadata
            ) {
                Notification::create([
                    'user_id' => $user->id,
                    'project_id' => $finding->project_id,
                    'scan_run_id' => $scanRun->id,
                    'scan_finding_id' => $finding->id,
                    'type' => $event === 'reopened'
                        ? 'vulnerability_reopened'
                        : 'vulnerability_detected',
                    'severity' => $severity,
                    'title' => $title,
                    'message' => $message,
                    'data' => [
                        'event' => $event,
                        'package_name' => $finding->package_name,
                        'vulnerability_id' => $finding->vulnerability_id,
                        'installed_version' => $finding->installed_version,
                        'fixed_version' => $finding->fixed_version,
                        'ecosystem' => $metadata['ecosystem'] ?? null,
                        'manifest_file' => $metadata['manifest_file'] ?? null,
                    ],
                ]);
            });
    }
    public function notifyScanFailed(ScanRun $scanRun): void
    {
        if ($scanRun->status !== 'failed') {
            return;
        }

        $projectName = $scanRun->project?->name
            ?? $scanRun->project?->docker_project_name
            ?? 'Unknown project';

        $errorMessage = $scanRun->error_message
            ?? 'Dependency scanning failed without details.';

        User::query()
            ->select('id')
            ->each(function (User $user) use (
                $scanRun,
                $projectName,
                $errorMessage
            ) {
                Notification::create([
                    'user_id' => $user->id,
                    'project_id' => $scanRun->project_id,
                    'scan_run_id' => $scanRun->id,
                    'type' => 'scan_failed',
                    'title' => 'Dependency scan failed',
                    'message' => "Scanning {$projectName} failed: {$errorMessage}",
                    'data' => [
                        'trigger_type' => $scanRun->trigger_type,
                        'status' => $scanRun->status,
                    ],
                ]);
            });
    }
    public function notifyAgentOffline(DockerHost $host): void
    {
        User::query()
            ->select('id')
            ->each(function (User $user) use ($host) {
                Notification::create([
                    'user_id' => $user->id,
                    'type' => 'agent_offline',
                    'title' => 'Docker agent went offline',
                    'message' => "Docker agent {$host->name} ({$host->hostname}) is no longer sending heartbeats.",
                    'data' => [
                        'docker_host_id' => $host->id,
                        'hostname' => $host->hostname,
                        'last_heartbeat_at' => $host->last_heartbeat_at?->toISOString(),
                    ],
                ]);
            });
    }
}
