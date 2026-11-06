<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class ScanFinding extends Model
{
    protected $fillable = [
        'scan_run_id',
        'project_id',
        'target_type',
        'target_identifier',
        'vulnerability_id',
        'package_name',
        'installed_version',
        'fixed_version',
        'severity',
        'title',
        'description',
        'primary_url',
        'status',
        'finding_fingerprint',
    ];

    public function scanRun(): BelongsTo
    {
        return $this->belongsTo(ScanRun::class);
    }

    public function project(): BelongsTo
    {
        return $this->belongsTo(Project::class);
    }
}
