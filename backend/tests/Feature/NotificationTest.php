<?php

namespace Tests\Feature;

use App\Models\Notification;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class NotificationTest extends TestCase
{
    use RefreshDatabase;

    public function test_user_can_only_see_their_own_notifications(): void
    {
        $user = User::factory()->create();
        $anotherUser = User::factory()->create();

        Notification::create([
            'user_id' => $user->id,
            'type' => 'vulnerability_detected',
            'severity' => 'critical',
            'title' => 'Critical vulnerability detected',
            'message' => 'A critical vulnerability was detected.',
        ]);

        Notification::create([
            'user_id' => $anotherUser->id,
            'type' => 'scan_failed',
            'title' => 'Scan failed',
            'message' => 'The scan failed.',
        ]);

        $this->actingAs($user)
            ->getJson('/api/notifications')
            ->assertOk()
            ->assertJsonCount(1, 'data')
            ->assertJsonPath('unread_count', 1);
    }

    public function test_user_can_mark_their_notification_as_read(): void
    {
        $user = User::factory()->create();

        $notification = Notification::create([
            'user_id' => $user->id,
            'type' => 'vulnerability_detected',
            'severity' => 'high',
            'title' => 'High vulnerability detected',
            'message' => 'A high vulnerability was detected.',
        ]);

        $this->actingAs($user)
            ->patchJson("/api/notifications/{$notification->id}/read")
            ->assertOk();

        $this->assertNotNull($notification->fresh()->read_at);
    }

    public function test_user_cannot_mark_another_users_notification_as_read(): void
    {
        $user = User::factory()->create();
        $anotherUser = User::factory()->create();

        $notification = Notification::create([
            'user_id' => $anotherUser->id,
            'type' => 'scan_failed',
            'title' => 'Scan failed',
            'message' => 'The scan failed.',
        ]);

        $this->actingAs($user)
            ->patchJson("/api/notifications/{$notification->id}/read")
            ->assertNotFound();

        $this->assertNull($notification->fresh()->read_at);
    }

    public function test_user_can_mark_all_notifications_as_read(): void
    {
        $user = User::factory()->create();

        foreach (['scan_failed', 'agent_offline'] as $type) {
            Notification::create([
                'user_id' => $user->id,
                'type' => $type,
                'title' => 'Notification',
                'message' => 'Test notification.',
            ]);
        }

        $this->actingAs($user)
            ->patchJson('/api/notifications/read-all')
            ->assertOk()
            ->assertJsonPath('updated', 2);

        $this->assertSame(
            0,
            $user->notifications()->whereNull('read_at')->count()
        );
    }
}