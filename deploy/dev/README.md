# BMA camera relay

The Oracle VM cannot reliably reach BMA Traffic's HTTP-only server. This
camera-only relay runs on the always-on Raspberry Pi, which can reach BMA, and
is published through Tailscale Funnel. It exposes only `/healthz` and validated
`/camera/{id}/frame` JPEG responses. It performs no polling: work happens only
when BUS287 requests a camera check. The user service is capped at 10% CPU,
64 MB of memory, and 32 tasks.

Build and start the user service:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -trimpath \
  -o /home/pi/bin/buswatch-camera-relay .
install -Dm644 deploy/dev/buswatch-camera-relay.service \
  /home/pi/.config/systemd/user/buswatch-camera-relay.service
systemctl --user daemon-reload
systemctl --user enable --now buswatch-camera-relay.service
sudo loginctl enable-linger pi
sudo tailscale funnel --bg --yes --https=8443 8187
```

Verify both paths:

```sh
curl -fsS http://127.0.0.1:8187/healthz
curl -fsS https://raspberrypi.tailec6f8a.ts.net:8443/healthz
```

Oracle sets `BMA_CAMERA_RELAY_URL` to the Funnel origin. Its public
same-origin `/api/camera/{id}/frame` endpoint fetches from the relay and checks
the frame again before returning it to the browser.

If the Pi or Funnel is unavailable, `/api/camera/healthz` returns unavailable
without affecting the main health check. The first dialog then offers
bus-tracking-only mode instead of blocking access to BUS287.

The BMA session is reused for 20 minutes. A camera that continues returning
valid frames stays active without repeating the index/PlayVideo sequence; a
blank frame invalidates that camera so the next request reopens it.
