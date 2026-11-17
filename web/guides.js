// Picture guides: real OBS screenshots and simplified drawings of the PS5,
// OBS and Streamlabs screens, with numbered arrows on the exact thing to press.

const gEsc = s => String(s ?? "").replace(/[&<>"]/g, c => ({"&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;"}[c]));
const FONT = "IBM Plex Sans, Segoe UI, Roboto, sans-serif";
function T(x, y, s, o = {}) {
  return `<text x="${x}" y="${y}" font-family="${FONT}" font-size="${o.size || 30}" font-weight="${o.w || 400}" fill="${o.fill || "#E8EDF7"}" text-anchor="${o.anchor || "start"}"${o.op ? ` opacity="${o.op}"` : ""}>${gEsc(s)}</text>`;
}
function R(x, y, w, h, o = {}) {
  return `<rect x="${x}" y="${y}" width="${w}" height="${h}" rx="${o.rx ?? 10}" fill="${o.fill || "none"}"${o.stroke ? ` stroke="${o.stroke}" stroke-width="${o.sw || 2}"` : ""}${o.op ? ` opacity="${o.op}"` : ""}/>`;
}
function C(cx, cy, r, o = {}) {
  return `<circle cx="${cx}" cy="${cy}" r="${r}" fill="${o.fill || "none"}"${o.stroke ? ` stroke="${o.stroke}" stroke-width="${o.sw || 2}"` : ""}${o.op ? ` opacity="${o.op}"` : ""}/>`;
}
function svg(w, h, bg, inner) {
  return `<svg class="g-img" viewBox="0 0 ${w} ${h}" xmlns="http://www.w3.org/2000/svg" style="direction:ltr" aria-hidden="true">${R(0, 0, w, h, {rx: 0, fill: bg})}${inner}</svg>`;
}

// ---------- PS5-style screens (simplified, not real screenshots)
const P = {bg: "#081026", item: "#101A35", focus: "#1C2B52", text: "#E8EDF7", faint: "#8C98B8"};
const IH = 84; // item height
function psItem(x, y, w, label, o = {}) {
  let s = "";
  if (o.focus) s += R(x, y, w, IH, {rx: 14, fill: P.focus, stroke: "#FFFFFF", sw: 3});
  else if (o.sel) s += R(x, y, w, IH, {rx: 14, fill: P.item});
  s += T(x + 30, y + 55, label, {size: 36, w: o.focus || o.sel ? 600 : 400});
  if (o.value) s += T(x + w - 30, y + 55, o.value, {size: 36, anchor: "end", fill: o.vfill || P.faint, w: o.vw || 400});
  return s;
}
function psTitle(s) { return T(90, 112, s, {size: 56, w: 600}); }
function gear(cx, cy, r, color) {
  let s = C(cx, cy, r, {stroke: color, sw: 6}) + C(cx, cy, r * 0.35, {fill: color});
  for (let k = 0; k < 8; k++) {
    const a = k * Math.PI / 4, x1 = cx + Math.cos(a) * r, y1 = cy + Math.sin(a) * r, x2 = cx + Math.cos(a) * (r + 9), y2 = cy + Math.sin(a) * (r + 9);
    s += `<line x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}" stroke="${color}" stroke-width="7" stroke-linecap="round"/>`;
  }
  return s;
}

const MOCK = {
  ps5Home() {
    let s = T(90, 80, "Games", {size: 38, w: 600}) + T(250, 80, "Media", {size: 38, op: 0.55});
    s += C(1230, 66, 19, {stroke: P.text, sw: 5}) + `<line x1="1244" y1="80" x2="1259" y2="95" stroke="${P.text}" stroke-width="5" stroke-linecap="round"/>`;
    s += gear(1335, 66, 21, P.text);
    s += C(1430, 66, 24, {fill: "#4D6AA8"}) + T(1505, 79, "9:41", {size: 32});
    s += R(90, 150, 210, 210, {rx: 18, fill: "#22355F", stroke: "#FFFFFF", sw: 3});
    for (let k = 0; k < 7; k++) s += R(330 + k * 176, 160, 150, 150, {rx: 16, fill: "#16223F"});
    s += R(90, 410, 1420, 420, {rx: 20, fill: "#0E1A38"});
    s += T(140, 780, "Your game", {size: 60, w: 600, op: 0.8});
    return svg(1600, 900, P.bg, s);
  },
  ps5Settings() {
    const items = ["Accessibility", "Network", "Users and Accounts", "System", "Storage", "Sound", "Screen and Video", "Captures and Broadcasts"];
    let s = psTitle("Settings");
    items.forEach((it, k) => { s += C(125, 170 + k * 88 + 42, 10, {fill: P.faint, op: 0.6}) + psItem(150, 170 + k * 88, 780, it, {focus: k === 1}); });
    s += R(990, 170, 520, 690, {rx: 18, fill: "#0D1733"}) + T(1030, 235, "Network", {size: 36, w: 600, op: 0.7});
    return svg(1600, 900, P.bg, s);
  },
  ps5Network() {
    let s = psTitle("Network");
    s += psItem(90, 180, 440, "Connection Status") + psItem(90, 272, 440, "Settings", {focus: true});
    s += psItem(580, 180, 930, "Internet Connected", {value: "On", vfill: "#7FD1A6"});
    s += psItem(580, 272, 930, "Set Up Internet Connection", {sel: true});
    s += psItem(580, 364, 930, "Test Internet Connection");
    return svg(1600, 900, P.bg, s);
  },
  ps5Connections() {
    let s = psTitle("Set Up Internet Connection");
    s += T(90, 172, "Registered Networks", {size: 28, fill: P.faint});
    s += psItem(90, 190, 760, "Home Wi-Fi", {focus: true, value: "Connected"});
    s += psItem(90, 282, 760, "Wired LAN");
    s += T(90, 410, "Networks Found", {size: 28, fill: P.faint});
    s += psItem(90, 428, 760, "Neighbor-5G");
    s += R(895, 172, 520, 302, {rx: 18, fill: "#14204A", stroke: "#33447A", sw: 2});
    s += psItem(910, 188, 490, "Connect") + psItem(910, 280, 490, "Advanced Settings", {focus: true}) + psItem(910, 372, 490, "Forget");
    s += R(90, 770, 70, 70, {rx: 16, fill: "#1A2546", stroke: "#8C98B8", sw: 2});
    for (let k = 0; k < 3; k++) s += `<line x1="108" y1="${791 + k * 14}" x2="142" y2="${791 + k * 14}" stroke="#E8EDF7" stroke-width="4" stroke-linecap="round"/>`;
    s += T(185, 817, "Options", {size: 36, w: 600});
    return svg(1600, 900, P.bg, s);
  },
  ps5Advanced(ip) {
    let s = psTitle("Advanced Settings");
    const rows = [["DNS Settings", "Manual", true], ["Primary DNS", ip || "—", false], ["Secondary DNS", "1.1.1.1", false], ["MTU Settings", "Automatic"], ["Proxy Server", "Do Not Use"]];
    rows.forEach((r, k) => { s += psItem(90, 170 + k * 96, 1420, r[0], {focus: r[2], value: r[1], vfill: k < 3 ? "#FFFFFF" : P.faint, vw: k < 3 ? 600 : 400}); });
    s += R(1270, 700, 240, 84, {rx: 42, fill: "#FFFFFF"}) + T(1390, 755, "OK", {size: 36, w: 600, fill: "#0A1430", anchor: "middle"});
    return svg(1600, 900, P.bg, s);
  },
  ps5Link() {
    let s = psTitle("Users and Accounts");
    s += psItem(90, 180, 440, "Account") + psItem(90, 272, 440, "Linked Services", {sel: true});
    s += psItem(580, 180, 930, "Twitch", {focus: true, value: "Link", vfill: "#FFFFFF", vw: 600});
    s += psItem(580, 272, 930, "YouTube", {value: "Link"});
    return svg(1600, 900, P.bg, s);
  },
  ps5Quality() {
    let s = psTitle("Captures and Broadcasts");
    s += psItem(90, 180, 440, "Captures") + psItem(90, 272, 440, "Broadcasts", {sel: true});
    s += psItem(580, 180, 930, "Video Quality", {focus: true, value: "1920 × 1080 at 60 fps", vfill: "#FFFFFF", vw: 600});
    s += psItem(580, 272, 930, "Include Party Voice", {value: "Off"});
    s += psItem(580, 364, 930, "Display Camera", {value: "Off"});
    return svg(1600, 900, P.bg, s);
  },
  controller() {
    const body = "M560 300 Q600 230 700 230 L900 230 Q1000 230 1040 300 L1150 610 Q1180 720 1100 735 Q1030 750 975 640 L940 585 L660 585 L625 640 Q570 750 500 735 Q420 720 450 610 Z";
    let s = `<path d="${body}" fill="#DCE2EC" stroke="#9AA6BC" stroke-width="3"/>`;
    s += R(705, 255, 190, 120, {rx: 16, fill: "#B9C2D2"});
    s += R(620, 270, 46, 22, {rx: 11, fill: "#5A6680"}) + R(934, 270, 46, 22, {rx: 11, fill: "#5A6680"});
    s += `<path d="M585 385h30v-30h30v30h30v30h-30v30h-30v-30h-30z" fill="#5A6680"/>`;
    [[1010, 360], [1050, 400], [1010, 440], [970, 400]].forEach(([x, y]) => { s += C(x, y, 18, {fill: "#5A6680"}); });
    s += C(705, 480, 38, {fill: "#3F4A60"}) + C(895, 480, 38, {fill: "#3F4A60"});
    s += T(520, 205, "Create", {size: 38, w: 600, anchor: "middle"}) + T(1020, 200, "Options", {size: 38, w: 600, anchor: "middle", op: 0.55});
    return svg(1600, 900, "#0E1424", s);
  },
  ps5Create() {
    let s = C(1200, 250, 160, {fill: "#22304F"}) + R(200, 290, 520, 220, {rx: 30, fill: "#1F2B47"});
    s += R(0, 600, 1600, 300, {rx: 0, fill: "#0B1226"});
    [["Screenshot", 300], ["Video Clip", 650], ["Broadcast", 1000]].forEach(([l, x], k) => {
      s += R(x, 640, 300, 200, {rx: 18, fill: k === 2 ? P.focus : "#141E3B", stroke: k === 2 ? "#FFFFFF" : null, sw: 3});
      s += (k === 2 ? C(x + 150, 706, 36, {stroke: "#E8EDF7", sw: 6}) + C(x + 150, 706, 11, {fill: "#E8EDF7"}) : R(x + 114, 678, 72, 56, {rx: 8, stroke: "#E8EDF7", sw: 5}));
      s += T(x + 150, 800, l, {size: 34, w: k === 2 ? 600 : 400, anchor: "middle"});
    });
    return svg(1600, 900, "#1A2338", s);
  },
  ps5Broadcast() {
    let s = R(150, 220, 560, 300, {rx: 30, fill: "#1F2B47"});
    s += R(860, 70, 680, 780, {rx: 22, fill: "#0E1730", stroke: "#2A3A66", sw: 2});
    s += T(910, 145, "Broadcast", {size: 46, w: 600}) + T(910, 215, "Platform", {size: 28, fill: P.faint});
    s += psItem(895, 235, 610, "Twitch", {focus: true}) + C(1465, 277, 14, {fill: "#FFFFFF"});
    s += psItem(895, 327, 610, "YouTube") + C(1465, 369, 14, {stroke: P.faint, sw: 3});
    s += T(910, 470, "Title", {size: 28, fill: P.faint}) + R(910, 490, 580, 76, {rx: 12, fill: "#16213F"}) + T(940, 540, "My stream", {size: 34});
    s += R(910, 700, 580, 92, {rx: 46, fill: "#FFFFFF"}) + T(1200, 759, "Go Live", {size: 38, w: 600, fill: "#0A1430", anchor: "middle"});
    return svg(1600, 900, "#1A2338", s);
  },

  // ---------- OBS-style screens
  obsToolsMenu() {
    let s = R(0, 0, 1600, 70, {rx: 0, fill: "#24272F"});
    [["File", 30], ["Edit", 125], ["View", 225], ["Docks", 335], ["Profile", 465], ["Scene Collection", 600], ["Tools", 890], ["Help", 1015]].forEach(([l, x]) => {
      if (l === "Tools") s += R(x - 16, 10, 118, 50, {rx: 6, fill: "#3A3F4B"});
      s += T(x, 47, l, {size: 32});
    });
    s += R(874, 70, 640, 430, {rx: 8, fill: "#2B2E37", stroke: "#3A3F4B", sw: 2});
    let y = 86;
    ["Auto-Configuration Wizard", "Output Timer", "Scripts", null, "WebSocket Server Settings", "Captions (Experimental)"].forEach(l => {
      if (!l) { s += `<line x1="890" y1="${y + 9}" x2="1498" y2="${y + 9}" stroke="#3A3F4B" stroke-width="2"/>`; y += 18; return; }
      if (l === "WebSocket Server Settings") s += R(884, y, 620, 70, {rx: 6, fill: "#2D5BD0"});
      s += T(915, y + 47, l, {size: 32}); y += 76;
    });
    s += R(40, 560, 780, 300, {rx: 10, fill: "#14161C"});
    return svg(1600, 900, "#1C1E25", s);
  },
  obsWsDialog() {
    let s = R(230, 30, 1140, 840, {rx: 12, fill: "#2A2D36", stroke: "#3A3F4B", sw: 2}) + R(230, 30, 1140, 70, {rx: 12, fill: "#22252D"}) + T(265, 77, "WebSocket Server Settings", {size: 30});
    const cb = (x, y, label, on) => R(x, y - 28, 34, 34, {rx: 6, fill: on ? "#2D5BD0" : "#1C1E25", stroke: "#5A6070", sw: 2}) + (on ? `<path d="M${x + 8} ${y - 11} l8 8 l13 -16" fill="none" stroke="#fff" stroke-width="4" stroke-linecap="round" stroke-linejoin="round"/>` : "") + T(x + 52, y, label, {size: 30});
    s += T(270, 150, "Plugin Settings", {size: 27, fill: "#9AA3B5"}) + cb(300, 205, "Enable System Tray Alerts", false) + cb(300, 260, "Enable Debug Logging", false);
    s += T(270, 335, "Server Settings", {size: 27, fill: "#9AA3B5"}) + cb(300, 395, "Enable WebSocket server", true);
    s += T(300, 458, "Server Port", {size: 30}) + R(620, 420, 220, 52, {rx: 6, fill: "#1C1E25"}) + T(642, 457, "4455", {size: 30});
    s += cb(300, 522, "Enable Authentication", true);
    s += T(300, 588, "Server Password", {size: 30}) + R(620, 550, 380, 52, {rx: 6, fill: "#1C1E25"}) + T(642, 587, "••••••••••••", {size: 30}) + R(1015, 550, 320, 52, {rx: 6, fill: "#3A3F4B"}) + T(1175, 586, "Generate Password", {size: 28, anchor: "middle"});
    s += R(300, 630, 350, 60, {rx: 6, fill: "#3A3F4B"}) + T(475, 670, "Show Connect Info", {size: 29, anchor: "middle"});
    [["OK", 940], ["Cancel", 1100], ["Apply", 1260]].forEach(([l, x]) => { s += R(x - 70, 790, 140, 58, {rx: 6, fill: "#3A3F4B"}) + T(x, 829, l, {size: 29, anchor: "middle"}); });
    return svg(1600, 900, "#16181E", s);
  },
  obsWsInfo() {
    let s = R(300, 110, 1000, 680, {rx: 12, fill: "#2A2D36", stroke: "#3A3F4B", sw: 2}) + R(300, 110, 1000, 70, {rx: 12, fill: "#22252D"}) + T(335, 157, "WebSocket Connect Info", {size: 30});
    const row = (y, label, val) => T(350, y, label, {size: 30}) + R(620, y - 38, 400, 54, {rx: 6, fill: "#1C1E25"}) + T(642, y, val, {size: 30}) + R(1040, y - 38, 200, 54, {rx: 6, fill: "#3A3F4B"}) + T(1140, y - 2, "Copy", {size: 29, anchor: "middle"});
    s += row(270, "Server IP", "192.168.1.20") + row(355, "Server Port", "4455") + row(440, "Server Password", "••••••••••••");
    s += R(350, 510, 220, 220, {rx: 8, fill: "#E8EAF0", op: 0.9});
    return svg(1600, 900, "#16181E", s);
  },

  // ---------- Streamlabs-style screens
  slMain() {
    let s = R(0, 0, 100, 900, {rx: 0, fill: "#0F1F28"});
    for (let k = 0; k < 5; k++) s += R(27, 40 + k * 88, 46, 46, {rx: 10, fill: k === 0 ? "#31C3A2" : "#22343F"});
    s += gear(50, 834, 17, "#C9D6DC");
    s += R(135, 30, 1435, 520, {rx: 10, fill: "#04090C"}) + T(850, 300, "Editor", {size: 46, fill: "#2E4450", anchor: "middle"});
    [["Scenes", 135], ["Sources", 625], ["Mixer", 1115]].forEach(([l, x]) => { s += R(x, 580, 455, 290, {rx: 10, fill: "#17262F"}) + T(x + 26, 630, l, {size: 32, w: 600}); });
    return svg(1600, 900, "#09161D", s);
  },
  slSettings(sel = "Remote Control") {
    let s = R(170, 40, 1260, 820, {rx: 14, fill: "#17262F", stroke: "#25404C", sw: 2}) + T(210, 105, "Settings", {size: 40, w: 600});
    ["General", "Multistream", "Stream", "Output", "Audio", "Video", "Hotkeys", "Game Overlay", "Scene Collections", "Notifications", "Appearance", "Remote Control", "Installed Apps"].forEach((l, k) => {
      const y = 135 + k * 54;
      if (l === sel) s += R(190, y, 350, 48, {rx: 8, fill: "#25404C"});
      s += T(215, y + 35, l, {size: 28, w: l === sel ? 600 : 400, fill: l === sel ? "#FFFFFF" : "#B7C7CE"});
    });
    s += `<line x1="565" y1="130" x2="565" y2="840" stroke="#25404C" stroke-width="2"/>`;
    return s;
  },
  slRemote() {
    let s = MOCK.slSettings() + T(610, 180, "Remote Control", {size: 38, w: 600});
    s += T(610, 232, "Control Streamlabs from your phone or other apps.", {size: 27, fill: "#B7C7CE"});
    s += R(610, 270, 290, 290, {rx: 12, fill: "#E8EEF0"});
    for (let i = 0; i < 9; i++) for (let j = 0; j < 9; j++) if ((i * 7 + j * 3) % 4 === 0) s += R(632 + i * 28, 292 + j * 28, 24, 24, {rx: 3, fill: "#3B4A52", op: 0.35});
    s += R(610, 270, 290, 290, {rx: 12, fill: "#17262F", op: 0.55}) + T(755, 425, "Click to show", {size: 29, w: 600, anchor: "middle"});
    s += T(612, 630, "Show details", {size: 32, w: 600, fill: "#31C3A2"});
    return svg(1600, 900, "#09161D", s);
  },
  slToken() {
    let s = MOCK.slSettings() + T(610, 180, "Remote Control", {size: 38, w: 600});
    s += T(610, 270, "API Token", {size: 30, fill: "#B7C7CE"}) + R(610, 292, 520, 62, {rx: 8, fill: "#0F1F28"}) + T(635, 334, "••••••••••••••••••", {size: 30}) + R(1150, 292, 170, 62, {rx: 8, fill: "#31C3A2"}) + T(1235, 334, "Copy", {size: 30, w: 600, fill: "#062018", anchor: "middle"});
    s += T(610, 430, "Port", {size: 30, fill: "#B7C7CE"}) + R(610, 452, 220, 62, {rx: 8, fill: "#0F1F28"}) + T(635, 494, "59650", {size: 30});
    return svg(1600, 900, "#09161D", s);
  }
};

// ---------- guide definitions. Marks are in image pixels. A mark is either a
// ring {x, y} (small icons) or a highlight box {box: [x, y, w, h]} (buttons,
// rows, fields). a = direction in degrees from the target to the arrow's tail;
// len scales the arrow for crowded images.
const GUIDES = {
  ps5dns: [
    {w: 896, h: 504, img: "guide/ps5-home.jpg", marks: [{x: 722, y: 29, a: 135}], cap: "g_dns1"},
    {w: 1600, h: 900, svg: () => MOCK.ps5Settings(), marks: [{box: [150, 258, 780, 84], a: 10}], cap: "g_dns2"},
    {w: 1600, h: 900, svg: () => MOCK.ps5Network(), marks: [{box: [90, 272, 440, 84], a: 100, n: 1}, {box: [580, 272, 930, 84], a: 70, n: 2}], cap: "g_dns3"},
    {w: 1600, h: 900, svg: () => MOCK.ps5Connections(), marks: [{box: [90, 770, 70, 70], a: -50, n: 1}, {box: [910, 280, 490, 84], a: 110, n: 2}], cap: "g_dns4"},
    {w: 1600, h: 900, svg: c => MOCK.ps5Advanced(c.ip), marks: [{box: [1150, 178, 350, 68], a: 180, n: 1, len: 0.8}, {box: [1150, 274, 350, 68], a: 180, n: 2, len: 0.8}, {box: [1150, 370, 350, 68], a: 180, n: 3, len: 0.8}, {box: [1270, 700, 240, 84], a: 180, n: 4, len: 0.8}], cap: "g_dns5"}
  ],
  golive: [
    {w: 1600, h: 900, svg: () => MOCK.ps5Link(), marks: [{box: [580, 180, 930, 84], a: -40}], cap: "g_live1"},
    {w: 1600, h: 900, svg: () => MOCK.ps5Quality(), marks: [{box: [580, 180, 930, 84], a: -40}], cap: "g_live2"},
    {w: 1600, h: 900, svg: () => MOCK.controller(), marks: [{x: 643, y: 281, a: -60, len: 0.75}], cap: "g_live3"},
    {w: 1600, h: 900, svg: () => MOCK.ps5Create(), marks: [{box: [1000, 640, 300, 200], a: -55}], cap: "g_live4"},
    {w: 1600, h: 900, svg: () => MOCK.ps5Broadcast(), marks: [{box: [895, 235, 610, 84], a: 180, n: 1}, {box: [910, 700, 580, 92], a: 180, n: 2}], cap: "g_live5"}
  ],
  obsws: [
    {w: 760, h: 300, img: "guide/obs-tools-menu.jpg", marks: [{box: [328, 16, 50, 32], a: 60}], cap: "g_ws1"},
    {w: 1600, h: 900, svg: () => MOCK.obsToolsMenu(), marks: [{box: [884, 334, 620, 70], a: 30}], cap: "g_ws2"},
    {w: 1600, h: 900, svg: () => MOCK.obsWsDialog(), marks: [{box: [292, 360, 490, 46], a: 180, n: 1}, {box: [1190, 790, 140, 58], a: -90, n: 2, len: 0.6}, {box: [300, 630, 350, 60], a: 180, n: 3}], cap: "g_ws3"},
    {w: 1600, h: 900, svg: () => MOCK.obsWsInfo(), marks: [{box: [1040, 402, 200, 54], a: 0}], cap: "g_ws4"}
  ],
  obsmanual: [
    {w: 820, h: 506, img: "guide/obs-sources-plus.jpg", marks: [{x: 11, y: 458, a: -35}], cap: "g_m1"},
    {w: 1110, h: 658, img: "guide/obs-add-media.jpg", marks: [{box: [52, 364, 238, 30], a: 10, n: 1}, {box: [313, 90, 742, 58], a: 60, n: 2}], cap: "g_m2"},
    {w: 722, h: 611, img: "guide/obs-props-localfile.jpg", marks: [{x: 156, y: 321, a: 205}], cap: "g_m3"},
    {w: 735, h: 618, img: "guide/obs-props-network.jpg", url: true, marks: [
      {box: [157, 413, 542, 31], a: -150, n: 1, len: 0.6}, {box: [592, 372, 76, 30], a: -60, n: 2, len: 0.6}, {box: [592, 489, 76, 30], a: 150, n: 3, len: 0.6},
      {x: 165, y: 545, a: 180, n: 4, len: 0.55}, {box: [583, 571, 52, 28], a: 180, n: 5, len: 0.6}], cap: "g_m4"}
  ],
  slremote: [
    {w: 1600, h: 900, svg: () => MOCK.slMain(), marks: [{x: 50, y: 834, a: -40}], cap: "g_sl1"},
    {w: 1600, h: 900, svg: () => svg(1600, 900, "#09161D", MOCK.slSettings()), marks: [{box: [190, 729, 350, 48], a: 10}], cap: "g_sl2"},
    {w: 1600, h: 900, svg: () => MOCK.slRemote(), marks: [{box: [610, 270, 290, 290], a: 0, n: 1}, {box: [602, 596, 220, 46], a: 15, n: 2}], cap: "g_sl3"},
    {w: 1600, h: 900, svg: () => MOCK.slToken(), marks: [{box: [1150, 292, 170, 62], a: 15}], cap: "g_sl4"}
  ]
};

function marksSVG(sl) {
  const s = sl.w / 1000, multi = sl.marks.length > 1;
  const dark = "#0A0F18", amber = "var(--arrow, #FFB020)";
  let out = "";
  sl.marks.forEach(m => {
    const a = (m.a ?? -35) * Math.PI / 180, ca = Math.cos(a), sa = Math.sin(a);
    const L = 125 * s * (m.len || 1), gap = 8 * s;
    let cx, cy, reach;
    if (m.box) {
      const [bx0, by0, bw, bh] = m.box, pad = 6 * s;
      cx = bx0 + bw / 2; cy = by0 + bh / 2;
      const hw0 = bw / 2 + pad, hh0 = bh / 2 + pad;
      reach = Math.min(Math.abs(ca) > 1e-6 ? hw0 / Math.abs(ca) : Infinity, Math.abs(sa) > 1e-6 ? hh0 / Math.abs(sa) : Infinity);
      const rx = 10 * s;
      out += `<rect class="g-pulse" x="${bx0 - pad * 2.2}" y="${by0 - pad * 2.2}" width="${bw + pad * 4.4}" height="${bh + pad * 4.4}" rx="${rx * 1.6}" fill="none" stroke="${amber}" stroke-width="${3 * s}"/>`;
      out += `<rect x="${bx0 - pad}" y="${by0 - pad}" width="${bw + pad * 2}" height="${bh + pad * 2}" rx="${rx}" fill="none" stroke="${dark}" stroke-width="${8 * s}" opacity=".5"/>`;
      out += `<rect x="${bx0 - pad}" y="${by0 - pad}" width="${bw + pad * 2}" height="${bh + pad * 2}" rx="${rx}" fill="none" stroke="${amber}" stroke-width="${4.5 * s}"/>`;
    } else {
      const R0 = 20 * s;
      cx = m.x; cy = m.y; reach = R0;
      out += `<circle class="g-pulse" cx="${cx}" cy="${cy}" r="${R0 * 1.7}" fill="none" stroke="${amber}" stroke-width="${3 * s}"/>`;
      out += `<circle cx="${cx}" cy="${cy}" r="${R0}" fill="none" stroke="${dark}" stroke-width="${8 * s}" opacity=".55"/>`;
      out += `<circle cx="${cx}" cy="${cy}" r="${R0}" fill="none" stroke="${amber}" stroke-width="${4.5 * s}"/>`;
    }
    const hx = cx + ca * (reach + gap), hy = cy + sa * (reach + gap);           // arrow tip
    const tx = hx + ca * L, ty = hy + sa * L;                                   // tail
    const hl = 26 * s, hw = 15 * s, bx = hx + ca * hl, by = hy + sa * hl, px = -sa, py = ca;
    const head = `${hx},${hy} ${bx + px * hw},${by + py * hw} ${bx - px * hw},${by - py * hw}`;
    out += `<line x1="${tx}" y1="${ty}" x2="${bx}" y2="${by}" stroke="${dark}" stroke-width="${13 * s}" stroke-linecap="round" opacity=".7"/>`;
    out += `<line x1="${tx}" y1="${ty}" x2="${bx}" y2="${by}" stroke="${amber}" stroke-width="${7.5 * s}" stroke-linecap="round"/>`;
    out += `<polygon points="${head}" fill="${amber}" stroke="${dark}" stroke-width="${3 * s}" stroke-linejoin="round"/>`;
    if (multi && m.n) {
      out += `<circle cx="${tx}" cy="${ty}" r="${19 * s}" fill="${amber}" stroke="${dark}" stroke-width="${3 * s}"/>`;
      out += `<text x="${tx}" y="${ty + 7.5 * s}" text-anchor="middle" font-family="${FONT}" font-size="${22 * s}" font-weight="700" fill="${dark}">${m.n}</text>`;
    }
  });
  return out;
}

const gState = {};
function renderGuide(el, id, ctx) {
  const slides = GUIDES[id];
  if (!el || !slides) return;
  let i = gState[id] || 0;
  if (i >= slides.length) i = 0;
  const sl = slides[i];
  const media = sl.img ? `<img src="${sl.img}" alt="" width="${sl.w}" height="${sl.h}">` : sl.svg(ctx);
  el.dataset.gid = id;
  el.innerHTML = `<div class="g-frame" dir="ltr">${media}<svg class="g-over" viewBox="0 0 ${sl.w} ${sl.h}" preserveAspectRatio="none" aria-hidden="true">${marksSVG(sl)}</svg></div>
    <div class="g-side">
      <div class="g-count">${t("g_step", {i: i + 1, n: slides.length})}</div>
      <div class="g-cap">${t(sl.cap, {ip: gEsc(ctx.ip || "—")})}</div>
      <div class="g-nav">
        <button class="btn small" data-gstep="-1" ${i === 0 ? "disabled" : ""}>${t("g_prev")}</button>
        <div class="g-dots">${slides.map((_, k) => `<button aria-label="${k + 1}" class="${k === i ? "on" : ""}" data-gdot="${k}"></button>`).join("")}</div>
        <button class="btn small primary" data-gstep="1" ${i === slides.length - 1 ? "disabled" : ""}>${t("g_next")}</button>
      </div>
    </div>`;
  return sl;
}
