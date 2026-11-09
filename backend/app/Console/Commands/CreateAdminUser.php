<?php

namespace App\Console\Commands;

use App\Models\User;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\Validator;

class CreateAdminUser extends Command
{
    protected $signature = 'app:create-admin';

    protected $description = 'Create a dashboard administrator account';

    public function handle(): int
    {
        $this->info('Create Dashboard Administrator');
        $this->newLine();

        $name = trim($this->ask('Full name'));
        $username = strtolower(trim($this->ask('Username')));
        $email = strtolower(trim($this->ask('Email')));
        $password = $this->secret('Password (minimum 12 characters)');
        $confirmation = $this->secret('Confirm password');

        if ($password !== $confirmation) {
            $this->error('Password confirmation does not match.');

            return self::FAILURE;
        }

        $validator = Validator::make([
            'name' => $name,
            'username' => $username,
            'email' => $email,
            'password' => $password,
        ], [
            'name' => ['required', 'string', 'max:255'],
            'username' => [
                'required',
                'string',
                'min:3',
                'max:32',
                'alpha_dash:ascii',
                'unique:users,username',
            ],
            'email' => ['required', 'email', 'max:255', 'unique:users,email'],
            'password' => ['required', 'string', 'min:12'],
        ]);

        if ($validator->fails()) {
            foreach ($validator->errors()->all() as $error) {
                $this->error($error);
            }

            return self::FAILURE;
        }

        $user = User::create([
            'name' => $name,
            'username' => $username,
            'email' => $email,
            'password' => $password,
        ]);

        $this->newLine();
        $this->info('Administrator account created successfully.');
        $this->table(
            ['ID', 'Name', 'Username', 'Email'],
            [[
                $user->id,
                $user->name,
                $user->username,
                $user->email,
            ]]
        );

        return self::SUCCESS;
    }
}