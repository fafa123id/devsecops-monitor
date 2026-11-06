<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class ProjectContainer extends Model
{
    protected $fillable = [
        'project_id',
        'container_id',
        'container_name',
        'service_name',
        'image_reference',
        'image_digest',
        'container_state',
    ];

    public function project(): BelongsTo
    {
        return $this->belongsTo(Project::class);
    }
}
