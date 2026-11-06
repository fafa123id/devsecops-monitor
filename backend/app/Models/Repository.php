<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

class Repository extends Model
{
    protected $fillable = [
        'project_id',
        'provider',
        'external_repository_id',
        'owner',
        'name',
        'full_name',
        'html_url',
        'clone_url',
        'default_branch',
        'is_private',
        'installation_id',
        'last_synced_at',
    ];

    protected function casts(): array
    {
        return [
            'is_private' => 'boolean',
            'last_synced_at' => 'datetime',
        ];
    }

    public function project(): BelongsTo
    {
        return $this->belongsTo(Project::class);
    }

    public function branches(): HasMany
    {
        return $this->hasMany(RepositoryBranch::class);
    }
}
