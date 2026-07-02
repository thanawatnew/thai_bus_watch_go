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
- 🤖 **Optional AI bus check** — with an `ANTHROPIC_API_KEY` set, Claude vision inspects that snapshot and says whether your bus is actually visible (cameras are 352×288, so it judges by bus type/route sign — plates aren't readable at that resolution)
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
| `ANTHROPIC_API_KEY` | optional | Enables the AI camera check (uses `claude-opus-4-8`; override with `VISION_MODEL`) |
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
