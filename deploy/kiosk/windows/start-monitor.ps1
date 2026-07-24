param(
    [string]$Url = "http://localhost:8080/monitor",
    [int]$RestartDelaySeconds = 3,
    [ValidateSet("", "en", "it")]
    [string]$Language = "",
    [ValidateRange(0.5, 2.0)]
    [double]$ScaleFactor = 1.0
)

$ErrorActionPreference = "Stop"

$edgeCandidates = @(
    @(
        "${env:ProgramFiles(x86)}\Microsoft\Edge\Application\msedge.exe",
        "${env:ProgramFiles}\Microsoft\Edge\Application\msedge.exe"
    ) | Where-Object { $_ -and (Test-Path $_ -PathType Leaf) }
)

if ($edgeCandidates.Count -eq 0) {
    throw "Microsoft Edge was not found."
}

$edge = $edgeCandidates[0]
if (-not (Test-Path $edge -PathType Leaf)) {
    throw "Microsoft Edge executable is invalid: $edge"
}
$profile = Join-Path $env:LOCALAPPDATA "Turnino\EdgeKiosk"
$launchUrl = $Url
if ($Language) {
    $separator = if ($launchUrl.Contains("?")) { "&" } else { "?" }
    $launchUrl = "${launchUrl}${separator}lang=${Language}"
}
$arguments = @(
    "--kiosk",
    $launchUrl,
    "--edge-kiosk-type=fullscreen",
    "--no-first-run",
    "--autoplay-policy=no-user-gesture-required",
    "--disable-session-crashed-bubble",
    "--user-data-dir=`"$profile`""
)
if ($ScaleFactor -ne 1.0) {
    $scaleArgument = $ScaleFactor.ToString([System.Globalization.CultureInfo]::InvariantCulture)
    $arguments += "--force-device-scale-factor=$scaleArgument"
}

Write-Host "Turnino kiosk URL: $launchUrl"
Write-Host "Turnino kiosk scale factor: $ScaleFactor"
Write-Host "Close this PowerShell window to stop the kiosk supervisor."

while ($true) {
    try {
        $process = Start-Process -FilePath $edge -ArgumentList $arguments -PassThru
        $process.WaitForExit()
    }
    catch {
        Write-Warning $_
    }
    Start-Sleep -Seconds $RestartDelaySeconds
}
