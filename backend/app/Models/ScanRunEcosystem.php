<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class ScanRunEcosystem extends Model
{
    protected $fillable = [
        'scan_run_id',
        'ecosystem',
        'manifest_file',
        'status',
        'error_message',
    ];

    public function scanRun(): BelongsTo
    {
        return $this->belongsTo(ScanRun::class);
    }
}
