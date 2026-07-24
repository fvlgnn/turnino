#!/bin/sh
set -eu

monitor_url="${TURNINO_MONITOR_URL:-http://localhost:8080/monitor}"
monitor_language="${TURNINO_MONITOR_LANGUAGE:-}"
kiosk_scale="${TURNINO_KIOSK_SCALE:-1}"
restart_delay="${TURNINO_KIOSK_RESTART_DELAY:-3}"

case "$monitor_language" in
    "") ;;
    en|it)
        case "$monitor_url" in
            *\?*) monitor_url="${monitor_url}&lang=${monitor_language}" ;;
            *) monitor_url="${monitor_url}?lang=${monitor_language}" ;;
        esac
        ;;
    *)
        echo "TURNINO_MONITOR_LANGUAGE must be empty, en or it." >&2
        exit 1
        ;;
esac

if [ -n "${TURNINO_BROWSER:-}" ]; then
    browser="$TURNINO_BROWSER"
elif command -v chromium >/dev/null 2>&1; then
    browser="chromium"
elif command -v chromium-browser >/dev/null 2>&1; then
    browser="chromium-browser"
else
    echo "Chromium was not found. Set TURNINO_BROWSER to its executable path." >&2
    exit 1
fi

echo "Turnino kiosk URL: $monitor_url"
echo "Turnino kiosk scale factor: $kiosk_scale"

while true; do
    "$browser" \
        --kiosk \
        --noerrdialogs \
        --disable-infobars \
        --no-first-run \
        --start-maximized \
        --disable-session-crashed-bubble \
        --autoplay-policy=no-user-gesture-required \
        --force-device-scale-factor="$kiosk_scale" \
        "$monitor_url" || true
    sleep "$restart_delay"
done
