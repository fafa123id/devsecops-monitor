<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

class Project extends Model
{
    protected $fillable = [
        'docker_host_id',
        'name',
        'docker_project_name',
        'source_path_reference',
        'scan_source',
        'weekly_scan_enabled',
        'status',
        'last_scanned_at',
        'last_seen_at',
        'missing_since_at',
        'missing_snapshots',
    ];

    protected function casts(): array
    {
        return [
            'last_seen_at' => 'datetime',
            'missing_since_at' => 'datetime',
            'missing_snapshots' => 'integer',
            'scan_source' => 'boolean',
            'weekly_scan_enabled' => 'boolean',
            'last_scanned_at' => 'datetime',
        ];
    }
    public function scanRuns(): HasMany
    {
        return $this->hasMany(ScanRun::class);
    }
    public function dockerHost(): BelongsTo
    {
        return $this->belongsTo(DockerHost::class);
    }

    public function containers(): HasMany
    {
        return $this->hasMany(ProjectContainer::class);
    }
    public function findings(): HasMany
    {
        return $this->hasMany(ScanFinding::class);
    }
    public function repositories(): HasMany
    {
        return $this->hasMany(Repository::class);
    }
}
