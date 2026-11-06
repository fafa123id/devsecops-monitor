<?php

namespace App\Http\Controllers\Api\Agent;

use App\Http\Controllers\Controller;
use Illuminate\Http\Request;

class HeartbeatController extends Controller
{
    public function __invoke(Request $request)
    {
        $request->validate([
            'agent_version' => ['required', 'string', 'max:50'],
        ]);

        $host = $request->user();

        $host->update([
            'agent_version' => $request->agent_version,
            'last_heartbeat_at' => now(),
            'status' => 'online',
        ]);

        return response()->json([
            'message' => 'Heartbeat received.',
            'host_id' => $host->id,
            'status' => $host->status,
            'last_heartbeat_at' => $host->last_heartbeat_at,
        ]);
    }
}
