// StreamBridge control panel
let lang = "en", S = null, page = "dashboard", previewPlayer = null, logTimer = null, stopped = false;
let lastDestSig = "", lastQR = "", lastGuideIP = "", obsGuide = "obsws", impTarget = "obs";
const $ = id => document.getElementById(id);
const esc = s => String(s ?? "").replace(/[&<>"']/g, c => ({"&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;"}[c]));
function t(k, v) {
  let s = (I18N[lang] && I18N[lang][k]) || I18N.en[k] || k;
  if (v) for (const x in v) s = s.split("{" + x + "}").join(v[x]);
  return s;
}

// ---------- icons (24px stroke)
const ICONS = {
  dashboard: '<path d="M3 12h4l3-7 4 14 3-7h4"/>',
  setup: '<path d="M7 7h10a4 4 0 0 1 4 4v2a4 4 0 0 1-6.9 2.8L13 14.5h-2l-1.1 1.3A4 4 0 0 1 3 13v-2a4 4 0 0 1 4-4z"/><path d="M7.5 10v3M6 11.5h3"/><circle cx="16" cy="10.5" r=".8" fill="currentColor"/><circle cx="17.8" cy="12.5" r=".8" fill="currentColor"/>',
  obs: '<path d="M12 3 3 8l9 5 9-5-9-5z"/><path d="m3 13 9 5 9-5"/>',
  platforms: '<circle cx="12" cy="12" r="2"/><path d="M16.2 7.8a6 6 0 0 1 0 8.4M7.8 16.2a6 6 0 0 1 0-8.4M19.1 4.9a10 10 0 0 1 0 14.2M4.9 19.1a10 10 0 0 1 0-14.2"/>',
  settings: '<path d="M4 6h9M17 6h3M4 12h3M11 12h9M4 18h11M19 18h1"/><circle cx="15" cy="6" r="2"/><circle cx="9" cy="12" r="2"/><circle cx="17" cy="18" r="2"/>',
  help: '<circle cx="12" cy="12" r="9"/><path d="M9.6 9.2a2.5 2.5 0 1 1 3.4 2.3c-.6.3-1 .8-1 1.5v.6"/><circle cx="12" cy="16.8" r=".7" fill="currentColor"/>',
  pc: '<rect x="3" y="4" width="18" height="12" rx="2"/><path d="M8 20h8M12 16v4"/>',
  pad: '<path d="M7 7h10a4 4 0 0 1 4 4v2a4 4 0 0 1-6.9 2.8L13 14.5h-2l-1.1 1.3A4 4 0 0 1 3 13v-2a4 4 0 0 1 4-4z"/><path d="M7.5 10v3M6 11.5h3"/>'
};
const icon = n => `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${ICONS[n] || ""}</svg>`;
const PAGES = ["dashboard", "setup", "obs", "platforms", "settings", "help"];

function buildNav() {
  $("nav").innerHTML = PAGES.map(p => `<button data-page="${p}">${icon(p)}<span data-i18n="nav_${p}"></span><i class="dotmark" id="dm-${p}"></i></button>`).join("");
  $("tabbar").innerHTML = PAGES.map(p => `<button data-page="${p}">${icon(p)}<span data-i18n="${p === "obs" ? "tab_obs" : "nav_" + p}"></span><i class="dotmark" id="tdm-${p}"></i></button>`).join("");
  document.querySelectorAll("[data-icon]").forEach(el => { el.innerHTML = icon(el.dataset.icon); });
}

function showPage(p, push) {
  if (!PAGES.includes(p)) p = "dashboard";
  page = p;
  PAGES.forEach(x => { $("page-" + x).hidden = x !== p; });
  document.querySelectorAll("[data-page]").forEach(b => b.setAttribute("aria-current", b.dataset.page === p ? "page" : "false"));
  $("page-title").textContent = t("nav_" + p);
  if (push !== false && location.hash !== "#" + p) history.replaceState(null, "", "#" + p);
  if (p === "help") loadLog();
  window.scrollTo(0, 0);
}

function guideCtx() { return {ip: (S && S.activeIP) || ""}; }
function renderGuides() {
  document.querySelectorAll("[data-guide]").forEach(el => renderGuide(el, el.dataset.guide, guideCtx()));
  const og = $("obs-guide");
  const sl = renderGuide(og, obsGuide, guideCtx());
  $("manual-url").hidden = !(sl && sl.url);
  document.querySelectorAll("#obs-tabs button").forEach(b => b.classList.toggle("on", b.dataset.g === obsGuide));
}

function applyLang() {
  document.documentElement.lang = lang;
  document.documentElement.dir = lang === "fa" ? "rtl" : "ltr";
  document.querySelectorAll("[data-i18n]").forEach(el => { el.innerHTML = t(el.dataset.i18n); });
  document.querySelectorAll("[data-lang]").forEach(b => b.classList.toggle("on", b.dataset.lang === lang));
  $("page-title").textContent = t("nav_" + page);
  onPlatformChange(true);
  lastDestSig = "";
  renderGuides();
  if (S) render(S);
}
function setLang(l) { lang = l; applyLang(); post("/api/settings", {lang: l}); }

async function post(url, body) {
  const r = await fetch(url, {method: "POST", headers: {"Content-Type": "application/json", "X-StreamBridge": "1"}, body: JSON.stringify(body || {})});
  return r.json();
}

function fmtDur(sec) {
  sec = Math.max(0, Math.floor(sec)); const h = Math.floor(sec / 3600), m = Math.floor(sec % 3600 / 60), s = sec % 60;
  return (h ? h + ":" + String(m).padStart(2, "0") : m) + ":" + String(s).padStart(2, "0");
}
function fmtAgo(sec) {
  if (sec < 60) return t("ago_s", {n: Math.max(1, Math.round(sec))});
  if (sec < 3600) return t("ago_m", {n: Math.round(sec / 60)});
  return t("ago_h", {n: Math.round(sec / 3600)});
}
function stateTag(state, err) {
  state = state || "idle";
  let h = `<span class="tag ${esc(state)}">${esc(t("st_" + state))}</span>`;
  if (err && (state === "retrying" || state === "error")) h += `<div class="err-line ltr">${esc(err)}</div>`;
  return h;
}
const cls = (el, base, state) => { el.className = base + (state ? " " + state : ""); };

// ---------- render
function render(s) {
  const st = s.stream, ip = s.activeIP || "—";
  document.querySelectorAll("[data-ip]").forEach(el => el.textContent = ip);
  document.querySelectorAll("[data-obsurl]").forEach(el => el.textContent = s.obsURL);
  $("lan-url").textContent = s.lanObsURL || "";
  $("ver").textContent = "v" + s.version;
  $("phone-badge").hidden = !s.isRemote;
  if (ip !== lastGuideIP) { lastGuideIP = ip; renderGuides(); }

  // derived state
  const recent = s.devices.filter(d => d.kind !== "this-pc" && d.lastSeenAgo < 180);
  const ps = recent.find(d => d.kind === "playstation" || d.kind === "xbox") || recent.find(d => d.askedTwitch);
  const psFound = !!ps || st.live;
  const live = st.live && !st.bandwidthTest;
  const enabled = s.destinations.filter(d => d.enabled);
  const liveOuts = s.destinations.filter(d => d.state === "live").length + (s.twitchForward.state === "live" ? 1 : 0) + (st.viewers > 0 ? 1 : 0);

  // tally pill + nav dots
  const tally = live ? "onair" : psFound ? "ready" : "";
  cls($("tally"), "tally-pill", tally);
  $("tally-t").textContent = t(live ? "tally_onair" : psFound ? "tally_ready" : "tally_standby");
  const setDot = (p, c) => { [$("dm-" + p), $("tdm-" + p)].forEach(el => el && (el.className = "dotmark" + (c ? " " + c : ""))); };
  setDot("dashboard", live ? "onair" : "");
  setDot("setup", psFound ? "" : "warn");
  setDot("platforms", s.destinations.some(d => d.enabled && (d.state === "retrying" || d.state === "error")) ? "warn" : "");

  // alerts
  const al = [];
  const known = (k) => I18N.en[k] !== undefined;
  if (!s.dns.running && s.dns.error) al.push(["red", known("alert_dns_" + s.dns.error) ? t("alert_dns_" + s.dns.error) : t("alert_generic", {e: esc(s.dns.error)})]);
  if (!s.rtmpRunning && s.rtmpError) al.push(["red", known("alert_rtmp_" + s.rtmpError) ? t("alert_rtmp_" + s.rtmpError) : t("alert_generic", {e: esc(s.rtmpError)})]);
  if (s.warnings.includes("self_dns")) al.push(["amber", t("alert_self_dns")]);
  if (s.warnings.includes("no_network")) al.push(["amber", t("alert_no_network")]);
  const tn = s.tunnel;
  if (tn.enabled && tn.err443) al.push(["red", t("tun_err_" + tn.err443) !== "tun_err_" + tn.err443 ? t("tun_err_" + tn.err443) : t("alert_generic", {e: esc(tn.err443)})]);
  else if (tn.enabled && tn.running && !tn.proxyUp) al.push(["amber", t("alert_tun_down", {addr: esc(tn.proxy || "—")})]);
  $("alerts").innerHTML = al.map(([c, m]) => `<div class="alert ${c}">${m}</div>`).join("");

  // ----- signal path
  const ps5State = live ? "onair" : psFound ? "ready" : "";
  cls($("n-ps5"), "node", ps5State);
  cls($("n-ps5").querySelector(".tl"), "tl", ps5State);
  $("n-ps5-sub").innerHTML = live ? t("ps5_live", {ip: esc(st.from)}) : ps ? t("ps5_ok", {ip: esc(ps.ip)}) : t("ps5_wait");
  cls($("w1"), "wire", live ? "onair" : psFound ? "ready" : "");
  const tunOn = tn.enabled && tn.running;
  $("n-ps5-chips").hidden = !tunOn;
  if (tunOn) $("n-ps5-chips").innerHTML = `<span class="chip tun ${tn.proxyUp ? "" : "down"}"><i></i>${esc(t("tun_chip"))}</span>`;

  const pcState = !s.rtmpRunning || !s.dns.running ? "warn" : live ? (st.stalled ? "warn" : "onair") : st.live ? "warn" : "ready";
  cls($("n-pc"), "node", pcState === "onair" ? "onair" : "");
  cls($("n-pc").querySelector(".tl"), "tl", pcState);
  let pcSub = t("pc_wait");
  if (st.live && st.bandwidthTest) pcSub = t("pc_test");
  else if (live) pcSub = st.stalled ? t("pc_stalled") : t("pc_live");
  else if (s.lastStream) pcSub = t("pc_last", {ago: fmtAgo(s.lastStream.endedAgoSec), dur: fmtDur(s.lastStream.durationSec)});
  $("n-pc-sub").textContent = pcSub;
  const chips = [];
  if (live) {
    if (st.width) chips.push(`${st.width}×${st.height}`);
    if (st.fps) chips.push(`${Math.round(st.fps)} fps`);
    if (st.kbps) chips.push(`${(st.kbps / 1000).toFixed(1)} Mbps`);
    chips.push(fmtDur(st.durationSec));
  }
  $("n-pc-chips").innerHTML = chips.map(c => `<span class="chip">${esc(c)}</span>`).join("");
  $("n-pc-chips").hidden = !chips.length;
  cls($("w2"), "wire", live && liveOuts ? "onair" : (enabled.length || s.settings.forwardToTwitch || st.viewers) ? "ready" : "");

  const outs = [];
  const outRow = (name, state, sub) => `<div class="out ${state}"><i class="tl ${state}"></i><span class="name">${name}</span><span class="st">${sub}</span></div>`;
  if (st.viewers > 0 || s.import.hasObsPassword || s.import.hasStreamlabsToken) outs.push(outRow(esc(t("out_obs")), st.viewers > 0 ? (live ? "onair" : "ready") : "", st.viewers > 0 ? esc(t("out_watching", {n: st.viewers})) : esc(t("out_waiting"))));
  enabled.forEach(d => outs.push(outRow(`<span class="pf ${esc(d.platform)}">${esc(d.platform === "custom" ? "RTMP" : d.name)}</span> ${d.platform === "custom" ? esc(d.name) : ""}`, d.state === "live" ? "onair" : (d.state === "retrying" || d.state === "error") ? "warn" : "ready", esc(t("st_" + (d.state || "idle"))))));
  if (s.settings.forwardToTwitch) outs.push(outRow(`<span class="pf twitch">Twitch</span>`, s.twitchForward.state === "live" ? "onair" : s.twitchForward.state === "retrying" ? "warn" : "ready", esc(t("st_" + (s.twitchForward.state || "idle")))));
  if (!outs.length) outs.push(`<div class="out add" data-goto="obs">${esc(t("out_add"))}</div>`);
  $("outs").innerHTML = outs.join("");
  $("outs").classList.toggle("single", outs.length === 1);

  // next step
  let nx = null;
  if (!s.dns.running || !s.rtmpRunning) nx = {t: "nx_dns_t", d: "nx_dns_d"};
  else if (!psFound) nx = {t: "nx_ps5_t", d: "nx_ps5_d", btn: "nx_btn_guide", go: "setup"};
  else if (!st.live) nx = {t: "nx_live_t", d: "nx_live_d", btn: "nx_btn_guide", go: "setup"};
  else if (live && !liveOuts) nx = {t: "nx_out_t", d: "nx_out_d", btn: "nx_btn_obs", go: "obs", btn2: "nx_btn_platforms", go2: "platforms"};
  const nxEl = $("next");
  if (nx) {
    nxEl.hidden = false;
    const sig = JSON.stringify(nx) + lang + ip;
    if (nxEl.dataset.sig !== sig) {
      nxEl.dataset.sig = sig;
      nxEl.innerHTML = `<div class="t"><b>${t(nx.t)}</b><span>${t(nx.d, {ip: esc(ip)})}</span></div>` +
        (nx.btn2 ? `<button class="btn" data-goto="${nx.go2}">${t(nx.btn2)}</button>` : "") +
        (nx.btn ? `<button class="btn primary" data-goto="${nx.go}">${t(nx.btn)}</button>` : "");
    }
  } else nxEl.hidden = true;

  // quick outputs list
  const q = [];
  q.push(`<div class="qrow"><div class="name"><b>${esc(t("out_obs"))}</b><span>${esc(st.viewers > 0 ? t("q_obs_sub", {n: st.viewers}) : t("q_obs_none"))}</span></div><button class="btn small" data-goto="obs">${esc(t("q_setup"))}</button></div>`);
  s.destinations.forEach(d => q.push(`<div class="qrow"><div class="name"><b><span class="pf ${esc(d.platform)}">${esc(d.platform === "custom" ? "RTMP" : d.name)}</span> ${d.platform === "custom" ? esc(d.name) : ""}</b><span>${d.enabled ? esc(t("st_" + (d.state || "idle"))) : esc(t("st_off"))}</span></div><label class="switch"><input type="checkbox" ${d.enabled ? "checked" : ""} data-toggle-dest="${esc(d.id)}"><span></span></label></div>`));
  q.push(`<div class="qrow"><div class="name"><b><span class="pf twitch">Twitch</span> ${esc(t("fwd_t"))}</b><span>${s.settings.forwardToTwitch ? esc(t("st_" + (s.twitchForward.state || "idle"))) : esc(t("st_off"))}</span></div><label class="switch"><input type="checkbox" ${s.settings.forwardToTwitch ? "checked" : ""} data-toggle-fwd><span></span></label></div>`);
  if (!s.destinations.length) q.push(`<div class="qrow"><div class="name"><span>${esc(t("q_none"))}</span></div><button class="btn small" data-goto="platforms">${esc(t("q_add"))}</button></div>`);
  const qsig = q.join("");
  if ($("qlist").dataset.sig !== qsig && !$("qlist").contains(document.activeElement)) { $("qlist").innerHTML = qsig; $("qlist").dataset.sig = qsig; }
  if (!previewPlayer) $("pv-note").textContent = t("pv_wait");

  // ----- setup page
  $("sn1").classList.toggle("done", psFound); $("sn2").classList.toggle("done", st.live);
  $("ck1").textContent = psFound ? "✓ " + t("ck_ok1") : t("ck_wait1"); $("ck1").classList.toggle("ok", psFound);
  $("ck2").textContent = st.live ? "✓ " + t("ck_ok2") : t("ck_wait2"); $("ck2").classList.toggle("ok", st.live);
  const sel = $("ip-select");
  if ((s.ips || []).length > 1 && !s.isRemote) {
    $("ip-picker").hidden = false;
    const sig = s.ips.map(x => x.ip).join(",") + "|" + s.activeIP;
    if (sel.dataset.sig !== sig) { sel.innerHTML = s.ips.map(x => `<option value="${esc(x.ip)}" ${x.ip === s.activeIP ? "selected" : ""}>${esc(x.ip)} — ${esc(x.iface)}</option>`).join(""); sel.dataset.sig = sig; }
  } else $("ip-picker").hidden = true;

  // ----- platforms page
  const fwd = $("fwd-twitch"); if (document.activeElement !== fwd) fwd.checked = !!s.settings.forwardToTwitch;
  $("fwd-state").innerHTML = s.settings.forwardToTwitch ? stateTag(s.twitchForward.state, s.twitchForward.error) : "";
  renderDests(s);

  // ----- settings page
  $("set-local").hidden = !!s.isRemote; $("set-phone-note").hidden = !s.isRemote;
  if (!s.isRemote) {
    $("auto-row").hidden = !s.autostart.supported;
    const at = $("auto-toggle"); if (document.activeElement !== at) at.checked = !!s.autostart.enabled;
    const rt = $("remote-toggle"); if (document.activeElement !== rt) rt.checked = !!s.remote.enabled;
    const on = s.remote.enabled && s.remote.running && s.remote.url;
    $("remote-box").hidden = !on;
    $("remote-err").textContent = s.remote.error ? t("rem_err_" + s.remote.error) : "";
    if (on) {
      $("rem-url").textContent = s.remote.url; $("rem-pin").textContent = s.remote.pin || "";
      const qsig = s.remote.url + "|" + s.remote.pin;
      if (qsig !== lastQR) { lastQR = qsig; $("qr").src = "/api/qr.png?v=" + encodeURIComponent(qsig); }
    }
  }

  // ----- Iran mode
  if (!s.isRemote) {
    const tt = $("tun-toggle"); if (document.activeElement !== tt) tt.checked = !!tn.enabled;
    $("tun-box").hidden = !tn.enabled;
    const tp = $("tun-proxy"); if (document.activeElement !== tp && tp.dataset.dirty !== "1") tp.value = tn.proxy || "";
    const tr = $("tun-relays"); if (document.activeElement !== tr) tr.checked = !tn.skipRelays;
    const te = $("tun-extra"); if (document.activeElement !== te && te.dataset.dirty !== "1") te.value = (tn.extra || []).join("\n");
    $("tun-err").innerHTML = tn.enabled && tn.err443 ? (t("tun_err_" + tn.err443) !== "tun_err_" + tn.err443 ? t("tun_err_" + tn.err443) : esc(tn.err443)) : "";
    const kind = k => k === "socks5" ? "SOCKS" : k === "http" ? "HTTP" : k;
    const lines = [];
    if (tn.enabled && tn.running) {
      lines.push(tn.proxyUp ? `<span class="ok">✓ ${t("tun_up", {addr: esc(tn.proxy), kind: kind(tn.proxyKind)})}</span>` : `<span class="bad">${t("tun_down", {addr: esc(tn.proxy || "—")})}</span>`);
      if (tn.total > 0) lines.push(`<span class="dim">${t("tun_stats", {n: tn.total, host: esc(tn.lastHost)})}</span>`);
      if (tn.lastErr) lines.push(`<span class="bad">${t("tun_lasterr", {e: esc(tn.lastErr)})}</span>`);
    }
    $("tun-status").innerHTML = lines.join("");
  }
  $("iran-block").hidden = !!s.isRemote;

  // ----- help page
  $("fw-wrap").hidden = !s.firewallFix;
  if (page === "help") $("dev-body").innerHTML = s.devices.map(d => `<tr><td>${esc(t("k_" + d.kind))}</td><td class="mono">${esc(d.ip)}</td><td>${esc(fmtAgo(d.lastSeenAgo))}</td><td>${d.queries}</td></tr>`).join("");
}

function renderDests(s) {
  const sig = JSON.stringify(s.destinations) + lang;
  if (sig === lastDestSig) return; lastDestSig = sig;
  const el = $("dest-list");
  if (!s.destinations.length) { el.innerHTML = `<p class="note">${esc(t("dest_empty"))}</p>`; return; }
  el.innerHTML = s.destinations.map(d => `
    <div class="dest">
      <label class="switch"><input type="checkbox" ${d.enabled ? "checked" : ""} data-toggle-dest="${esc(d.id)}"><span></span></label>
      <div class="meta">
        <div><span class="pf ${esc(d.platform)}">${esc(d.platform === "custom" ? "RTMP" : d.name)}</span> ${d.platform === "custom" ? esc(d.name) : ""}</div>
        <div class="sub">${esc(d.url)}  ${esc(d.keyMasked)}</div>
        ${d.enabled ? stateTag(d.state, d.error) : `<span class="tag">${esc(t("st_off"))}</span>`}
      </div>
      <button class="btn small" data-del-dest="${esc(d.id)}" data-name="${esc(d.name)}">${esc(t("del"))}</button>
    </div>`).join("");
}

async function loadLog() {
  if (page !== "help") { clearTimeout(logTimer); return; }
  try {
    const rows = await (await fetch("/api/dnslog")).json();
    $("log-body").innerHTML = rows.map(r => `<tr><td>${esc(new Date(r.time).toLocaleTimeString())}</td><td class="mono">${esc(r.client)}</td><td class="mono">${esc(r.name)} <span style="color:var(--faint)">${esc(r.type)}</span></td><td>${r.tunnel ? `<span class="tag-r">${esc(t("redirected_tun"))}</span>` : r.hijacked ? `<span class="tag-r">${t("redirected", {ip: esc(r.answer || "∅")})}</span>` : ""}</td></tr>`).join("");
  } catch (e) {}
  clearTimeout(logTimer); logTimer = setTimeout(loadLog, 3000);
}

// ---------- actions
const PLATFORM_URL = {kick: "", youtube: "rtmp://a.rtmp.youtube.com/live2", twitch: "rtmp://live.twitch.tv/app", custom: ""};
const PLATFORM_PH = {kick: "rtmps://xxxxxxxx.global-contribute.live-video.net", youtube: "", twitch: "", custom: "rtmp://server/app"};
function onPlatformChange(keep) {
  const p = $("f-platform").value, u = $("f-url");
  if (!keep || !u.value || Object.values(PLATFORM_URL).includes(u.value)) u.value = PLATFORM_URL[p];
  u.placeholder = PLATFORM_PH[p] || "";
  $("f-help").innerHTML = t("help_" + p);
  $("f-key").placeholder = p === "kick" ? "sk_us-west-2_…" : "";
}
async function addDest() {
  const body = {platform: $("f-platform").value, url: $("f-url").value.trim(), key: $("f-key").value.trim(), enabled: true};
  if (!body.url || !body.key) { $("f-err").textContent = t("f_missing"); return; }
  $("f-err").textContent = "";
  const r = await post("/api/destinations", body);
  if (r.ok) { $("f-key").value = ""; if (body.platform === "kick" || body.platform === "custom") $("f-url").value = ""; poll(); }
  else $("f-err").textContent = t("f_missing");
}
async function setForward(on) { await post("/api/settings", {forwardToTwitch: on}); poll(); }
async function setPreferredIP(ip) { await post("/api/settings", {preferredIP: ip}); poll(); }
async function setAutostart(on) { await post("/api/autostart", {enabled: on}); poll(); }
async function setRemote(on) { await post("/api/remote", {enabled: on}); setTimeout(poll, 300); }
async function resetRemote() { await post("/api/remote/reset", {}); lastQR = ""; poll(); }
async function fixFirewall() {
  $("fw-msg").textContent = "…";
  const r = await post("/api/firewall", {});
  $("fw-msg").textContent = r.ok ? t("fw_ok") : t("fw_err", {e: r.error || ""});
}
async function importTo(target, withSecret) {
  impTarget = target;
  const msg = $("imp-msg"); msg.className = "imp-msg"; msg.textContent = t("imp_wait");
  const body = {};
  if (withSecret) body[target === "obs" ? "password" : "token"] = $("imp-secret").value.trim();
  let r;
  try { r = await post("/api/import/" + target, body); } catch (e) { r = {ok: false, code: "failed", detail: String(e)}; }
  const box = $("imp-box");
  const showGuide = g => { obsGuide = g; gState[g] = 0; renderGuides(); };
  if (r.ok) { box.hidden = true; $("imp-secret").value = ""; msg.className = "imp-msg ok"; msg.textContent = t("imp_ok", {scene: r.scene || ""}); return; }
  if (r.code === "need_password" || r.code === "bad_password") {
    box.hidden = false;
    $("imp-help").innerHTML = (r.code === "bad_password" && withSecret ? t("imp_bad") + " " : "") + t(target === "obs" ? "imp_obs_help" : "imp_sl_help");
    showGuide(target === "obs" ? "obsws" : "slremote");
    if (target === "obs") gState.obsws = 3, renderGuides();
    else gState.slremote = 3, renderGuides();
    $("imp-secret").focus(); msg.textContent = ""; return;
  }
  msg.className = "imp-msg err";
  if (r.code === "not_running") { msg.innerHTML = t(target === "obs" ? "imp_obs_off" : "imp_sl_off"); showGuide(target === "obs" ? "obsws" : "slremote"); }
  else msg.textContent = t("imp_fail", {d: r.detail || r.code});
}
async function setTunnel(body) { await post("/api/tunnel", body); setTimeout(poll, 300); }
async function saveProxy() { const el = $("tun-proxy"); el.dataset.dirty = ""; await setTunnel({proxy: el.value}); testTunnelNow(); }
async function saveExtra() { const el = $("tun-extra"); el.dataset.dirty = ""; await setTunnel({extra: el.value.split(/\s+/).filter(Boolean)}); }
const proxyKindName = k => k === "socks5" ? "SOCKS" : k === "http" ? "HTTP" : k;
async function detectProxy() {
  const msg = $("tun-msg"); msg.className = "imp-msg"; msg.textContent = t("tun_wait");
  const r = await post("/api/tunnel/detect", {});
  if (!r.found || !r.found.length) { msg.className = "imp-msg err"; msg.textContent = t("tun_none"); return; }
  const f = r.found[0];
  $("tun-proxy").value = f.addr; $("tun-proxy").dataset.dirty = "";
  await post("/api/tunnel", {proxy: f.addr});
  if (r.twitchOK) { msg.className = "imp-msg ok"; msg.innerHTML = t("tun_found", {addr: esc(f.addr), kind: proxyKindName(f.kind), ms: r.ms}); }
  else { msg.className = "imp-msg err"; msg.innerHTML = t("tun_found_fail", {addr: esc(f.addr), kind: proxyKindName(f.kind), e: esc(r.error || "")}); }
  poll();
}
async function testTunnelNow() {
  const msg = $("tun-msg"); msg.className = "imp-msg"; msg.textContent = t("tun_wait");
  const r = await post("/api/tunnel/test", {proxy: $("tun-proxy").value});
  if (r.ok) { msg.className = "imp-msg ok"; msg.textContent = t("tun_test_ok", {ms: r.ms}); }
  else { msg.className = "imp-msg err"; msg.textContent = t("tun_test_fail", {e: r.error || ""}); }
}
document.addEventListener("input", e => { if (e.target.id === "tun-proxy" || e.target.id === "tun-extra") e.target.dataset.dirty = "1"; });

async function stopApp() {
  if (!confirm(t("stopConfirm"))) return;
  try { await post("/api/quit", {}); } catch (e) {}
  stopped = true; showOverlay(t("stopped"));
}
function showOverlay(msg) { $("overlay-msg").textContent = msg; $("overlay").style.display = "flex"; }
function hideOverlay() { $("overlay").style.display = "none"; }

function loadScript(src) { return new Promise((res, rej) => { const s = document.createElement("script"); s.src = src; s.onload = res; s.onerror = rej; document.head.appendChild(s); }); }
async function togglePreview() {
  const v = $("pv"), btn = $("pv-btn");
  if (previewPlayer) {
    try { previewPlayer.destroy(); } catch (e) {}
    previewPlayer = null; v.style.display = "none"; $("pv-note").hidden = false; btn.dataset.i18n = "pv_show"; btn.textContent = t("pv_show"); return;
  }
  try {
    if (!window.mpegts) await loadScript("https://cdn.jsdelivr.net/npm/mpegts.js@1.7.3/dist/mpegts.min.js");
    previewPlayer = mpegts.createPlayer({type: "flv", isLive: true, url: location.origin + "/live.flv"}, {enableStashBuffer: false, liveBufferLatencyChasing: true});
    previewPlayer.attachMediaElement(v); previewPlayer.load(); v.style.display = "block"; $("pv-note").hidden = true;
    previewPlayer.play().catch(() => {});
    btn.dataset.i18n = "pv_hide"; btn.textContent = t("pv_hide");
  } catch (e) { $("pv-note").textContent = t("pv_err"); previewPlayer = null; }
}

// ---------- events
document.addEventListener("click", async e => {
  const nav = e.target.closest("[data-page]"); if (nav) { showPage(nav.dataset.page); return; }
  const go = e.target.closest("[data-goto]"); if (go) { showPage(go.dataset.goto); return; }
  const lb = e.target.closest("[data-lang]"); if (lb) { setLang(lb.dataset.lang); return; }
  const gt = e.target.closest("#obs-tabs [data-g]"); if (gt) { obsGuide = gt.dataset.g; renderGuides(); return; }
  const gs = e.target.closest("[data-gstep],[data-gdot]");
  if (gs) {
    const g = gs.closest("[data-gid]"), id = g.dataset.gid, n = GUIDES[id].length;
    gState[id] = gs.dataset.gdot !== undefined ? +gs.dataset.gdot : Math.min(n - 1, Math.max(0, (gState[id] || 0) + +gs.dataset.gstep));
    renderGuide(g, id, guideCtx());
    if (g.id === "obs-guide") $("manual-url").hidden = !GUIDES[id][gState[id]].url;
    return;
  }
  const del = e.target.closest("[data-del-dest]");
  if (del) { if (confirm(t("del_confirm", {name: del.dataset.name}))) { await post("/api/destinations/delete", {id: del.dataset.delDest}); poll(); } return; }
  const cp = e.target.closest("[data-copy]");
  if (cp) {
    const txt = cp.dataset.copy === "ip" ? ((S && S.activeIP) || "") : ((S && S.obsURL) || "");
    const done = () => { const o = cp.textContent; cp.textContent = t("copied"); setTimeout(() => cp.textContent = o, 1400); };
    if (navigator.clipboard) navigator.clipboard.writeText(txt).then(done, () => fallbackCopy(txt, done)); else fallbackCopy(txt, done);
  }
});
document.addEventListener("change", async e => {
  const td = e.target.closest("[data-toggle-dest]"); if (td) { await post("/api/destinations", {id: td.dataset.toggleDest, enabled: td.checked}); $("qlist").dataset.sig = ""; poll(); return; }
  const tf = e.target.closest("[data-toggle-fwd]"); if (tf) { await setForward(tf.checked); $("qlist").dataset.sig = ""; }
});
document.addEventListener("keydown", e => {
  if (e.key !== "ArrowLeft" && e.key !== "ArrowRight") return;
  const g = e.target.closest && e.target.closest("[data-gid]"); if (!g) return;
  const dir = (e.key === "ArrowRight") !== (lang === "fa") ? 1 : -1;
  const id = g.dataset.gid; gState[id] = Math.min(GUIDES[id].length - 1, Math.max(0, (gState[id] || 0) + dir));
  renderGuide(g, id, guideCtx()); g.querySelector("[data-gstep='" + dir + "']")?.focus();
});
function fallbackCopy(txt, done) { const ta = document.createElement("textarea"); ta.value = txt; document.body.appendChild(ta); ta.select(); try { document.execCommand("copy"); done(); } catch (e) {} ta.remove(); }
window.addEventListener("hashchange", () => showPage(location.hash.slice(1), false));

let failCount = 0, firstLoad = true;
async function poll() {
  if (stopped) return;
  try {
    const resp = await fetch("/api/status", {cache: "no-store"});
    if (resp.status === 401) { location.href = "/login"; return; }
    const s = await resp.json();
    failCount = 0; hideOverlay();
    S = s;
    if (firstLoad) {
      firstLoad = false;
      lang = s.settings.lang || ((navigator.language || "").startsWith("fa") ? "fa" : "en");
      applyLang();
    }
    render(s);
  } catch (e) { if (++failCount >= 3) showOverlay(t("offline")); }
}

buildNav();
showPage(location.hash.slice(1) || "dashboard", false);
applyLang();
poll(); setInterval(poll, 1000);
