# BMA camera relay

The Oracle VM cannot reliably reach BMA Traffic's HTTP-only server. This
camera-only relay runs on the development PC, which can reach BMA, and is
published through Tailscale Funnel. It exposes only `/healthz` and validated
`/camera/{id}/frame` JPEG responses.

Build and start the user service:

```sh
go build -o /home/thanawatnew/bin/buswatch-camera-relay .
install -Dm644 deploy/dev/buswatch-camera-relay.service \
  /home/thanawatnew/.config/systemd/user/buswatch-camera-relay.service
systemctl --user daemon-reload
systemctl --user enable --now buswatch-camera-relay.service
tailscale funnel --bg --yes 8187
```

Verify both paths:

```sh
curl -fsS http://127.0.0.1:8187/healthz
curl -fsS https://thanawat-ms7c37-kubuntu.tailec6f8a.ts.net/healthz
```

Oracle sets `BMA_CAMERA_RELAY_URL` to the Funnel origin. Its public
same-origin `/api/camera/{id}/frame` endpoint fetches from the relay and checks
the frame again before returning it to the browser.

The BMA session is reused for 20 minutes. A camera that continues returning
valid frames stays active without repeating the index/PlayVideo sequence; a
blank frame invalidates that camera so the next request reopens it.
