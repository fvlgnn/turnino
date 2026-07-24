# Public monitor kiosk deployment

The production monitor is available at `/monitor`. It contains no sound, fullscreen or Caller controls. The only operational indicator is a small connection dot: green while SSE is connected and amber while reconnecting.

The development view `/monitor?test=1` keeps explicit controls for sound, fullscreen and opening the Caller in a second tab.

## Why the browser must provide kiosk capabilities

Ordinary pages cannot reliably enter fullscreen or start audible playback without a user gesture. These are browser security policies and are unaffected by using an OGG file instead of the Web Audio API.

A production display must therefore launch its browser in kiosk mode with an autoplay policy. Turnino attempts to start its audio context automatically; a browser started with the documented autoplay option permits it. Test mode retains the button as a fallback for an ordinary browser.

Configuration stays split by responsibility:

| Value | Configuration |
|---|---|
| Installation name, default language, sound, recent calls, highlight duration | Turnino administration settings |
| Monitor URL, forced language, scale workaround, browser restart delay | Kiosk launcher parameters/environment |
| Fullscreen and autoplay permission | Browser kiosk flags |

This avoids duplicating application settings in a second configuration file. The launcher parameters can later be placed in an operating-system task or environment file without changing Turnino.

## Windows and Microsoft Edge

The provided PowerShell supervisor launches Edge as digital signage:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass `
  -File .\deploy\kiosk\windows\start-monitor.ps1
```

When running the repository copy directly through `\\wsl.localhost`, invoke a new PowerShell process with the execution policy override instead of executing the UNC script directly:

```powershell
$script = "\\wsl.localhost\<distro>\home\<user>\turnino\deploy\kiosk\windows\start-monitor.ps1"
powershell.exe -NoProfile -ExecutionPolicy Bypass `
  -File $script
```

For a permanent installation, copy the script to `C:\Turnino` as described below. This also avoids the security prompt applied to scripts opened from a UNC path.

### PowerShell trust warning for `\\wsl.localhost`

Windows can classify a script opened directly through the WSL UNC path as remote. The recommended workaround is to copy and unblock it:

```powershell
New-Item -ItemType Directory -Force C:\Turnino
$source = "\\wsl.localhost\<distro>\home\<user>\turnino\deploy\kiosk\windows\start-monitor.ps1"
Copy-Item `
  $source `
  "C:\Turnino\start-monitor.ps1"
Unblock-File "C:\Turnino\start-monitor.ps1"
```

Then run the Windows copy:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass `
  -File "C:\Turnino\start-monitor.ps1"
```

Do not globally relax the PowerShell execution policy. See `docs/security.md` for the security boundaries and symptom-based troubleshooting.

Its default URL is:

```text
http://localhost:8080/monitor
```

This uses WSL's hostname-based localhost forwarding and also works when Docker Desktop publishes the port on Windows. Override it when the server is remote:

```powershell
.\deploy\kiosk\windows\start-monitor.ps1 `
  -Url "http://turnino.local:8080/monitor"
```

Optional launcher parameters:

```powershell
.\deploy\kiosk\windows\start-monitor.ps1 `
  -Url "http://localhost:8080/monitor" `
  -Language it `
  -ScaleFactor 1.0 `
  -RestartDelaySeconds 3
```

`ScaleFactor` accepts values from `0.5` to `2.0`. Keep `1.0` unless a specific Chromium/Edge display needs a workaround. Turnino's responsive layout is expected to fit at 100%; the scale flag is not the primary layout mechanism.

The launcher uses:

```text
--kiosk
--edge-kiosk-type=fullscreen
--no-first-run
--autoplay-policy=no-user-gesture-required
```

It also gives Turnino a separate Edge profile. When Edge exits or crashes, the PowerShell loop starts it again after three seconds. Close the PowerShell supervisor to stop intentional restarts.

### Start automatically when the monitor user signs in

First copy the script to a stable location, for example:

```text
C:\Turnino\start-monitor.ps1
```

From PowerShell, create a task for the current user:

```powershell
$script = "C:\Turnino\start-monitor.ps1"
$action = New-ScheduledTaskAction `
  -Execute "powershell.exe" `
  -Argument "-NoProfile -WindowStyle Hidden -ExecutionPolicy Bypass -File `"$script`""
$trigger = New-ScheduledTaskTrigger -AtLogOn -User $env:USERNAME
$settings = New-ScheduledTaskSettingsSet `
  -ExecutionTimeLimit (New-TimeSpan -Days 0) `
  -RestartCount 3 `
  -RestartInterval (New-TimeSpan -Minutes 1)

Register-ScheduledTask `
  -TaskName "Turnino Monitor" `
  -Action $action `
  -Trigger $trigger `
  -Settings $settings `
  -Description "Start the Turnino public monitor in Edge kiosk mode." `
  -Force
```

Test and remove the task with:

```powershell
Start-ScheduledTask -TaskName "Turnino Monitor"
Stop-ScheduledTask -TaskName "Turnino Monitor"
Unregister-ScheduledTask -TaskName "Turnino Monitor" -Confirm:$false
```

For a locked-down appliance, Windows Assigned Access or Shell Launcher can replace the normal desktop and restart Edge at the operating-system level. The included supervisor is intentionally simpler for development and small installations.

Microsoft references:

- <https://learn.microsoft.com/en-us/deployedge/microsoft-edge-configure-kiosk-mode>
- <https://learn.microsoft.com/en-us/troubleshoot/microsoft-edge/development/video-playback-issues>

## Raspberry Pi OS and Chromium

Use Raspberry Pi OS with Desktop and automatic desktop login. Install the launcher:

```sh
mkdir -p ~/.local/bin
cp deploy/kiosk/raspberry-pi/start-monitor.sh ~/.local/bin/turnino-monitor
chmod +x ~/.local/bin/turnino-monitor
```

The default URL assumes Turnino runs on the same Raspberry Pi:

```text
http://localhost:8080/monitor
```

For a different server, add the environment variable before the launcher command:

```sh
TURNINO_MONITOR_URL=http://turnino.local:8080/monitor ~/.local/bin/turnino-monitor
```

Additional environment configuration:

```sh
TURNINO_MONITOR_LANGUAGE=it
TURNINO_KIOSK_SCALE=1
TURNINO_KIOSK_RESTART_DELAY=3
```

The script discovers `chromium` or `chromium-browser`, starts it with kiosk and autoplay options, and restarts it after three seconds whenever it exits.

### Start automatically with the Raspberry Pi desktop

Current Raspberry Pi OS desktop releases use labwc. Create the autostart file if necessary:

```sh
mkdir -p ~/.config/labwc
```

Append this line to `~/.config/labwc/autostart`:

```sh
~/.local/bin/turnino-monitor &
```

Then reboot:

```sh
sudo reboot
```

Do not add `&` inside `start-monitor.sh`: the autostart entry backgrounds the supervisor, while the supervisor itself must wait for Chromium so it can detect and restart it.

For a remote Turnino server:

```sh
TURNINO_MONITOR_URL=http://turnino.local:8080/monitor ~/.local/bin/turnino-monitor &
```

Official Raspberry Pi kiosk tutorial:

- <https://www.raspberrypi.com/tutorials/how-to-use-a-raspberry-pi-in-kiosk-mode/>

## Operational verification

After startup:

1. confirm that the browser has no address bar or controls;
2. make a call and verify that the monitor updates through SSE;
3. verify audible playback without clicking the page;
4. stop and restart Turnino, then verify snapshot recovery;
5. close or terminate the browser and verify that the supervisor restarts it;
6. disconnect the network and verify that the connection dot becomes amber, then green after recovery.

Use test mode only during setup:

```text
http://localhost:8080/monitor?test=1
```

Production launchers deliberately use `/monitor` without `test=1`.

## Docker Desktop and LAN access

When Docker Desktop publishes `8080:8080` directly on Windows, Turnino is no longer reached through the WSL NAT boundary. A phone on the same LAN should be able to use the Windows LAN address, provided:

- Docker publishes the port on all host interfaces;
- Windows Firewall permits inbound TCP 8080;
- the phone is not on an isolated guest Wi-Fi or another VLAN.

Verify the Windows listener before testing from another device:

```powershell
netstat -ano | findstr :8080
curl.exe http://localhost:8080/healthz
```

In a WSL development setup, prefer `localhost` when that is the address confirmed by the health check.
