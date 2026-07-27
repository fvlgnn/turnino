# Security model and troubleshooting boundaries

Turnino is designed for a trusted local network or for deployment behind HTTPS. This document identifies where each security control lives and helps distinguish application failures from networking or browser-policy failures.

## Network listener

`cmd/turnino/main.go` reads `TURNINO_ADDR` or `-addr`. The default `:8080` listens on every interface visible inside the current operating environment.

Turnino does not configure Windows Firewall, WSL or Docker forwarding, router VLANs, Wi-Fi client isolation, TLS certificates or a reverse proxy.

`ERR_CONNECTION_REFUSED` happens before an HTTP request reaches Turnino. It therefore cannot be caused by administrator authentication, CSRF, browser tokens or HTTP security headers.

```sh
# Inside Linux or WSL
curl -v http://localhost:8080/healthz
```

```powershell
# From Windows
curl.exe -v http://localhost:8080/healthz
netstat -ano | findstr :8080
```

In some WSL configurations, the special `localhost` forwarding path works while an explicit `127.0.0.1` or Windows LAN address does not. Use the address that passes the health check; this behavior belongs to WSL networking rather than Turnino.

## HTTP security headers

`securityHeaders` in `cmd/turnino/main.go` wraps the complete router and sets:

```text
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Referrer-Policy: no-referrer
```

These headers prevent MIME sniffing, framing and referrer leakage. They are applied after a TCP connection and HTTP response exist, so they cannot produce `ERR_CONNECTION_REFUSED`.

Inspect them with:

```sh
curl -I http://localhost:8080/monitor
```

## Administrator API

`adminOnly` in `internal/httpapi/api.go` protects `/api/v1/admin/*`:

1. administration must have been bootstrapped;
2. the request must contain `Authorization: Bearer <token>`;
3. Turnino hashes the presented token and compares it with the stored hash.

Expected failures:

- `401 Unauthorized`: missing or invalid Bearer token;
- `503 Service Unavailable`: administrator credential has not been configured.

## Administration browser UI

`internal/webui/webui.go` exchanges the administrator access key for a random in-memory session:

- cookie `HttpOnly`;
- `SameSite=Strict`;
- `Secure` when Turnino is served through TLS;
- twelve-hour expiration;
- session disappears on server restart;
- every state-changing form requires a session-specific CSRF token.

A missing or incorrect CSRF token produces `403 Forbidden`.

The administration page can rotate the access key. Rotation invalidates the previous key and all in-memory administrator sessions. If the key is lost, `turnino admin reset --confirm` provides an explicit offline recovery path; see `docs/administration.md`.

## Plain HTTP and the infrastructure boundary

HTTP provides no transport encryption. Administrator and counter keys, cookies and displayed data can be intercepted on an untrusted plaintext network. Use HTTP only on a deliberately trusted, isolated LAN. Otherwise expose Turnino through HTTPS, redirect or block HTTP, and prevent direct access to the plaintext backend.

Turnino is responsible for application authentication, authorization, CSRF protection and storing only credential hashes. The deployment owner is responsible for TLS termination, certificates, reverse-proxy policy, firewalls, VLANs, Wi-Fi isolation and routing. Plaintext exposure is an infrastructure configuration risk, not an application-authentication defect.

## Counter and Caller authentication

Each counter has a separate random token. Only its SHA-256 hash is stored. The Caller keeps the original token in that browser's `localStorage` and sends it as a Bearer credential when resolving its identity or making a call.

Revoking or rotating a counter invalidates the old token. A rejected counter call produces `401 Unauthorized`.

## Deliberately public endpoints

The following resources are intentionally public because they display waiting-room information:

- `/monitor`;
- `/counter-display` (a browser-local public counter ID selects the display, while call events remain public);
- `/about` (public project, license and source information);
- `/api/v1/events/calls`;
- the home page and static assets;
- `/healthz`.

They cannot create or modify calls. Do not put personal or confidential information in queue or counter names.

## Browser kiosk policies

Fullscreen and autoplay are browser controls, not Turnino authorization controls:

- `--kiosk` and `--edge-kiosk-type=fullscreen` control Edge presentation;
- `--autoplay-policy=no-user-gesture-required` allows unattended audio;
- `--force-device-scale-factor` is an optional Chromium rendering workaround.

The production page does not contain buttons to weaken or override browser policy. `/monitor?test=1` exposes interactive controls only for development.

## PowerShell execution warning

A script executed directly from `\\wsl.localhost` is opened through a UNC path and Windows may classify it as coming from a remote security zone. The confirmation is generated by PowerShell/Windows, before the Turnino launcher starts.

Preferred workaround:

```powershell
New-Item -ItemType Directory -Force C:\Turnino
$source = "\\wsl.localhost\<distro>\home\<user>\turnino\deploy\kiosk\windows\start-monitor.ps1"
Copy-Item `
  $source `
  "C:\Turnino\start-monitor.ps1"
Unblock-File "C:\Turnino\start-monitor.ps1"

powershell.exe -NoProfile -ExecutionPolicy Bypass `
  -File "C:\Turnino\start-monitor.ps1"
```

`Unblock-File` removes the downloaded-file zone marker when present. Copying to a stable Windows path also avoids treating every execution as access to a remote UNC location.

For a one-time repository test:

```powershell
$script = "\\wsl.localhost\<distro>\home\<user>\turnino\deploy\kiosk\windows\start-monitor.ps1"
powershell.exe -NoProfile -ExecutionPolicy Bypass `
  -File $script
```

Do not globally disable PowerShell execution policy merely to run Turnino.

## Troubleshooting by symptom

| Symptom | Layer to inspect |
|---|---|
| `ERR_CONNECTION_REFUSED` | process, listen address, WSL/Docker forwarding, firewall |
| HTTP `401` | administrator or counter Bearer token |
| HTTP `403` | browser administration session and CSRF token |
| HTTP `503` on admin API | initial administrator bootstrap |
| Monitor connection dot remains amber | SSE endpoint, reverse-proxy buffering, network interruption |
| Monitor updates but has no sound | Turnino sound setting and browser autoplay policy |
| Edge is not fullscreen | kiosk launcher arguments or Edge policy |
| PowerShell trust prompt | UNC/zone marker and execution policy |

Access keys and Bearer tokens must not be sent over an untrusted plaintext network. Use HTTPS through a reverse proxy when Turnino is reachable beyond a trusted LAN.
