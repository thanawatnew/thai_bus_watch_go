/* Thai Bus Watch — mobile PWA frontend */
"use strict";

const BANGKOK = [13.7563, 100.5018];
const REFRESH_MS = 5000;
const APP_VERSION = "0.5.9";
const BMA_PREFLIGHT_KEY = "bmaCameraPreflightV1";
const I18N = {
  en: { step:"Step", open:"Open", hide:"Hide", location:"Choose a location", locationHelp:"Use your location or tap your position on the map.", stop:"Choose a nearby stop", stopHelp:"Tap a stop to see its live routes and arrivals.", route:"Choose a bus route", routeHelp:"Tap the route you want to follow.", routeStop:"Choose a route stop", routeStopHelp:"Tap the stop where you want to meet the bus.", bus:"Choose a live bus", busHelp:"Tap a bus below to open its live details.", view:"View bus and camera", viewHelp:"Review the live bus details, then open the available traffic camera.", reset:"Start over from Step 1 and run the BMA camera test again? Your recent routes will be kept.", preflightTitle:"Test BMA camera access", iphoneTitle:"iPhone users: Firefox is recommended.", iphoneText:"Safari may not open BMA Traffic's external HTTP-only camera page reliably. Open Bus-287 in Firefox before running this test.", preflightIntro:"BMA Traffic is a separate, HTTP-only website. Its availability and content are controlled by BMA Traffic, not Bus-287.", preflightStep1:"1. Open the test: tap the blue BMA camera button below.", preflightStep2:"2. Allow the external page only if you accept opening BMA's HTTP website.", preflightStep3:"3. Check whether the BMA camera content appears.", preflightStep4:"4. Return to this Bus-287 browser tab.", preflightStep5:"5. Report Yes or No below to enter Bus Watch.", openBmaTest:"🎥 Open BMA camera test ↗", bmaWorkedQuestion:"Did the BMA camera page open correctly?", bmaYes:"Yes, camera worked — continue", bmaNo:"No — continue with bus tracking only", preflightDisclaimer:"By continuing, you understand that external camera access may be insecure, unavailable, or behave differently in each browser." },
  th: { step:"ขั้นตอน", open:"เปิด", hide:"ซ่อน", location:"เลือกตำแหน่ง", locationHelp:"ใช้ตำแหน่งปัจจุบันหรือแตะตำแหน่งบนแผนที่", stop:"เลือกป้ายใกล้เคียง", stopHelp:"แตะป้ายเพื่อดูสายรถและเวลาถึงแบบสด", route:"เลือกสายรถโดยสาร", routeHelp:"แตะสายรถที่ต้องการติดตาม", routeStop:"เลือกป้ายในเส้นทาง", routeStopHelp:"แตะป้ายที่คุณต้องการขึ้นรถ", bus:"เลือกรถที่กำลังวิ่ง", busHelp:"แตะรถด้านล่างเพื่อดูรายละเอียดสด", view:"ดูรถและกล้อง", viewHelp:"ดูรายละเอียดรถ แล้วเปิดกล้องจราจรที่มีอยู่", reset:"เริ่มใหม่จากขั้นตอนที่ 1 และทดสอบกล้อง BMA อีกครั้งหรือไม่? รายการเส้นทางล่าสุดจะยังอยู่", preflightTitle:"ทดสอบการเข้าถึงกล้อง BMA", iphoneTitle:"ผู้ใช้ iPhone: แนะนำ Firefox", iphoneText:"Safari อาจเปิดหน้ากล้อง HTTP ของ BMA ได้ไม่สมบูรณ์ กรุณาใช้ Firefox", preflightIntro:"BMA Traffic เป็นเว็บไซต์ HTTP ภายนอก ซึ่งไม่ได้ควบคุมโดย Bus-287", preflightStep1:"1. แตะปุ่มสีน้ำเงินเพื่อเปิดหน้าทดสอบกล้อง BMA", preflightStep2:"2. อนุญาตหน้าเว็บภายนอกเมื่อคุณยอมรับการเปิดเว็บไซต์ HTTP", preflightStep3:"3. ตรวจสอบว่าภาพจากกล้อง BMA แสดงหรือไม่", preflightStep4:"4. กลับมายังแท็บ Bus-287", preflightStep5:"5. ตอบว่าใช่หรือไม่ใช่เพื่อเข้าใช้งาน", openBmaTest:"🎥 เปิดหน้าทดสอบกล้อง BMA ↗", bmaWorkedQuestion:"หน้ากล้อง BMA เปิดได้ถูกต้องหรือไม่?", bmaYes:"ใช่ กล้องใช้งานได้ — ต่อไป", bmaNo:"ไม่ — ใช้เฉพาะการติดตามรถ", preflightDisclaimer:"เมื่อดำเนินการต่อ คุณเข้าใจว่ากล้องภายนอกอาจไม่ปลอดภัยหรือไม่พร้อมใช้งาน" }
};
Object.assign(I18N.en, {
  preflightStep1: "Open the test by tapping the blue BMA camera button below.",
  preflightStep2: "If your browser warns you, confirm that you want to open the BMA Traffic HTTP website.",
  preflightStep3: "Check whether the BMA camera content appears.", preflightStep4: "Return to this Bus-287 browser tab.",
  preflightStep5: "Report Yes or No below to enter Bus Watch.",
  resetLabel: "↻ Reset", twoNearest: "Two nearest live buses", allBuses: "All buses",
  noGps: "No buses reporting GPS right now.", noApproaching: "No approaching bus estimate is available.",
  noLiveTrip: "No live buses on this trip.", loadingArrival: "Loading arrival estimate…",
  showArrivals: "Show arrivals ›", selectStopBuses: "Select this bus stop to see buses.",
  backToStop: "‹ Back to selected stop",
  homeTitle: "1. Select your nearest location first",
  homeHelp: "Use your location or tap your position on the map. Then choose a bus stop and bus.",
  mobileTip: "Android/iPhone tip: tap or swipe the bar above this panel to hide it, then tap the bar again to reopen it.",
  mapTapHint: "👆 Or tap anywhere on the map to find the nearest bus stops there.",
  findStops: "📍 Find nearest bus stops", clearSaved: "🧹 Clear saved data & cache",
  optionalRoute: "Optional: open a route by trip ID", tripPlaceholder: "Trip ID, e.g. 7179", go: "Go",
  tripHelp: "Only use this if you already know a trip ID from", recentRoutes: "Recent routes",
  civic: "Independent experimental civic-tech project. Not affiliated with or endorsed by Bangkok Metropolitan Administration. Camera content remains on the official BMA Traffic service.",
  nearestCamera: "Nearest traffic camera", upcomingCamera: "Upcoming traffic camera", fromBus: "m from bus",
  onRoute: "on route", nearRoute: "near route", cameraId: "Current camera ID", openCamera: "Open camera on BMA ↗",
  previousCamera: "‹ Previous camera", nextCamera: "Next camera ›", showPins: "🗺️ Show bus + camera pins",
  cameraAvailable: "🎥 Traffic camera available", cameraAvailableHelp: "Tap to view the nearest camera for this bus ↓",
  cameraNote: "BMA Traffic is a separate website. On first use, you must open and allow the camera feed there yourself. If it does not start, allow the BMA page, return here, and open it again.",
  alertPlace: "🔔 Alert me when it reaches a place",
  bmaHelpTitle: "Camera did not open?", bmaHelpText: "Open the BMA camera page directly first, allow or start the camera feed there, then return to Bus-287 and run the test again.",
  openBmaAgain: "Open BMA camera page again ↗", continueWithoutCamera: "I understand — continue without cameras",
  usageTitle: "How to use Bus-287", usageStep1: "Choose your location using GPS or by tapping the map.",
  usageStep2: "Choose a nearby bus stop.", usageStep3: "Choose the bus route you want to follow.",
  usageStep4: "Choose a live bus from the list.", usageStep5: "View live bus details and open an available BMA camera.",
  usagePanelTip: "Use the labeled bar at the bottom to hide or reopen the panel while viewing the map.",
});
Object.assign(I18N.th, {
  preflightDisclaimer: "โปรดทราบว่า BMA Traffic เป็นเว็บไซต์ HTTP ภายนอก กล้องอาจไม่พร้อมใช้งาน หรืออาจทำงานแตกต่างกันในแต่ละเบราว์เซอร์",
  preflightStep1: "แตะปุ่มสีน้ำเงินเพื่อเปิดหน้าทดสอบกล้อง BMA",
  preflightStep2: "เมื่อเบราว์เซอร์แจ้งเตือน ให้ยืนยันการเปิดเว็บไซต์ BMA Traffic (HTTP)",
  preflightStep3: "ตรวจสอบว่าภาพจากกล้อง BMA แสดงหรือไม่", preflightStep4: "กลับมายังแท็บ Bus-287",
  preflightStep5: "ตอบว่าใช่หรือไม่ใช่เพื่อเข้าใช้งาน",
  resetLabel: "↻ รีเซ็ต", twoNearest: "รถที่กำลังวิ่งใกล้ที่สุด 2 คัน", allBuses: "รถทั้งหมด",
  noGps: "ขณะนี้ไม่มีรถส่งข้อมูล GPS", noApproaching: "ไม่มีข้อมูลประมาณเวลาของรถที่กำลังเข้าใกล้",
  noLiveTrip: "ไม่มีรถที่กำลังวิ่งในเที่ยวนี้", loadingArrival: "กำลังโหลดเวลาถึงโดยประมาณ…",
  showArrivals: "ดูรถที่จะมาถึง ›", selectStopBuses: "เลือกป้ายนี้เพื่อดูรถโดยสาร",
  backToStop: "‹ กลับไปยังป้ายที่เลือก",
  homeTitle: "1. เลือกตำแหน่งที่ใกล้คุณที่สุดก่อน",
  homeHelp: "ใช้ตำแหน่งปัจจุบันหรือแตะตำแหน่งของคุณบนแผนที่ จากนั้นเลือกป้ายและรถโดยสาร",
  mobileTip: "คำแนะนำ Android/iPhone: แตะหรือปัดแถบด้านบนแผงนี้เพื่อซ่อน แล้วแตะแถบอีกครั้งเพื่อเปิด",
  mapTapHint: "👆 หรือแตะตำแหน่งใดก็ได้บนแผนที่เพื่อค้นหาป้ายใกล้เคียง",
  findStops: "📍 ค้นหาป้ายรถโดยสารใกล้เคียง", clearSaved: "🧹 ล้างข้อมูลที่บันทึกและแคช",
  optionalRoute: "ตัวเลือกเพิ่มเติม: เปิดเส้นทางด้วยรหัสเที่ยวรถ", tripPlaceholder: "รหัสเที่ยวรถ เช่น 7179", go: "ไป",
  tripHelp: "ใช้ตัวเลือกนี้เมื่อคุณทราบรหัสเที่ยวรถจาก", recentRoutes: "เส้นทางล่าสุด",
  civic: "โครงการทดลองเทคโนโลยีเพื่อสังคมอิสระ ไม่ได้เป็นส่วนหนึ่งหรือได้รับการรับรองจากกรุงเทพมหานคร เนื้อหากล้องยังคงอยู่บนบริการ BMA Traffic อย่างเป็นทางการ",
  nearestCamera: "กล้องจราจรที่ใกล้ที่สุด", upcomingCamera: "กล้องจราจรข้างหน้า", fromBus: "ม. จากรถ",
  onRoute: "อยู่บนเส้นทาง", nearRoute: "อยู่ใกล้เส้นทาง", cameraId: "รหัสกล้องปัจจุบัน", openCamera: "เปิดกล้องบน BMA ↗",
  previousCamera: "‹ กล้องก่อนหน้า", nextCamera: "กล้องถัดไป ›", showPins: "🗺️ แสดงตำแหน่งรถและกล้อง",
  cameraAvailable: "🎥 มีกล้องจราจร", cameraAvailableHelp: "แตะเพื่อดูกล้องที่ใกล้รถคันนี้ที่สุด ↓",
  cameraNote: "BMA Traffic เป็นเว็บไซต์แยกต่างหาก เมื่อใช้งานครั้งแรก คุณต้องเปิดและอนุญาตฟีดกล้องบนเว็บไซต์นั้นด้วยตนเอง หากกล้องไม่เริ่มทำงาน ให้อนุญาตหน้า BMA จากนั้นกลับมาที่นี่แล้วเปิดอีกครั้ง",
  alertPlace: "🔔 แจ้งเตือนเมื่อรถถึงสถานที่",
  bmaHelpTitle: "เปิดกล้องไม่ได้ใช่หรือไม่?", bmaHelpText: "ให้เปิดหน้ากล้อง BMA โดยตรงก่อน อนุญาตหรือเริ่มฟีดกล้องบนเว็บไซต์นั้น จากนั้นกลับมายัง Bus-287 แล้วทดสอบอีกครั้ง",
  openBmaAgain: "เปิดหน้ากล้อง BMA อีกครั้ง ↗", continueWithoutCamera: "เข้าใจแล้ว — ใช้งานต่อโดยไม่ใช้กล้อง",
  usageTitle: "วิธีใช้งาน Bus-287", usageStep1: "เลือกตำแหน่งด้วย GPS หรือแตะบนแผนที่",
  usageStep2: "เลือกป้ายรถโดยสารใกล้เคียง", usageStep3: "เลือกสายรถที่ต้องการติดตาม",
  usageStep4: "เลือกรถที่กำลังวิ่งจากรายการ", usageStep5: "ดูรายละเอียดรถแบบสดและเปิดกล้อง BMA ที่มีอยู่",
  usagePanelTip: "ใช้แถบที่มีป้ายกำกับด้านล่างเพื่อซ่อนหรือเปิดแผงขณะดูแผนที่",
});
let currentLang = (() => { try { return localStorage.getItem("buswatchLanguage") || (navigator.language?.startsWith("th") ? "th" : "en"); } catch { return "en"; } })();
const t = (key) => I18N[currentLang]?.[key] || I18N.en[key] || key;
function applyLanguage() {
  document.documentElement.lang = currentLang;
  $("#language-select").value = currentLang;
  document.querySelectorAll("[data-i18n]").forEach((element) => { element.textContent = t(element.dataset.i18n); });
}

const state = {
  view: "home",          // home | trip | alert-pick
  tripId: null,
  trip: null,
  selectedBus: null,     // bus id string
  selectedStop: null,
  visibleBusIds: null,
  busMotion: {},
  selectionVersion: 0,  // invalidates slower responses from earlier clicks
  activeCameraBus: null,
  activeCameraId: null,
  pendingCameraId: null,
  cameraHandoffTimer: null,
  cameraIndexOffset: 0,
  follow: false,
  alertDraft: null,      // {lat, lon, radiusM}
  tg: { configured: false, connected: false },
  access: { enabled: false, authorized: true, active: 0, maxUsers: 0, rank: 0 },
  guideStep: { number: 1, label: "Choose a location" },
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
  camera: L.layerGroup().addTo(map),
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

function setGuideStep(number, label) {
  state.guideStep = { number, label };
  updateSheetHandle();
}

function revealSheetTarget(target, delay = 80) {
  $("#sheet").classList.remove("collapsed");
  setTimeout(() => {
    const element = typeof target === "string" ? document.querySelector(target) : target;
    element?.scrollIntoView({ behavior: "smooth", block: "start" });
  }, delay);
}

function guideBanner(number, title, instruction) {
  return `<div class="guide-banner" id="guide-step-${number}">
    <b>${t("step")} ${number}/5 · ${esc(title)}</b><span>${esc(instruction)}</span>
  </div>`;
}

function getRecents() {
  try { return JSON.parse(localStorage.getItem("recents") || "[]"); } catch { return []; }
}

function addRecent(tripId, name, headsign) {
  const recents = getRecents().filter((r) => r.tripId !== tripId);
  recents.unshift({ tripId, name, headsign });
  localStorage.setItem("recents", JSON.stringify(recents.slice(0, 8)));
}

function getLastStopSelection() {
  try { return JSON.parse(localStorage.getItem("lastStopSelection") || "null"); } catch { return null; }
}

function rememberStopSelection(stop) {
  localStorage.setItem("lastStopSelection", JSON.stringify({ tripId: state.tripId, stopId: stop.stopId }));
}

async function clearAppCache() {
  localStorage.removeItem("recents");
  localStorage.removeItem("lastStopSelection");
  localStorage.removeItem("bmaCameraNoticeSeen");
  localStorage.removeItem(BMA_PREFLIGHT_KEY);
  if ("caches" in window) {
    await Promise.all((await caches.keys()).map((key) => caches.delete(key)));
  }
  if ("serviceWorker" in navigator) {
    await Promise.all((await navigator.serviceWorker.getRegistrations()).map((r) => r.unregister()));
  }
  location.reload();
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
    pill.classList.remove("hidden");
    pill.className = "pill pill-on";
    pill.textContent = "🔔 on";
  } else {
    pill.className = "pill pill-off hidden";
    pill.textContent = "";
  }
}

function telegramSetupHTML() {
  if (state.tg.connected) {
    if (state.tg.chatPinned || !state.tg.chatId) return "";
    return `<div class="tg-setup">💡 Telegram is connected, but the link is lost when the
      server restarts. To make it permanent, add <code>TELEGRAM_CHAT_ID=${esc(state.tg.chatId)}</code>
      in Render → Environment.</div>`;
  }
  if (!state.tg.configured) {
    return "";
  }
  const bot = state.tg.botUsername
    ? `<a href="https://t.me/${esc(state.tg.botUsername)}" style="color:#7fb8ff">@${esc(state.tg.botUsername)}</a>`
    : "your bot";
  return `<div class="tg-setup">🔔 <b>One step left:</b> open ${bot} in Telegram and send
    <code>/start</code>. Alerts will then reach your iPhone as push notifications.</div>`;
}

async function requirePriorityAccess() {
  try {
    const response = await fetch("/api/access/status", { cache: "no-store" });
    state.access = await response.json();
  } catch {
    return true;
  }
  if (!state.access.enabled || state.access.authorized) return true;
  setSheet(`
    ${guideBanner(1, "Choose a location", "Use your location or tap your position on the map.")}
    <div class="onboarding-note">
      <b>Priority pass required</b>
      <span>${state.access.active} of ${state.access.maxUsers} concurrent places are currently active.</span>
    </div>
    <div class="input-row">
      <input type="text" id="priority-pass-input" autocomplete="off" placeholder="Enter priority pass">
      <button class="btn" id="priority-pass-enter">Enter</button>
    </div>
    <small id="priority-pass-error">Access stops when the concurrent-user limit is full.</small>
  `);
  const enter = async () => {
    const pass = $("#priority-pass-input").value.trim();
    if (!pass) return;
    const response = await fetch("/api/access/enter", {
      method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ pass }),
    });
    const result = await response.json().catch(() => ({}));
    if (!response.ok) {
      $("#priority-pass-error").textContent = result.error || "Access denied";
      return;
    }
    location.reload();
  };
  $("#priority-pass-enter").onclick = enter;
  $("#priority-pass-input").addEventListener("keydown", (event) => { if (event.key === "Enter") enter(); });
  return false;
}

/* ---------- home view ---------- */
function renderHome() {
  setGuideStep(1, t("location"));
  state.view = "home";
  stopRefresh();
  const recents = getRecents();
  setSheet(`
    ${guideBanner(1, t("location"), t("locationHelp"))}
    ${telegramSetupHTML()}
    ${state.access.enabled ? `<small class="access-count">Priority access: ${state.access.active}/${state.access.maxUsers} active · pass rank ${state.access.rank}</small>` : ""}
    <div class="onboarding-note">
      <b>${t("homeTitle")}</b>
      <span>${t("homeHelp")}</span>
      <span>${t("mobileTip")}</span>
    </div>
    <button class="btn" id="btn-near">${t("findStops")}</button>
    <div class="map-pick-hint">${t("mapTapHint")}</div>
    <details class="optional-route">
      <summary>${t("optionalRoute")}</summary>
      <div class="input-row">
        <input type="text" id="trip-input" inputmode="numeric" placeholder="${t("tripPlaceholder")}">
        <button class="btn" id="btn-open-trip">${t("go")}</button>
      </div>
      <small>${t("tripHelp")} <a href="https://namtang.otp.go.th" style="color:#7fb8ff">namtang.otp.go.th</a>.</small>
    </details>
    ${recents.length ? `<h2>${t("recentRoutes")}</h2><div class="chips">` +
      recents.map((r) => `<button class="route-chip" style="background:#2f6fed"
        data-trip="${esc(r.tripId)}">${esc(r.name)} → ${esc(r.headsign)}</button>`).join("") + `</div>` : ""}
    <div id="nearby-out"></div>
    <button class="btn btn-ghost btn-clear-cache" id="btn-clear-cache">${t("clearSaved")}</button>
    <div class="civic-tech-label">${t("civic")}</div>
    <small class="app-version">Version ${APP_VERSION}</small>
  `);

  $("#btn-near").onclick = loadNearby;
  $("#btn-clear-cache").onclick = clearAppCache;
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
    const locationHelp = window.isSecureContext
      ? "Allow location access for this site in your browser settings, then try again."
      : "Phone browsers block location on an HTTP connection. Tap your location on the map instead, or use the site over HTTPS.";
    out.innerHTML = `<h2>Nearby stops</h2><small>⚠️ Couldn't get your location. ${locationHelp}</small>`;
    return;
  }
  loadNearbyAt(pos, true);
}

async function loadNearbyAt(pos, showUserPin = false) {
  const out = $("#nearby-out");
  if (!out || state.view !== "home") return;
  if (showUserPin) showMe(pos);
  else {
    layers.me.clearLayers();
    L.circleMarker(pos, { radius: 9, color: "#fff", weight: 2, fillColor: "#8b5cf6", fillOpacity: 1 })
      .bindTooltip("Your selected location", { permanent: false }).addTo(layers.me);
  }
  const desktop = window.matchMedia("(min-width: 760px)").matches;
  map.setView(pos, Math.max(map.getZoom(), desktop ? 17 : 16));
  out.innerHTML = `<h2>Nearby stops</h2><small>Loading routes…</small>`;
  try {
    const stops = await api(`/api/nearby?lat=${pos[0]}&lon=${pos[1]}`);
    if (!stops.length) {
      out.innerHTML = `<h2>Nearby stops</h2><small>No stops found near you.</small>`;
      return;
    }
    layers.stops.clearLayers();
    stops.slice(0, 10).forEach((s) => {
      if (!s.location?.lat || !s.location?.lon) return;
      L.circleMarker([s.location.lat, s.location.lon], {
        radius: 7, color: "#fff", weight: 2, fillColor: "#e8b25a", fillOpacity: 1,
        bubblingMouseEvents: false,
      }).bindTooltip(esc(s.name)).on("click", () => {
        $("#sheet").classList.remove("collapsed");
        showNearbyStop(s);
        setTimeout(() => {
          document.getElementById(`near-stop-${s.id}`)?.scrollIntoView({ behavior: "smooth", block: "start" });
        }, 280);
      }).addTo(layers.stops);
    });
    out.innerHTML = `${guideBanner(2, t("stop"), t("stopHelp"))}` + stops.slice(0, 10).map((s) => `
      <div class="stop-card" id="near-stop-${esc(s.id)}">
        <button class="stop-name nearby-stop-open" data-show-stop="${esc(s.id)}">🚏 ${esc(s.name)} <span>${t("showArrivals")}</span></button>
        <div class="nearby-routes" data-stop-routes="${esc(s.id)}"><small>${t("selectStopBuses")}</small></div>
      </div>`).join("");
    setGuideStep(2, t("stop"));
    revealSheetTarget("#guide-step-2");
    out.querySelectorAll("[data-trip][data-stop]").forEach((b) => {
      b.onclick = () => openTrip(b.dataset.trip, b.dataset.stop || null);
    });
    out.querySelectorAll("[data-show-stop]").forEach((button) => {
      button.onclick = () => showNearbyStop(stops.find((s) => String(s.id) === button.dataset.showStop));
    });
    if (!showUserPin && !window.matchMedia("(min-width: 760px)").matches) {
      // Give the map back to the user after a manual location pick so the
      // nearby stop pins are not hidden behind the iPhone sheet.
      $("#sheet").classList.add("collapsed");
    }
  } catch (e) {
    out.innerHTML = `<h2>Nearby stops</h2><small>⚠️ ${esc(e.message)}</small>`;
  }
}

function arrivalText(waitTime) {
  const seconds = Number(waitTime) || 0;
  if (seconds <= 0) return "Live time unavailable";
  if (seconds > 3600) return ">1 hr";
  return `${Math.max(1, Math.ceil(seconds / 60))} min`;
}

function nearbyTripsHTML(trips, stopId, live) {
  const available = (trips || []).filter((t) => t.hasGps);
  if (!available.length) return `<small>No live buses currently serve this stop.</small>`;
  return available.map((t) => `
    <button class="nearby-route-card" data-trip="${esc(t.tripId)}" data-stop="${esc(stopId)}">
      <span class="nearby-route-number" style="background:#${esc(t.color || "555")}">${esc(t.name)}</span>
      <span class="nearby-route-info">
        <b>${esc(t.tripHeadsign)} <small>(${t.airCondition ? "Air Condition" : "Ordinary Standard 3"})</small></b>
        <span>${esc(t.routeLongName)}</span>
      </span>
      <strong>${live ? esc(arrivalText(t.waitTime)) : "Select route"}</strong>
    </button>`).join("");
}

async function showNearbyStop(stop) {
  if (!stop) return;
  const box = document.querySelector(`[data-stop-routes="${CSS.escape(String(stop.id))}"]`);
  if (!box) return;
  box.innerHTML = `<small>Loading live arrivals for ${esc(stop.name)}…</small>`;
  try {
    const trips = await api(`/api/passing/${encodeURIComponent(stop.id)}`);
    document.querySelectorAll("#guide-step-3").forEach((banner) => banner.remove());
    box.innerHTML = guideBanner(3, t("route"), t("routeHelp")) + nearbyTripsHTML(trips, stop.id, true);
    setGuideStep(3, t("route"));
    revealSheetTarget(box.querySelector("#guide-step-3"));
    box.querySelectorAll("[data-trip]").forEach((button) => {
      button.onclick = () => openTrip(button.dataset.trip, button.dataset.stop);
    });
  } catch (e) {
    box.innerHTML = `<small>⚠️ ${esc(e.message)}</small>`;
  }
}

map.on("click", (e) => {
  if (state.view === "home") loadNearbyAt([e.latlng.lat, e.latlng.lng]);
});

/* ---------- trip view ---------- */
async function openTrip(tripId, preferredStopId = null) {
  state.view = "trip";
  state.tripId = String(tripId);
  state.selectedBus = null;
  state.selectedStop = null;
  state.visibleBusIds = null;
  state.follow = false;
  setSheet(`<h2>Loading trip ${esc(tripId)}…</h2>`);
  try {
    const trip = await api(`/api/trip/${encodeURIComponent(tripId)}`);
    state.trip = trip;
    addRecent(state.tripId, trip.routeShortName, trip.tripHeadsign);
    drawTrip(trip, false);
    const remembered = getLastStopSelection();
    const stopToRestore = preferredStopId || (String(remembered?.tripId) === String(tripId) ? remembered.stopId : null);
    const preferredStop = stopToRestore
      ? (trip.stopList || []).find((s) => String(s.stopId) === String(stopToRestore))
      : null;
    if (preferredStop) selectStop(preferredStop);
    else renderTripSheet();
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
    }).bindTooltip(esc(s.stopName)).on("click", () => selectStop(s)).addTo(layers.stops);
  });

  updateBuses(trip);
}

function updateBuses(trip) {
  const seen = new Set();
  (trip.gpsList || []).filter((b) => !state.visibleBusIds || state.visibleBusIds.has(b.id)).forEach((b) => {
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
      }).on("click", () => selectBus(b.id, { userAction: true })).addTo(layers.buses);
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
  setGuideStep(2, t("routeStop"));
  state.selectedStop = null;
  state.visibleBusIds = null;
  updateBuses(t);
  setSheet(`
    <div style="display:flex;align-items:center;gap:10px">
      <span class="route-chip" style="background:#${esc(t.routeColor || "555")};cursor:default">${esc(t.routeShortName)}</span>
      <div style="flex:1;min-width:0">
        <div style="font-weight:600">→ ${esc(t.tripHeadsign)}</div>
        <small>${esc(t.routeLongName)} · Trip ID ${esc(state.tripId)}</small><br>${tripFeatureHTML(t)}
      </div>
      <button class="btn btn-ghost" style="width:auto;padding:8px 12px" id="btn-back">‹ Back</button>
    </div>
    ${telegramSetupHTML()}
    ${guideBanner(2, t("routeStop"), t("routeStopHelp"))}
    <small>Tap a stop below or tap its pin on the map. Buses will be sorted nearest-first afterward.</small>
    <div id="stop-list" class="route-stop-list">
      ${(t.stopList || []).length ? t.stopList.map((s) => `
        <button class="route-stop-button" data-stop-id="${esc(s.stopId)}">
          <span>🚏</span><span>${esc(s.stopName)}</span><span>›</span>
        </button>`).join("") : `<small>No stops are available for this trip.</small>`}
    </div>
    <div id="bus-detail"></div>
  `);
  $("#btn-back").onclick = () => { clearTripLayers(); renderHome(); };
  sheetContent.querySelectorAll(".route-stop-button").forEach((button) => {
    button.onclick = () => {
      const stop = (t.stopList || []).find((s) => String(s.stopId) === button.dataset.stopId);
      if (stop) selectStop(stop);
    };
  });
}

function tripFeatureHTML(trip) {
  const vehicle = (trip.vehicleList || [])[0] || {};
  const service = trip.airCondition ? "❄️ Air-conditioned" : "🌬️ Non-air-conditioned";
  const subtype = vehicle.subType ? ` · ${esc(vehicle.subType)}` : "";
  const price = vehicle.price ? ` · ${esc(vehicle.price)}` : "";
  return `<span class="service-badge"><i style="background:#${esc(trip.routeColor || "777")}"></i>${service}${subtype}${price}</span>`;
}

function distanceMeters(lat1, lon1, lat2, lon2) {
  const rad = (v) => v * Math.PI / 180;
  const dLat = rad(lat2 - lat1), dLon = rad(lon2 - lon1);
  const a = Math.sin(dLat / 2) ** 2 + Math.cos(rad(lat1)) * Math.cos(rad(lat2)) * Math.sin(dLon / 2) ** 2;
  return 6371000 * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
}

function busDistanceFromStop(b, stop) {
  const lat = Number(b.snapped_lat) || Number(b.lat);
  const lon = Number(b.snapped_lon) || Number(b.lon);
  return lat && lon ? distanceMeters(lat, lon, stop.location.lat, stop.location.lon) : Infinity;
}

function updateBusMotion(trip) {
  if (!state.selectedStop) return;
  const now = Date.now();
  (trip.gpsList || []).forEach((b) => {
    const distance = busDistanceFromStop(b, state.selectedStop);
    if (!Number.isFinite(distance)) return;
    const previous = state.busMotion[b.id];
    let trend = previous?.trend || "measuring";
    if (previous && now - previous.sampledAt >= 4000) {
      const change = distance - previous.distance;
      if (change <= -5) trend = "approaching";
      else if (change >= 5) trend = "moving-away";
      else trend = "unclear";
    }
    state.busMotion[b.id] = { distance, sampledAt: now, trend };
  });
}

function selectStop(stop) {
  setGuideStep(4, t("bus"));
  state.selectedStop = stop;
  rememberStopSelection(stop);
  state.selectedBus = null;
  state.busMotion = {};
  updateBusMotion(state.trip);
  stopCamViewer();
  layers.camera.clearLayers();
  const buses = [...(state.trip?.gpsList || [])].sort((a, b) =>
    busDistanceFromStop(a, stop) - busDistanceFromStop(b, stop)).slice(0, 2);
  state.visibleBusIds = new Set(buses.map((b) => b.id));
  updateBuses(state.trip);
  setSheet(`
    <div style="display:flex;align-items:center;gap:10px">
      <div style="font-size:24px">🚏</div>
      <div style="flex:1;min-width:0"><b>${esc(stop.stopName)}</b><br><small>${t("twoNearest")}</small></div>
      <button class="btn btn-ghost" style="width:auto;padding:8px 12px" id="btn-all-buses">${t("allBuses")}</button>
    </div>
    <div id="arrival-estimate" class="arrival-card"><small>${t("loadingArrival")}</small></div>
    ${guideBanner(4, t("bus"), t("busHelp"))}
    <div id="bus-list">
      ${buses.length ? buses.map((b) => busRowHTML(b, stop)).join("") : `<small>${t("noLiveTrip")}</small>`}
    </div>
    <div id="bus-detail"></div>`);
  $("#btn-all-buses").onclick = () => renderAllBuses(stop);
  bindBusRows();
  loadArrivalEstimate(stop);
  const points = [[stop.location.lat, stop.location.lon], ...buses.map((b) => [
    Number(b.snapped_lat) || Number(b.lat), Number(b.snapped_lon) || Number(b.lon),
  ])].filter((p) => p[0] && p[1]);
  if (points.length > 1) {
    const desktop = window.matchMedia("(min-width: 760px)").matches;
    map.fitBounds(L.latLngBounds(points), {
      paddingTopLeft: desktop ? [470, 80] : [30, 80],
      paddingBottomRight: desktop ? [40, 40] : [30, 160],
      maxZoom: 16,
    });
  } else map.setView([stop.location.lat, stop.location.lon], 16);
  revealSheetTarget("#guide-step-4");
}

function renderAllBuses(returnStop) {
  setGuideStep(4, t("allBuses"));
  state.selectedStop = null;
  state.selectedBus = null;
  state.visibleBusIds = null;
  updateBuses(state.trip);
  const buses = state.trip?.gpsList || [];
  setSheet(`
    <div style="display:flex;align-items:center;gap:10px">
      <div style="font-size:24px">🚌</div>
      <div style="flex:1;min-width:0"><b>${t("allBuses")}</b><br><small>${esc(state.trip?.routeShortName || "")}</small></div>
      <button class="btn btn-ghost" style="width:auto;padding:8px 12px" id="btn-return-stop">${t("backToStop")}</button>
    </div>
    ${guideBanner(4, t("chooseBus"), t("chooseBusHelp"))}
    <div id="bus-list">
      ${buses.length ? buses.map((bus) => busRowHTML(bus)).join("") : `<small>${t("noGps")}</small>`}
    </div>
    <div id="bus-detail"></div>
  `);
  $("#btn-return-stop").onclick = () => selectStop(returnStop);
  bindBusRows();
  revealSheetTarget("#guide-step-4");
}

async function loadArrivalEstimate(stop) {
  const box = document.getElementById("arrival-estimate");
  if (!box || !stop || state.view !== "trip") return;
  try {
    const arrival = await api(`/api/arrivals?stop=${encodeURIComponent(stop.stopId)}&trip=${encodeURIComponent(state.tripId)}`);
    if (state.selectedStop?.stopId !== stop.stopId) return;
    const first = (arrival.gpsList || []).find((b) => b.is_first_to_arrive) || (arrival.gpsList || [])[0];
    const seconds = Math.max(0, Math.round(Number(arrival.waitTime) || 0));
    const eta = seconds <= 30 ? "Arriving now" : `About ${Math.max(1, Math.ceil(seconds / 60))} min`;
    box.innerHTML = first
      ? `<div><span class="arrival-icon">⏱️</span><b>${eta}</b> <small>· Namtang estimate</small></div>
         <div class="arrival-bus">Next: bus ${esc(first.id.split(" ")[0])}${first.is_approaching_stop ? " · approaching" : ""}</div>`
      : `<small>${t("noApproaching")}</small>`;
  } catch {
    box.innerHTML = `<small>Namtang arrival estimate is temporarily unavailable.</small>`;
  }
}

function busRowHTML(b, stop = null) {
  const plate = b.id.split(" ")[0];
  const speed = Math.round(Number(b.speed) || 0);
  const ageSec = b.received ? Math.max(0, Math.round(Date.now() / 1000 - Number(b.received))) : Infinity;
  const stale = ageSec > 90;
  const stopped = !stale && speed === 0;
  const slow = !stale && speed > 0 && speed < 4;
  const speedText = stale ? "GPS stale" : stopped ? "stopped" : `${speed} km/h`;
  const stopDistance = stop ? busDistanceFromStop(b, stop) : null;
  const proximity = stopDistance === null ? "" : ` · ${stopDistance < 1000 ? Math.round(stopDistance) + " m" : (stopDistance / 1000).toFixed(1) + " km"} from stop`;
  const trend = stop ? (state.busMotion[b.id]?.trend || "measuring") : "";
  const trendHTML = trend === "approaching" ? `<span class="motion motion-toward">↓ approaching stop</span>`
    : trend === "moving-away" ? `<span class="motion motion-away">↑ moving away</span>`
    : trend === "unclear" ? `<span class="motion motion-unclear">• stopped/unclear</span>`
    : stop ? `<span class="motion motion-measuring">… measuring direction (5s)</span>` : "";
  const sel = state.selectedBus === b.id;
  return `<div class="bus-row ${sel ? "selected" : ""}" data-bus="${esc(b.id)}">
    <div style="font-size:20px">🚌</div>
    <div class="grow">
      <div class="plate">${esc(plate)}</div>
      <div class="sub">→ ${esc(b.next_stop_name || "?")}${proximity} · updated ${fmtAgo(b.received)}</div>
      ${trendHTML}
    </div>
    <div class="speed-badge ${stale ? "stale" : stopped ? "stopped" : slow ? "slow" : ""}">${speedText}</div>
  </div>`;
}

function bindBusRows() {
  sheetContent.querySelectorAll(".bus-row").forEach((row) => {
    row.onclick = () => selectBus(row.dataset.bus, { userAction: true });
  });
}

async function selectBus(busId, options = {}) {
  const changedBus = state.selectedBus !== busId;
  if (changedBus) {
    clearTimeout(state.cameraHandoffTimer);
    state.cameraHandoffTimer = null;
    state.activeCameraBus = null;
    state.activeCameraId = null;
    state.pendingCameraId = null;
    state.cameraIndexOffset = 0;
  }
  if (changedBus || options.userAction) $("#sheet").classList.remove("collapsed");
  state.selectedBus = busId;
  if (options.userAction) setGuideStep(5, t("view"));
  const selectionVersion = ++state.selectionVersion;
  state.follow = true;
  updateBuses(state.trip);
  const m = busMarkers[busId];
  if (m) map.setView(m.getLatLng(), Math.max(map.getZoom(), 15));

  sheetContent.querySelectorAll(".bus-row").forEach((r) =>
    r.classList.toggle("selected", r.dataset.bus === busId));

  const detail = $("#bus-detail");
  if (!detail) return;
  const plate = busId.split(" ")[0];
  if (changedBus || !detail.innerHTML) {
    detail.innerHTML = `<h2>Bus ${esc(plate)}</h2><small>Loading details…</small>`;
  }

  let camHTML = "";
  let camId = null;
  let busCameraBounds = null;
  try {
    const d = await api(`/api/bus?trip=${encodeURIComponent(state.tripId)}&bus=${encodeURIComponent(plate)}`);
    if (selectionVersion !== state.selectionVersion || state.selectedBus !== busId) return;
    const cameraCandidates = d.cameraCandidates || [];
    if (cameraCandidates.length) {
      state.cameraIndexOffset = Math.min(state.cameraIndexOffset, cameraCandidates.length - 1);
      const chosen = cameraCandidates[state.cameraIndexOffset];
      d.nearestCamera = chosen.camera;
      d.cameraDistanceM = chosen.distanceM;
      if (state.cameraIndexOffset > 0) d.cameraSelection = "next";
    }
    if (d.nearestCamera) {
      const nextCameraId = String(d.nearestCamera.id);
      if (state.activeCameraBus === busId && state.activeCameraId && state.activeCameraId !== nextCameraId) {
        // Keep the just-passed camera live for five seconds. Repeated refreshes
        // must not restart the grace period.
        if (state.pendingCameraId !== nextCameraId) {
          clearTimeout(state.cameraHandoffTimer);
          state.pendingCameraId = nextCameraId;
          state.cameraHandoffTimer = setTimeout(() => {
            if (state.selectedBus !== busId || state.pendingCameraId !== nextCameraId) return;
            state.activeCameraId = nextCameraId;
            state.pendingCameraId = null;
            state.cameraHandoffTimer = null;
            selectBus(busId);
          }, 5000);
        }
        return;
      }
      state.activeCameraBus = busId;
      state.activeCameraId = nextCameraId;
      state.pendingCameraId = null;
      layers.camera.clearLayers();
      camId = d.nearestCamera.id;
      const cam = d.nearestCamera;
      const busLat = Number(d.bus.snapped_lat) || Number(d.bus.lat);
      const busLon = Number(d.bus.snapped_lon) || Number(d.bus.lon);
      const camIcon = L.divIcon({
        html: `<div class="camera-marker">📷</div>`,
        className: "", iconSize: [34, 34], iconAnchor: [17, 17],
      });
      L.marker([cam.lat, cam.lon], { icon: camIcon, zIndexOffset: 450 })
        .bindPopup(`<b>Camera ${esc(cam.id)}</b><br>${esc(cam.name_th || cam.name_en || "")}`)
        .addTo(layers.camera);
      if (busLat && busLon) {
        busCameraBounds = L.latLngBounds([[busLat, busLon], [cam.lat, cam.lon]]);
        L.polyline([[busLat, busLon], [cam.lat, cam.lon]], {
          color: "#e8b25a", weight: 2, opacity: .85, dashArray: "6 7",
        }).addTo(layers.camera);
      }
      camHTML = `
        <div id="camera-section">
        <h2>🎥 ${d.cameraSelection === "nearest" ? t("nearestCamera") : t("upcomingCamera")} <small>· ${Math.round(d.cameraDistanceM)} ${t("fromBus")} · ${d.cameraOnRoute ? t("onRoute") : t("nearRoute")}</small></h2>
        <div class="camera-link-info">
          <b>${esc(d.nearestCamera.name_th || d.nearestCamera.name_en || d.nearestCamera.id)}</b>
          <small>${t("cameraId")}: ${esc(d.nearestCamera.id)}</small>
        </div>
        <a id="camera-link" class="btn btn-ghost btn-direct-camera"
          data-camera-id="${esc(camId)}" href="${esc(d.nearestCamera.feed_url)}"
          target="_blank" rel="noopener">${t("openCamera")}</a>
        <div class="camera-site-note">ℹ️ ${t("cameraNote")}</div>
        <div class="btn-row camera-nav">
          <button class="btn btn-ghost" id="btn-prev-camera" ${state.cameraIndexOffset <= 0 ? "disabled" : ""}>${t("previousCamera")}</button>
          <button class="btn btn-ghost" id="btn-next-camera" ${state.cameraIndexOffset >= cameraCandidates.length - 1 ? "disabled" : ""}>${t("nextCamera")}</button>
        </div>
        <button class="btn btn-ghost btn-map-pins" id="btn-map-pins">${t("showPins")}</button>
        </div>`;
    }
  } catch { /* camera info is best-effort */ }

  if (selectionVersion !== state.selectionVersion || state.selectedBus !== busId) return;

  const b = (state.trip.gpsList || []).find((x) => x.id === busId) || {};
  if (options.background && !changedBus && detail.innerHTML) {
    const direction = document.getElementById("detail-direction");
    const nextStop = document.getElementById("detail-next-stop");
    const speed = document.getElementById("detail-speed");
    const updated = document.getElementById("detail-updated");
    if (direction) direction.textContent = b.is_reversed ? "↩ Opposite/return direction" : `→ Toward ${state.trip.tripHeadsign}`;
    if (nextStop) nextStop.textContent = `${b.next_stop_name || "?"} (${Math.round(Number(b.distance_to_next_stop) || 0)} m)`;
    if (speed) speed.textContent = `${Math.round(Number(b.speed) || 0)} km/h`;
    if (updated) updated.textContent = fmtAgo(b.received);
    const shownCamera = document.getElementById("camera-link")?.dataset.cameraId || null;
    // Missing camera data means "keep what is already rendered", never
    // "remove the camera". BMA availability can fluctuate between refreshes.
    if (!camId || (shownCamera && String(camId) === shownCamera)) return;
  }
  const stopBuses = state.selectedStop
    ? (state.trip.gpsList || []).filter((candidate) => !state.visibleBusIds || state.visibleBusIds.has(candidate.id))
    : [];
  const busSwitcher = stopBuses.length > 1 ? `
    <div class="bus-switcher">
      <small>Other buses near ${esc(state.selectedStop.stopName)}</small>
      <div class="chips">${stopBuses.map((candidate) => `
        <button class="bus-switch-chip ${candidate.id === busId ? "active" : ""}" data-switch-bus="${esc(candidate.id)}">
          ${esc(candidate.id.split(" ")[0])}
        </button>`).join("")}</div>
    </div>` : "";
  detail.innerHTML = `
    ${guideBanner(5, t("view"), t("viewHelp"))}
    <h2>Bus ${esc(plate)}</h2>
    ${state.selectedStop ? `<button class="btn btn-ghost btn-back-stop" id="btn-back-stop">‹ Other buses at ${esc(state.selectedStop.stopName)}</button>` : ""}
    ${busSwitcher}
    <div class="trip-id-line">Route ${esc(state.trip.routeShortName)} · Trip ID ${esc(state.tripId)}</div>
    ${tripFeatureHTML(state.trip)}
    ${camId ? `<button class="camera-available" id="btn-view-camera"><b>${t("cameraAvailable")}</b><span>${t("cameraAvailableHelp")}</span></button>` : ""}
    <dl class="detail-grid">
      <dt>Direction</dt><dd id="detail-direction">${b.is_reversed ? "↩ Opposite/return direction" : `→ Toward ${esc(state.trip.tripHeadsign)}`}</dd>
      <dt>Next stop</dt><dd id="detail-next-stop">${esc(b.next_stop_name || "?")} (${Math.round(Number(b.distance_to_next_stop) || 0)} m)</dd>
      <dt>Speed</dt><dd id="detail-speed">${Math.round(Number(b.speed) || 0)} km/h</dd>
      <dt>Updated</dt><dd id="detail-updated">${fmtAgo(b.received)}</dd>
    </dl>
    ${state.tg.connected ? `<button class="btn" id="btn-alert">${t("alertPlace")}</button>` : ""}
    ${camHTML}
  `;
  if (options.userAction) revealSheetTarget("#guide-step-5");
  $("#btn-view-camera")?.addEventListener("click", () => revealSheetTarget("#camera-section", 0));
  $("#btn-alert")?.addEventListener("click", startAlertFlow);
  $("#btn-back-stop")?.addEventListener("click", () => selectStop(state.selectedStop));
  detail.querySelectorAll("[data-switch-bus]").forEach((button) => {
    button.onclick = () => selectBus(button.dataset.switchBus, { userAction: true });
  });
  const showPins = () => {
    if (!busCameraBounds) return;
    const desktop = window.matchMedia("(min-width: 760px)").matches;
    if (!desktop) $("#sheet").classList.add("collapsed");
    const framePins = () => {
      map.invalidateSize({ pan: false });
      map.fitBounds(busCameraBounds, {
        paddingTopLeft: desktop ? [470, 90] : [36, 100],
        paddingBottomRight: desktop ? [40, 40] : [36, 100],
        maxZoom: 17,
      });
    };
    // Brave/iOS needs the sheet transition to finish before Leaflet fits the
    // pins, otherwise it uses stale viewport measurements.
    if (desktop) framePins(); else setTimeout(framePins, 320);
  };
  $("#btn-map-pins")?.addEventListener("click", showPins);
  $("#btn-prev-camera")?.addEventListener("click", () => {
    state.cameraIndexOffset = Math.max(0, state.cameraIndexOffset - 1);
    state.activeCameraId = null;
    selectBus(busId);
  });
  $("#btn-next-camera")?.addEventListener("click", () => {
    state.cameraIndexOffset++;
    state.activeCameraId = null;
    selectBus(busId);
  });
  if (busCameraBounds && window.matchMedia("(min-width: 760px)").matches) showPins();
  if ((changedBus || options.userAction) && camId) {
    const cameraImage = document.getElementById("cam-img");
    if (cameraImage) cameraImage.dataset.scrollBottom = "true";
    requestAnimationFrame(() => {
      sheetContent.scrollTo({ top: sheetContent.scrollHeight, behavior: "smooth" });
    });
  }
  if (camId && document.getElementById("cam-img")) startCamViewer(camId);
}

/* ---------- live camera viewer ---------- */
let camTimer = null;
let camGeneration = 0;
let camLoading = false;

function startCamViewer(camId) {
  stopCamViewer();
  const generation = ++camGeneration;
  const current = document.getElementById("cam-img");
  if (current) current.dataset.cameraId = camId;
  const load = () => {
    const el = document.getElementById("cam-img");
    if (!el || generation !== camGeneration || el.dataset.cameraId !== camId) return;
    if (document.hidden || camLoading) return;
    camLoading = true;
    const next = new Image();
    next.onload = () => {
      camLoading = false;
      if (generation !== camGeneration || el.dataset.cameraId !== camId) return;
      el.src = next.src;
      el.classList.remove("cam-err");
      const status = document.getElementById("cam-status");
      if (status) status.textContent = "Live camera · bus tracking stays independent";
      if (el.dataset.scrollBottom === "true") {
        el.dataset.scrollBottom = "false";
        requestAnimationFrame(() => {
          sheetContent.scrollTo({ top: sheetContent.scrollHeight, behavior: "smooth" });
        });
      }
    };
    next.onerror = () => {
      camLoading = false;
      if (generation === camGeneration && el.dataset.cameraId === camId) {
        el.classList.add("cam-err");
        const status = document.getElementById("cam-status");
        if (status) status.textContent = "Camera source is slow or unavailable; bus tracking is still live.";
      }
    };
    next.src = `/api/camera/${encodeURIComponent(camId)}/frame?t=${Date.now()}`;
  };
  load();
  camTimer = setInterval(load, 3000);
}

function stopCamViewer() {
  camGeneration++;
  camLoading = false;
  if (camTimer) clearInterval(camTimer);
  camTimer = null;
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
      updateBusMotion(trip);
      if (state.selectedStop && !state.selectedBus) {
        const nearest = [...(trip.gpsList || [])]
          .sort((a, b) => busDistanceFromStop(a, state.selectedStop) - busDistanceFromStop(b, state.selectedStop))
          .slice(0, 2);
        state.visibleBusIds = new Set(nearest.map((b) => b.id));
      }
      updateBuses(trip);
      const list = $("#bus-list");
      if (list) {
        const buses = state.selectedStop && !state.selectedBus
          ? [...(trip.gpsList || [])].sort((a, b) => busDistanceFromStop(a, state.selectedStop) - busDistanceFromStop(b, state.selectedStop)).slice(0, 2)
          : (trip.gpsList || []);
        list.innerHTML = buses.map((b) => busRowHTML(b, state.selectedStop && !state.selectedBus ? state.selectedStop : null)).join("") ||
          `<small>${t("noGps")}</small>`;
        bindBusRows();
      }
      // Refresh the selected bus's speed, next stop, position-dependent camera,
      // and detail timestamp too. Previously only its map marker moved.
      if (state.selectedBus) {
        const stillPresent = (trip.gpsList || []).some((b) => b.id === state.selectedBus);
        if (stillPresent) await selectBus(state.selectedBus, { background: true });
      }
      if (state.selectedStop) loadArrivalEstimate(state.selectedStop);
    } catch { /* transient */ }
  }, REFRESH_MS);
}

function stopRefresh() {
  if (state.refreshTimer) clearInterval(state.refreshTimer);
  state.refreshTimer = null;
}

function clearTripLayers() {
  state.selectionVersion++;
  clearTimeout(state.cameraHandoffTimer);
  state.cameraHandoffTimer = null;
  state.activeCameraBus = null;
  state.activeCameraId = null;
  state.pendingCameraId = null;
  state.cameraIndexOffset = 0;
  state.visibleBusIds = null;
  state.busMotion = {};
  stopCamViewer();
  layers.route.clearLayers();
  layers.stops.clearLayers();
  layers.buses.clearLayers();
  layers.camera.clearLayers();
  layers.alert.clearLayers();
  Object.keys(busMarkers).forEach((k) => delete busMarkers[k]);
  stopRefresh();
}

/* ---------- wire up ---------- */
$("#btn-home").onclick = () => { clearTripLayers(); renderHome(); };
$("#btn-reset").onclick = () => {
  if (!window.confirm(t("reset"))) return;
  localStorage.removeItem(BMA_PREFLIGHT_KEY);
  localStorage.removeItem("lastStopSelection");
  localStorage.removeItem("bmaCameraNoticeSeen");
  location.reload();
};
$("#language-select").onchange = (event) => {
  currentLang = event.target.value === "th" ? "th" : "en";
  try { localStorage.setItem("buswatchLanguage", currentLang); } catch {}
  location.reload();
};
$("#btn-about").onclick = () => {
  const thai = currentLang === "th";
  setGuideStep(5, thai ? "เกี่ยวกับ Bus-287" : "About Bus-287");
  setSheet(`
    <div class="about-copy">
      <h2>${thai ? "เกี่ยวกับ Bus-287" : "About Bus-287"}</h2>
      <p>${thai ? "<b>Bus-287</b> เป็นชื่อย่อของโครงการ Thai Bus Watch มาจากเลขซ้ำที่จำง่ายในที่อยู่ <b>287287287.xyz</b>" : "<b>Bus-287</b> is the short project nickname for Thai Bus Watch. It comes from the memorable repeated number in this service's address: <b>287287287.xyz</b>."}</p>
      <p>${thai ? "โครงการทดลองเทคโนโลยีเพื่อสังคมอิสระ สำหรับดูตำแหน่งรถโดยสาร ป้ายใกล้เคียง เวลาถึง และลิงก์กล้องจราจร" : "This is an independent experimental civic-tech project for viewing live Bangkok bus locations, nearby stops, arrival information, and links to public traffic-camera services."}</p>
      <p>${thai ? "โครงการนี้ไม่ได้เป็นส่วนหนึ่งหรือได้รับการรับรองจากกรุงเทพมหานครหรือผู้ให้บริการข้อมูลขนส่ง" : "It is not affiliated with or endorsed by Bangkok Metropolitan Administration or the public transport data providers it references."}</p>
      <button class="btn btn-ghost" id="btn-retest-camera">${thai ? "ทดสอบกล้อง BMA อีกครั้ง" : "Retest BMA camera access"}</button>
      <button class="btn btn-ghost" id="btn-about-close">${thai ? "ปิด" : "Close About"}</button>
    </div>
  `);
  $("#btn-retest-camera").onclick = () => {
    localStorage.removeItem(BMA_PREFLIGHT_KEY);
    location.reload();
  };
  $("#btn-about-close").onclick = () => {
    if (state.view === "trip" && state.trip) {
      if (state.selectedStop) selectStop(state.selectedStop);
      else renderTripSheet();
    } else renderHome();
  };
};
const sheet = $("#sheet");
const sheetHandle = $("#sheet-handle");
function updateSheetHandle() {
  const collapsed = sheet.classList.contains("collapsed");
  const step = state.guideStep;
  sheetHandle.textContent = `${t("step")} ${step.number}/5 · ${step.label} · ${collapsed ? `▲ ${t("open")}` : `▼ ${t("hide")}`}`;
  sheetHandle.setAttribute("aria-expanded", String(!collapsed));
  sheetHandle.setAttribute("aria-label", `Step ${step.number} of 5: ${step.label}. ${collapsed ? "Open" : "Hide"} panel`);
}
let ignoreNextSheetClick = false;
sheetHandle.onclick = () => {
  if (ignoreNextSheetClick) {
    ignoreNextSheetClick = false;
    return;
  }
  sheet.classList.toggle("collapsed");
};
new MutationObserver(updateSheetHandle).observe(sheet, { attributes: true, attributeFilter: ["class"] });
updateSheetHandle();
let sheetTouchY = null;
sheetHandle.addEventListener("touchstart", (e) => {
  sheetTouchY = e.changedTouches[0]?.clientY ?? null;
}, { passive: true });
sheetHandle.addEventListener("touchend", (e) => {
  if (sheetTouchY === null) return;
  const delta = (e.changedTouches[0]?.clientY ?? sheetTouchY) - sheetTouchY;
  sheetTouchY = null;
  if (Math.abs(delta) > 35) {
    ignoreNextSheetClick = true;
    setTimeout(() => { ignoreNextSheetClick = false; }, 500);
  }
  if (delta > 35) sheet.classList.add("collapsed");
  if (delta < -35) sheet.classList.remove("collapsed");
}, { passive: true });
let sheetContentTouchY = null;
let sheetContentStartedAtTop = false;
sheetContent.addEventListener("touchstart", (event) => {
  sheetContentTouchY = event.changedTouches[0]?.clientY ?? null;
  sheetContentStartedAtTop = sheetContent.scrollTop <= 1;
}, { passive: true });
sheetContent.addEventListener("touchend", (event) => {
  if (sheetContentTouchY === null) return;
  const delta = (event.changedTouches[0]?.clientY ?? sheetContentTouchY) - sheetContentTouchY;
  sheetContentTouchY = null;
  if (sheetContentStartedAtTop && delta > 60) sheet.classList.add("collapsed");
  sheetContentStartedAtTop = false;
}, { passive: true });
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

async function init() {
  applyLanguage();
  await requireCameraPreflight();
  if (!await requirePriorityAccess()) return;
  // Wait for the first status response before drawing the home sheet. Without
  // this, the default "Telegram off" state flashes and remains in the sheet
  // even after the status pill has updated.
  await refreshTelegram();
  const remembered = getLastStopSelection();
  if (remembered?.tripId && remembered?.stopId) await openTrip(remembered.tripId, remembered.stopId);
  else renderHome();
  refreshWatches();
  setInterval(refreshTelegram, 20000);
  state.watchTimer = setInterval(refreshWatches, 20000);

  if ("serviceWorker" in navigator) {
    navigator.serviceWorker.register("sw.js").catch(() => {});
  }
}

function requireCameraPreflight() {
  try {
    if (localStorage.getItem(BMA_PREFLIGHT_KEY)) return Promise.resolve();
  } catch { /* continue with the one-time check */ }
  const overlay = $("#camera-preflight");
  const result = $("#preflight-result");
  const help = $("#preflight-help");
  if (/iPhone|iPod/i.test(navigator.userAgent)) {
    $("#iphone-browser-warning").classList.remove("hidden");
  }
  overlay.classList.remove("hidden");
  return new Promise((resolve) => {
    $("#btn-test-bma").addEventListener("click", () => result.classList.remove("hidden"));
    const finish = (cameraWorked) => {
      try {
        localStorage.setItem(BMA_PREFLIGHT_KEY, JSON.stringify({ cameraWorked, checkedAt: Date.now() }));
      } catch { /* private browsing may not retain the answer */ }
      overlay.classList.add("hidden");
      resolve();
    };
    $("#btn-bma-worked").onclick = () => finish(true);
    $("#btn-bma-failed").onclick = () => {
      help.classList.remove("hidden");
      help.scrollIntoView({ behavior: "smooth", block: "nearest" });
    };
    $("#btn-bma-continue-without").onclick = () => finish(false);
  });
}

init();
