<?php

namespace Tests\Feature;

use Illuminate\Support\Facades\DB;
use Tests\TestCase;

class DatabaseTargetTest extends TestCase
{
    public function test_phpunit_uses_isolated_test_database(): void
    {
        $database = DB::selectOne(
            'SELECT current_database() AS name'
        )->name;

        $this->assertSame(
            'devsecops_test',
            $database,
            "PHPUnit menggunakan database {$database}, bukan devsecops_test."
        );
    }
}