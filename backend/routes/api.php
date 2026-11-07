<?php

use Illuminate\Http\Request;
use Illuminate\Support\Facades\Route;
use App\Http\Controllers\Api\Agent\HeartbeatController;
use App\Http\Controllers\Api\Agent\InventoryController;
use App\Http\Controllers\Api\Agent\ProjectPolicyController;
use App\Http\Controllers\Api\Agent\ScanController;

Route::middleware('auth:sanctum')
    ->prefix('agent')
    ->group(function () {
        Route::get('/me', function (Request $request) {
            $host = $request->user();

            return response()->json([
                'message' => 'Agent authenticated successfully.',
                'host' => [
                    'id' => $host->id,
                    'name' => $host->name,
                    'hostname' => $host->hostname,
                    'status' => $host->status,
                ],
            ]);
        });
        Route::get('/policies', ProjectPolicyController::class)
            ->middleware('abilities:inventory:write');
        Route::post('/scans', ScanController::class)
            ->middleware('abilities:scan:write');
        Route::post('/inventory', InventoryController::class)
            ->middleware('abilities:inventory:write');
        Route::post('/heartbeat', HeartbeatController::class)
            ->middleware('abilities:heartbeat:write');
    });
