<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Laravel\Sanctum\HasApiTokens;

class DockerHost extends Model
{
    use HasApiTokens;

    protected $fillable = [
        'name',
        'hostname',
        'ip_address',
        'agent_version',
        'last_heartbeat_at',
        'status',
    ];

    protected function casts(): array
    {
        return [
            'last_heartbeat_at' => 'datetime',
        ];
    }

    public function projects(): HasMany
    {
        return $this->hasMany(Project::class);
    }
}
