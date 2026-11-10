<?php

use App\Http\Controllers\Api\ScanHistoryController;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Route;
use App\Http\Controllers\Api\Agent\HeartbeatController;
use App\Http\Controllers\Api\Agent\InventoryController;
use App\Http\Controllers\Api\Agent\ProjectPolicyController;
use App\Http\Controllers\Api\Agent\ScanController;
use App\Http\Controllers\Api\AgentController;
use App\Http\Controllers\Api\AuthController;
use App\Http\Controllers\Api\DashboardController;
use App\Http\Controllers\Api\NotificationController;
use App\Http\Controllers\Api\ProjectController;
use App\Http\Controllers\Api\VulnerabilityController;

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
Route::prefix('auth')->group(function () {
    Route::post('/login', [AuthController::class, 'login'])
        ->middleware('throttle:6,1');

    Route::middleware('auth:sanctum')->group(function () {
        Route::get('/me', [AuthController::class, 'me']);
        Route::post('/logout', [AuthController::class, 'logout']);
    });
});
Route::middleware('auth:sanctum')->group(function () {
    Route::get('/notifications', [NotificationController::class, 'index']);
    Route::patch('/notifications/{id}/read', [NotificationController::class, 'markAsRead']);
    Route::patch('/notifications/read-all', [NotificationController::class, 'markAllAsRead']);
    Route::get('/agents', [AgentController::class, 'index']);
    Route::get('/scans', [ScanHistoryController::class, 'index']);
    Route::get('/vulnerabilities', [VulnerabilityController::class, 'index']);
    Route::get('/projects', [ProjectController::class, 'index']);
    Route::get('/projects/{project}', [ProjectController::class, 'show']);
    Route::get('/dashboard/vulnerability-trend', [
        DashboardController::class,
        'vulnerabilityTrend',
    ]);
});
