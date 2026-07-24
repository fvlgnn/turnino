param(
    [Parameter(Mandatory = $true)]
    [ValidateRange(1, 2147483647)]
    [int]$CounterID,
    [string]$BaseUrl = "http://localhost:8080",
    [int]$RestartDelaySeconds = 3,
    [ValidateSet("", "en", "it")]
    [string]$Language = "",
    [ValidateRange(0.5, 2.0)]
    [double]$ScaleFactor = 1.0
)

$launcher = Join-Path $PSScriptRoot "start-monitor.ps1"
$url = "$($BaseUrl.TrimEnd('/'))/counter-display?counter=$CounterID"

& $launcher `
    -Url $url `
    -RestartDelaySeconds $RestartDelaySeconds `
    -Language $Language `
    -ScaleFactor $ScaleFactor
