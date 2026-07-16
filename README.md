# Thai Bus Watch 🚌

Track Bangkok buses live from your phone: a mobile web app with a real-time map,
plus Telegram alerts that ping you when your bus gets close to your stop.

Built on top of [thanawatnew/thai_bus_watch_go](https://github.com/thanawatnew/thai_bus_watch_go),
which looks up a bus's live GPS from Thailand's [Namtang](https://namtang.otp.go.th/)
public transport system and finds the nearest BMA traffic camera. The original
one-shot CLI still works (see below); this fork adds a web server, a phone-friendly
map UI, and a background watch/notification engine.

## Features

- 📍 **Routes near me** — uses your location to list nearby stops and every route passing them
- 🗺️ **Live map** — route line, stops, and every bus on the route updating every ~12 s
- 🚌 **Bus details** — speed, next stop, last update
- 🎥 **Live traffic camera in the app** — the nearest BMA camera streams right in the bus detail view (the server proxies bmatraffic.com's HTTP/session-bound frames over HTTPS so phones can show them)
- 🔔 **Proximity alerts** — drop a pin (e.g. your stop), pick a radius, get a Telegram push when the bus is that close, with an ETA
- 📡 **Telegram live pin** — a live-location marker in Telegram that moves with the bus
- 📸 **Camera pass-by snapshots** — when your watched bus passes within ~130 m of a traffic camera, the camera's photo is sent to your Telegram
- 🔎 **Local bus check** — a key-free detector inspired by [WIMB](https://github.com/thanawatnew/wimb) checks bus color and shape in each camera snapshot. Live GPS identifies the bus; 352×288 camera frames are not sharp enough to identify a plate reliably.
- 📱 **Installable PWA** — add to your iPhone home screen and it opens like a native app

## Setup (~10 minutes, free)

### 1. Create a Telegram bot

1. In Telegram, open [@BotFather](https://t.me/BotFather), send `/newbot`, pick any name and username.
2. Copy the **bot token** it gives you (looks like `1234567890:AA...`).

### 2. Deploy to Render (free tier)

1. Fork / push this repo to your own GitHub account.
2. Sign up at [render.com](https://render.com) with that GitHub account.
3. Click **New → Blueprint**, pick this repo — Render reads `render.yaml` automatically.
4. When prompted, paste your bot token as `TELEGRAM_BOT_TOKEN`.
5. Deploy. You'll get a URL like `https://thai-bus-watch.onrender.com`.

### 3. Connect your phone

1. Open your bot's chat in Telegram and send `/start` — the app replies "✅ Connected".
2. Open your Render URL in Safari on your iPhone.
3. Tap **Share → Add to Home Screen**. Done.

> **Note on the free tier:** Render idles the service after 15 minutes without
> traffic (first open afterwards takes ~30 s). While a watch is active the app
> keeps itself awake automatically, so alerts are not missed. Telegram messages
> arrive via webhook, so sending `/start` also wakes the service.

### Environment variables

| Variable | Required | Purpose |
|---|---|---|
| `TELEGRAM_BOT_TOKEN` | for notifications | Bot token from @BotFather |
| `TELEGRAM_CHAT_ID` | recommended | Pins your chat so the connection survives restarts — the bot tells you the value after `/start` |
| `SELF_URL` | auto on Render | Public URL for the Telegram webhook + keep-alive (Render sets `RENDER_EXTERNAL_URL` automatically) |

## Run locally instead

```sh
TELEGRAM_BOT_TOKEN=... go run .        # serves on http://localhost:8080
```

Without the token the map still works; only notifications are disabled.

## Original CLI mode

```sh
go run . -trip-id 7179 -bus 11-9253
```

Prints the bus position and the nearest BMA traffic camera once, as text + JSON.

## API

| Endpoint | Description |
|---|---|
| `GET /api/nearby?lat=&lon=` | Nearby stops with passing routes (trip IDs) |
| `GET /api/trip/{id}` | Route shape, stops, and live GPS of all buses |
| `GET /api/bus?trip=&bus=` | One bus + nearest BMA traffic camera |
| `GET /api/camera/{id}/frame` | Live JPEG frame from a BMA traffic camera (HTTPS proxy) |
| `POST /api/watch` | Start watching: `{tripId, busId, alert?: {lat, lon, radiusM, label}}` |
| `GET /api/watch` / `DELETE /api/watch/{id}` | List / stop watches |
| `GET /api/telegram/status` | Bot configured / chat connected |

Data sources: [Namtang API](https://namtang.otp.go.th) (OTP, Ministry of Transport)
for bus GPS and stops; [bmatraffic.com](http://www.bmatraffic.com) for traffic cameras.

## License

AGPL-3.0, same as the upstream project.
# Priority-pass admission control

Priority-pass mode is optional and disabled by default. Generate 50 ranked
passes (the first code has the highest priority):

```sh
./scripts/generate_priority_passes.sh 50 priority-passes.json
```

Enable it when starting the server:

```sh
PRIORITY_PASS_ENABLED=true \
PRIORITY_PASS_FILE=/path/to/priority-passes.json \
MAX_CONCURRENT_USERS=10 \
./buswatch
```

Set `PRIORITY_PASS_ENABLED=false` to turn the mode off. The maximum is dynamic
configuration through `MAX_CONCURRENT_USERS`; restart the service after changing
either setting. Sessions expire after two inactive minutes. When full, a
higher-ranked pass can replace the lowest-ranked active session.
