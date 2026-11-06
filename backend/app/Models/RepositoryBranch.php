<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class RepositoryBranch extends Model
{
    protected $fillable = [
        'repository_id',
        'name',
        'last_commit_sha',
        'is_default',
        'is_deployed',
        'last_synced_at',
    ];

    protected function casts(): array
    {
        return [
            'is_default' => 'boolean',
            'is_deployed' => 'boolean',
            'last_synced_at' => 'datetime',
        ];
    }

    public function repository(): BelongsTo
    {
        return $this->belongsTo(Repository::class);
    }
}
