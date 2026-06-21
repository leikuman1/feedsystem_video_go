param(
    [Parameter(Mandatory = $true)]
    [string]$BaseUrl,

    [Parameter(Mandatory = $true)]
    [string]$Username,

    [Parameter(Mandatory = $true)]
    [string]$Password
)

$ErrorActionPreference = "Stop"
$base = $BaseUrl.TrimEnd("/")

function Invoke-JsonPost {
    param(
        [string]$Path,
        [object]$Body,
        [hashtable]$Headers = @{}
    )

    return Invoke-RestMethod `
        -Method Post `
        -Uri "$base/api$Path" `
        -Headers $Headers `
        -ContentType "application/json" `
        -Body ($Body | ConvertTo-Json -Depth 8)
}

Write-Host "[smoke] health"
$health = Invoke-RestMethod -Method Get -Uri "$base/api/healthz"
if ($health.status -ne "ok") {
    throw "health check failed"
}

Write-Host "[smoke] login"
$login = Invoke-JsonPost -Path "/account/login" -Body @{
    username = $Username
    password = $Password
}
if (-not $login.token) {
    throw "login response did not contain token"
}
$headers = @{ Authorization = "Bearer $($login.token)" }

Write-Host "[smoke] authenticated feed"
$feed = Invoke-JsonPost -Path "/feed/listLatest" -Headers $headers -Body @{
    limit       = 3
    latest_time = 0
}
if ($null -eq $feed.video_list) {
    throw "feed response did not contain video_list"
}

Write-Host "[smoke] notifications"
$notifications = Invoke-JsonPost -Path "/notification/list" -Headers $headers -Body @{}
if ($null -eq $notifications.notifications) {
    throw "notification response did not contain notifications"
}

Write-Host "[smoke] architecture route"
$architecture = Invoke-WebRequest -Method Get -Uri "$base/architecture"
if ($architecture.StatusCode -ne 200) {
    throw "architecture page failed"
}

Write-Host "[smoke] passed"
