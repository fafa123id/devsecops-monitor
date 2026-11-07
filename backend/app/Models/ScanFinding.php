<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;

class ScanFinding extends Model
{
    protected $fillable = [
        'project_id',
        'last_scan_run_id',
        'target_type',
        'target_identifier',
        'finding_type',
        'finding_id',
        'vulnerability_id',
        'package_name',
        'installed_version',
        'fixed_version',
        'severity',
        'title',
        'description',
        'primary_url',
        'metadata',
        'status',
        'finding_fingerprint',
        'first_seen_at',
        'last_seen_at',
    ];

    protected function casts(): array
    {
        return [
            'metadata' => 'array',
            'first_seen_at' => 'datetime',
            'last_seen_at' => 'datetime',
        ];
    }

    public function lastScanRun(): BelongsTo
    {
        return $this->belongsTo(ScanRun::class, 'last_scan_run_id');
    }

    public function project(): BelongsTo
    {
        return $this->belongsTo(Project::class);
    }

    public function scanRuns(): BelongsToMany
    {
        return $this->belongsToMany(
            ScanRun::class,
            'scan_run_findings'
        )->withTimestamps();
    }
}
