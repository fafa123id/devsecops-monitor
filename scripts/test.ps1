param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]] $TestArguments
)

$composeFile = Join-Path $PSScriptRoot "..\deployment\docker-compose.dev.yml"
$composeFile = (Resolve-Path $composeFile).Path

$dockerArgs = @(
    "compose",
    "-f", $composeFile,
    "exec",
    "-e", "DB_CONNECTION=pgsql",
    "-e", "DB_HOST=postgres",
    "-e", "DB_PORT=5432",
    "-e", "DB_DATABASE=devsecops_test",
    "-e", "DB_USERNAME=devsecops",
    "-e", "DB_PASSWORD=dev_password_local",
    "-e", "DB_URL=",
    "api",
    "php",
    "artisan",
    "test"
)

if ($TestArguments) {
    $dockerArgs += $TestArguments
}

& docker @dockerArgs

exit $LASTEXITCODE