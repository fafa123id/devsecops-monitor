<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;
use Illuminate\Database\Eloquent\Relations\HasMany;

class ScanRun extends Model
{
    protected $fillable = [
        'project_id',
        'trigger_type',
        'status',
        'started_at',
        'completed_at',
        'scanner_version',
        'source_commit_sha',
        'error_message',
    ];

    protected function casts(): array
    {
        return [
            'started_at' => 'datetime',
            'completed_at' => 'datetime',
        ];
    }

    public function project(): BelongsTo
    {
        return $this->belongsTo(Project::class);
    }

    public function findings(): BelongsToMany
    {
        return $this->belongsToMany(
            ScanFinding::class,
            'scan_run_findings'
        )->withTimestamps();
    }
    public function ecosystems(): HasMany
    {
        return $this->hasMany(ScanRunEcosystem::class);
    }
}
