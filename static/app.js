/* Thai Bus Watch — mobile PWA frontend */
"use strict";

const BANGKOK = [13.7563, 100.5018];
const REFRESH_MS = 12000;

const state = {
  view: "home",          // home | trip | alert-pick
  tripId: null,
  trip: null,
  selectedBus: null,     // bus id string
  follow: false,
  alertDraft: null,      // {lat, lon, radiusM}
  tg: { configured: false, connected: false },
  refreshTimer: null,
  watchTimer: null,
  myPos: null,
};

/* ---------- map setup ---------- */
const map = L.map("map", { zoomControl: false, attributionControl: true })
  .setView(BANGKOK, 12);

L.tileLayer("https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png", {
  attribution: '&copy; OpenStreetMap &copy; CARTO',
  maxZoom: 19,
}).addTo(map);

const layers = {
  route: L.layerGroup().addTo(map),
  stops: L.layerGroup().addTo(map),
  buses: L.layerGroup().addTo(map),
  alert: L.layerGroup().addTo(map),
  me: L.layerGroup().addTo(map),
};
const busMarkers = {}; // busId -> marker

/* ---------- helpers ---------- */
const $ = (sel) => document.querySelector(sel);
const sheetContent = $("#sheet-content");

function toast(msg, ms = 3000) {
  const t = $("#toast");
  t.textContent = msg;
  t.classList.remove("hidden");
  clearTimeout(t._h);
  t._h = setTimeout(() => t.classList.add("hidden"), ms);
}

function esc(s) {
  return String(s ?? "").replace(/[&<>"']/g, (c) =>
    ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
}

async function api(path, opts) {
  const r = await fetch(path, opts);
  const body = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(body.error || `HTTP ${r.status}`);
  return body;
}

function setSheet(html, expand = true) {
  sheetContent.innerHTML = html;
  if (expand) $("#sheet").classList.remove("collapsed");
}

function getRecents() {
  try { return JSON.parse(localStorage.getItem("recents") || "[]"); } catch { return []; }
}

function addRecent(tripId, name, headsign) {
  const recents = getRecents().filter((r) => r.tripId !== tripId);
  recents.unshift({ tripId, name, headsign });
  localStorage.setItem("recents", JSON.stringify(recents.slice(0, 8)));
}

function geolocate() {
  return new Promise((resolve, reject) => {
    if (!navigator.geolocation) return reject(new Error("no geolocation"));
    navigator.geolocation.getCurrentPosition(
      (p) => resolve([p.coords.latitude, p.coords.longitude]),
      (e) => reject(e),
      { enableHighAccuracy: true, timeout: 12000, maximumAge: 30000 }
    );
  });
}

function showMe(pos) {
  state.myPos = pos;
  layers.me.clearLayers();
  L.circleMarker(pos, { radius: 7, color: "#fff", weight: 2, fillColor: "#2f6fed", fillOpacity: 1 })
    .addTo(layers.me);
}

/* ---------- telegram status ---------- */
async function refreshTelegram() {
  try {
    state.tg = await api("/api/telegram/status");
  } catch { /* server unreachable; keep last */ }
  const pill = $("#tg-pill");
  if (state.tg.connected) {
    pill.className = "pill pill-on";
    pill.textContent = "🔔 on";
  } else {
    pill.className = "pill pill-off";
    pill.textContent = state.tg.configured ? "🔔 connect" : "🔔 off";
  }
}

function telegramSetupHTML() {
  if (state.tg.connected) return "";
  if (!state.tg.configured) {
    return `<div class="tg-setup">⚠️ <b>Notifications are off.</b> The server has no
      <code>TELEGRAM_BOT_TOKEN</code>. You can still watch buses on the map.</div>`;
  }
  const bot = state.tg.botUsername
    ? `<a href="https://t.me/${esc(state.tg.botUsername)}" style="color:#7fb8ff">@${esc(state.tg.botUsername)}</a>`
    : "your bot";
  return `<div class="tg-setup">🔔 <b>One step left:</b> open ${bot} in Telegram and send
    <code>/start</code>. Alerts will then reach your iPhone as push notifications.</div>`;
}

/* ---------- home view ---------- */
function renderHome() {
  state.view = "home";
  stopRefresh();
  const recents = getRecents();
  setSheet(`
    ${telegramSetupHTML()}
    <button class="btn" id="btn-near">📍 Find routes near me</button>
    <h2>Open a route by trip ID</h2>
    <div class="input-row">
      <input type="text" id="trip-input" inputmode="numeric" placeholder="Trip ID, e.g. 7179">
      <button class="btn" id="btn-open-trip">Go</button>
    </div>
    <small>Trip IDs come from <a href="https://namtang.otp.go.th" style="color:#7fb8ff">namtang.otp.go.th</a> — or just use “near me”.</small>
    ${recents.length ? `<h2>Recent routes</h2><div class="chips">` +
      recents.map((r) => `<button class="route-chip" style="background:#2f6fed"
        data-trip="${esc(r.tripId)}">${esc(r.name)} → ${esc(r.headsign)}</button>`).join("") + `</div>` : ""}
    <div id="nearby-out"></div>
  `);

  $("#btn-near").onclick = loadNearby;
  $("#btn-open-trip").onclick = () => {
    const v = $("#trip-input").value.trim();
    if (v) openTrip(v);
  };
  $("#trip-input").addEventListener("keydown", (e) => {
    if (e.key === "Enter") $("#btn-open-trip").click();
  });
  sheetContent.querySelectorAll(".route-chip[data-trip]").forEach((b) => {
    b.onclick = () => openTrip(b.dataset.trip);
  });
}

async function loadNearby() {
  const out = $("#nearby-out");
  out.innerHTML = `<h2>Nearby stops</h2><small>Locating you…</small>`;
  let pos;
  try {
    pos = await geolocate();
  } catch {
    out.innerHTML = `<h2>Nearby stops</h2><small>⚠️ Couldn't get your location. Allow location access in Settings → Safari, or enter a trip ID above.</small>`;
    return;
  }
  showMe(pos);
  map.setView(pos, 16);
  out.innerHTML = `<h2>Nearby stops</h2><small>Loading routes…</small>`;
  try {
    const stops = await api(`/api/nearby?lat=${pos[0]}&lon=${pos[1]}`);
    if (!stops.length) {
      out.innerHTML = `<h2>Nearby stops</h2><small>No stops found near you.</small>`;
      return;
    }
    out.innerHTML = `<h2>Nearby stops</h2>` + stops.slice(0, 10).map((s) => `
      <div class="stop-card">
        <div class="stop-name">🚏 ${esc(s.name)}</div>
        <div class="chips">${(s.passingTrips || []).map((t) => `
          <button class="route-chip ${t.hasGps ? "" : "no-gps"}"
            style="background:#${esc(t.color || "555")}"
            data-trip="${t.tripId}" ${t.hasGps ? "" : "disabled"}
            title="${esc(t.routeLongName)}">${esc(t.name)}</button>`).join("")}
        </div>
      </div>`).join("");
    out.querySelectorAll(".route-chip[data-trip]").forEach((b) => {
      b.onclick = () => openTrip(b.dataset.trip);
    });
  } catch (e) {
    out.innerHTML = `<h2>Nearby stops</h2><small>⚠️ ${esc(e.message)}</small>`;
  }
}

/* ---------- trip view ---------- */
async function openTrip(tripId) {
  state.view = "trip";
  state.tripId = String(tripId);
  state.selectedBus = null;
  state.follow = false;
  setSheet(`<h2>Loading trip ${esc(tripId)}…</h2>`);
  try {
    const trip = await api(`/api/trip/${encodeURIComponent(tripId)}`);
    state.trip = trip;
    addRecent(state.tripId, trip.routeShortName, trip.tripHeadsign);
    drawTrip(trip, true);
    renderTripSheet();
    startRefresh();
  } catch (e) {
    setSheet(`<h2>⚠️ ${esc(e.message)}</h2><button class="btn btn-ghost" onclick="renderHome()">Back</button>`);
  }
}

function drawTrip(trip, fit) {
  layers.route.clearLayers();
  layers.stops.clearLayers();
  const color = "#" + (trip.routeColor || "2f6fed");

  if (trip.shapeGeom && trip.shapeGeom.length) {
    const line = L.polyline(trip.shapeGeom.map((p) => [p.lat, p.lon]),
      { color, weight: 4, opacity: 0.8 }).addTo(layers.route);
    if (fit) map.fitBounds(line.getBounds(), { padding: [40, 40] });
  }

  (trip.stopList || []).forEach((s) => {
    L.circleMarker([s.location.lat, s.location.lon], {
      radius: 4, color: "#fff", weight: 1.5, fillColor: color, fillOpacity: 0.9,
    }).bindPopup(esc(s.stopName)).addTo(layers.stops);
  });

  updateBuses(trip);
}

function updateBuses(trip) {
  const seen = new Set();
  (trip.gpsList || []).forEach((b) => {
    const lat = Number(b.snapped_lat) || Number(b.lat);
    const lon = Number(b.snapped_lon) || Number(b.lon);
    if (!lat || !lon) return;
    seen.add(b.id);
    const plate = b.id.split(" ")[0];
    const sel = state.selectedBus === b.id;
    const html = `<div class="bus-marker ${sel ? "sel" : ""}">
      <div class="bus-emoji">🚌</div><div class="bus-label">${esc(plate)}</div></div>`;
    if (busMarkers[b.id]) {
      busMarkers[b.id].setLatLng([lat, lon]);
      busMarkers[b.id].setIcon(L.divIcon({ html, className: "", iconSize: [30, 40], iconAnchor: [15, 20] }));
    } else {
      busMarkers[b.id] = L.marker([lat, lon], {
        icon: L.divIcon({ html, className: "", iconSize: [30, 40], iconAnchor: [15, 20] }),
        zIndexOffset: 500,
      }).on("click", () => selectBus(b.id)).addTo(layers.buses);
    }
    if (sel && state.follow) map.panTo([lat, lon]);
  });
  Object.keys(busMarkers).forEach((id) => {
    if (!seen.has(id)) {
      layers.buses.removeLayer(busMarkers[id]);
      delete busMarkers[id];
    }
  });
}

function fmtAgo(unixSec) {
  if (!unixSec) return "-";
  const s = Math.max(0, Math.round(Date.now() / 1000 - unixSec));
  return s < 90 ? `${s}s ago` : `${Math.round(s / 60)}m ago`;
}

function renderTripSheet() {
  const t = state.trip;
  if (!t) return;
  const buses = t.gpsList || [];
  setSheet(`
    <div style="display:flex;align-items:center;gap:10px">
      <span class="route-chip" style="background:#${esc(t.routeColor || "555")};cursor:default">${esc(t.routeShortName)}</span>
      <div style="flex:1;min-width:0">
        <div style="font-weight:600">→ ${esc(t.tripHeadsign)}</div>
        <small>${esc(t.routeLongName)}</small>
      </div>
      <button class="btn btn-ghost" style="width:auto;padding:8px 12px" id="btn-back">‹ Back</button>
    </div>
    ${telegramSetupHTML()}
    <h2>${buses.length} bus${buses.length === 1 ? "" : "es"} live — tap one to track</h2>
    <div id="bus-list">
      ${buses.length ? buses.map((b) => busRowHTML(b)).join("")
        : `<small>No buses reporting GPS on this route right now. Try the opposite direction (routes usually have two trip IDs) or come back later.</small>`}
    </div>
    <div id="bus-detail"></div>
  `);
  $("#btn-back").onclick = () => { clearTripLayers(); renderHome(); };
  bindBusRows();
}

function busRowHTML(b) {
  const plate = b.id.split(" ")[0];
  const speed = Math.round(Number(b.speed) || 0);
  const sel = state.selectedBus === b.id;
  return `<div class="bus-row ${sel ? "selected" : ""}" data-bus="${esc(b.id)}">
    <div style="font-size:20px">🚌</div>
    <div class="grow">
      <div class="plate">${esc(plate)}</div>
      <div class="sub">→ ${esc(b.next_stop_name || "?")} · updated ${fmtAgo(b.received)}</div>
    </div>
    <div class="speed-badge ${speed < 3 ? "stopped" : ""}">${speed < 3 ? "stopped" : speed + " km/h"}</div>
  </div>`;
}

function bindBusRows() {
  sheetContent.querySelectorAll(".bus-row").forEach((row) => {
    row.onclick = () => selectBus(row.dataset.bus);
  });
}

async function selectBus(busId) {
  state.selectedBus = busId;
  state.follow = true;
  updateBuses(state.trip);
  const m = busMarkers[busId];
  if (m) map.setView(m.getLatLng(), Math.max(map.getZoom(), 15));

  sheetContent.querySelectorAll(".bus-row").forEach((r) =>
    r.classList.toggle("selected", r.dataset.bus === busId));

  const detail = $("#bus-detail");
  if (!detail) return;
  const plate = busId.split(" ")[0];
  detail.innerHTML = `<h2>Bus ${esc(plate)}</h2><small>Loading details…</small>`;

  let camHTML = "";
  try {
    const d = await api(`/api/bus?trip=${encodeURIComponent(state.tripId)}&bus=${encodeURIComponent(plate)}`);
    if (d.nearestCamera) {
      camHTML = `
        <dt>🎥 Camera</dt><dd>${esc(d.nearestCamera.name_th || d.nearestCamera.name_en || d.nearestCamera.id)}
          <small>(${Math.round(d.cameraDistanceM)} m away)</small><br>
          <a href="${esc(d.nearestCamera.feed_url)}" target="_blank" style="color:#7fb8ff">Open traffic camera feed ↗</a></dd>`;
    }
  } catch { /* camera info is best-effort */ }

  const b = (state.trip.gpsList || []).find((x) => x.id === busId) || {};
  detail.innerHTML = `
    <h2>Bus ${esc(plate)}</h2>
    <dl class="detail-grid">
      <dt>Next stop</dt><dd>${esc(b.next_stop_name || "?")} <small>(${Math.round(Number(b.distance_to_next_stop) || 0)} m)</small></dd>
      <dt>Speed</dt><dd>${Math.round(Number(b.speed) || 0)} km/h</dd>
      <dt>Updated</dt><dd>${fmtAgo(b.received)}</dd>
      ${camHTML}
    </dl>
    <button class="btn" id="btn-alert" ${state.tg.connected ? "" : "disabled"}>🔔 Alert me when it reaches a place</button>
    ${state.tg.connected ? "" : `<small>Connect Telegram (see above) to enable alerts.</small>`}
    <div class="btn-row">
      <button class="btn btn-ghost" id="btn-live" ${state.tg.connected ? "" : "disabled"}>📍 Live pin in Telegram</button>
    </div>
  `;
  $("#btn-alert").onclick = startAlertFlow;
  $("#btn-live").onclick = () => createWatch(null);
}

/* ---------- alert flow ---------- */
function startAlertFlow() {
  state.view = "alert-pick";
  $("#alert-overlay").classList.remove("hidden");
  $("#sheet").classList.add("collapsed");
  map.once("click", onAlertMapClick);
}

function endAlertFlow() {
  $("#alert-overlay").classList.add("hidden");
  map.off("click", onAlertMapClick);
  state.view = "trip";
}

function onAlertMapClick(e) {
  placeAlertDraft(e.latlng.lat, e.latlng.lng);
}

function placeAlertDraft(lat, lon) {
  endAlertFlow();
  state.alertDraft = { lat, lon, radiusM: 500 };
  drawAlertDraft();
  renderAlertConfirm();
}

function drawAlertDraft() {
  layers.alert.clearLayers();
  const a = state.alertDraft;
  if (!a) return;
  L.circle([a.lat, a.lon], { radius: a.radiusM, color: "#e8b25a", weight: 2, fillOpacity: 0.12 }).addTo(layers.alert);
  L.marker([a.lat, a.lon], {
    icon: L.divIcon({ html: `<div style="font-size:26px">📍</div>`, className: "", iconSize: [26, 26], iconAnchor: [13, 26] }),
  }).addTo(layers.alert);
  map.panTo([a.lat, a.lon]);
}

function renderAlertConfirm() {
  const plate = (state.selectedBus || "").split(" ")[0];
  setSheet(`
    <h2>🔔 Alert for bus ${esc(plate)}</h2>
    <p style="font-size:14px;margin:4px 0">Telegram will notify you when the bus is within:</p>
    <div class="radius-chips" id="radius-chips">
      <button data-r="300">300 m</button>
      <button data-r="500" class="sel">500 m</button>
      <button data-r="1000">1 km</button>
      <button data-r="2000">2 km</button>
    </div>
    <input type="text" id="alert-label" placeholder="Label (optional), e.g. My stop">
    <div class="btn-row">
      <button class="btn" id="btn-alert-go">Start alert</button>
      <button class="btn btn-ghost" id="btn-alert-abort">Cancel</button>
    </div>
  `);
  $("#radius-chips").querySelectorAll("button").forEach((b) => {
    b.onclick = () => {
      $("#radius-chips").querySelectorAll("button").forEach((x) => x.classList.remove("sel"));
      b.classList.add("sel");
      state.alertDraft.radiusM = Number(b.dataset.r);
      drawAlertDraft();
    };
  });
  $("#btn-alert-go").onclick = () => {
    state.alertDraft.label = $("#alert-label").value.trim();
    createWatch(state.alertDraft);
  };
  $("#btn-alert-abort").onclick = () => {
    state.alertDraft = null;
    layers.alert.clearLayers();
    renderTripSheet();
    selectBus(state.selectedBus);
  };
}

async function createWatch(alert) {
  const plate = (state.selectedBus || "").split(" ")[0];
  try {
    await api("/api/watch", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ tripId: state.tripId, busId: plate, alert }),
    });
    toast(alert ? "🔔 Alert set! Telegram will ping you." : "📍 Live pin started in Telegram");
    state.alertDraft = null;
    layers.alert.clearLayers();
    refreshWatches();
    renderTripSheet();
    if (state.selectedBus) selectBus(state.selectedBus);
  } catch (e) {
    toast("⚠️ " + e.message, 5000);
  }
}

/* ---------- active watches bar ---------- */
async function refreshWatches() {
  let list = [];
  try { list = await api("/api/watch"); } catch { return; }
  const bar = $("#watchbar");
  bar.innerHTML = list.filter((w) => w.status === "active").map((w) => `
    <div class="watch-chip">
      <span>👀</span>
      <span class="grow">${esc(w.routeName)} · bus ${esc(w.busId)}${w.alert ? " · 🔔 " + esc(w.alert.label || "pin") : ""}</span>
      <button data-id="${esc(w.id)}" title="Stop watching">✕</button>
    </div>`).join("");
  bar.querySelectorAll("button[data-id]").forEach((b) => {
    b.onclick = async () => {
      try { await api(`/api/watch/${b.dataset.id}`, { method: "DELETE" }); } catch {}
      refreshWatches();
    };
  });
}

/* ---------- refresh loop ---------- */
function startRefresh() {
  stopRefresh();
  state.refreshTimer = setInterval(async () => {
    if (state.view !== "trip" || document.hidden) return;
    try {
      const trip = await api(`/api/trip/${encodeURIComponent(state.tripId)}`);
      state.trip = trip;
      updateBuses(trip);
      const list = $("#bus-list");
      if (list && !state.selectedBus) {
        list.innerHTML = (trip.gpsList || []).map((b) => busRowHTML(b)).join("") ||
          `<small>No buses reporting GPS right now.</small>`;
        bindBusRows();
      }
    } catch { /* transient */ }
  }, REFRESH_MS);
}

function stopRefresh() {
  if (state.refreshTimer) clearInterval(state.refreshTimer);
  state.refreshTimer = null;
}

function clearTripLayers() {
  layers.route.clearLayers();
  layers.stops.clearLayers();
  layers.buses.clearLayers();
  layers.alert.clearLayers();
  Object.keys(busMarkers).forEach((k) => delete busMarkers[k]);
  stopRefresh();
}

/* ---------- wire up ---------- */
$("#btn-home").onclick = () => { clearTripLayers(); renderHome(); };
$("#sheet-handle").onclick = () => $("#sheet").classList.toggle("collapsed");
$("#alert-cancel").onclick = () => { endAlertFlow(); renderTripSheet(); selectBus(state.selectedBus); };
$("#alert-use-me").onclick = async () => {
  try {
    const pos = await geolocate();
    showMe(pos);
    placeAlertDraft(pos[0], pos[1]);
  } catch {
    toast("⚠️ Couldn't get your location");
  }
};

if ("serviceWorker" in navigator) {
  navigator.serviceWorker.register("sw.js").catch(() => {});
}

refreshTelegram();
setInterval(refreshTelegram, 20000);
refreshWatches();
state.watchTimer = setInterval(refreshWatches, 20000);
renderHome();
