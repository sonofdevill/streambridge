/* StreamBridge website: language switch, download picker, hero signal chain */
(function () {
  "use strict";
  var root = document.documentElement;

  var EN = {
    title: "StreamBridge: stream your PS5 to OBS, Kick and YouTube without a capture card",
    desc: "Free app that catches your PS5's Twitch broadcast on your computer and sends it to OBS, Streamlabs, Kick, YouTube or any RTMP server. No capture card, under 2 seconds of delay.",
    p_title: "Privacy: StreamBridge",
    g_title: "Support StreamBridge",
    dl_win: "Download for Windows", dl_mac: "Download for macOS", dl_linux: "Download for Linux", dl_see: "See downloads"
  };

  var FA = {
    title: "StreamBridge: استریم PS5 روی OBS، کیک و یوتیوب بدون کپچر کارت",
    desc: "برنامهٔ رایگانی که پخش توییچِ PS5 رو روی کامپیوترت می‌گیره و به OBS، Streamlabs، کیک، یوتیوب یا هر سرور RTMP می‌فرسته. بدون کپچر کارت، با تأخیر کمتر از ۲ ثانیه.",
    p_title: "حریم خصوصی: StreamBridge",
    g_title: "حمایت از StreamBridge",
    skip: "رفتن به محتوا",
    nav_how: "چطور کار می‌کنه", nav_dl: "دانلود", nav_win: "هشدار ویندوز", nav_faq: "سؤال‌ها",
    h1: "PS5 رو بدون کپچر کارت روی OBS، کیک و یوتیوب استریم کن.",
    lede: "PS5 فکر می‌کنه داره روی توییچ لایو می‌ره. StreamBridge همون پخش رو با همون کیفیتی که PS5 می‌فرسته روی کامپیوترت می‌گیره و می‌فرسته هر جا که استریم می‌کنی. تأخیرش کمتر از دو ثانیه‌ست.",
    dl_generic: "دانلود StreamBridge", dl_win: "دانلود برای ویندوز", dl_mac: "دانلود برای مک", dl_linux: "دانلود برای لینوکس", dl_see: "دیدن دانلودها",
    see_how: "ببین چطور کار می‌کنه",
    hero_meta: "رایگان. نسخهٔ ۱.۳ برای ویندوز، مک و لینوکس.",
    chain_aria: "PS5 پخشش رو به کامپیوترت می‌فرسته و کامپیوتر اون رو به OBS، کیک و یوتیوب می‌ده",
    c_src: "در حال لایو روی «توییچ»", c_dns: "از راه DNS", c_hub: "کامپیوتر تو", c_hub_s: "StreamBridge پخش رو می‌گیره",
    c_wait: "آماده", c_air: "روی آنتن",
    how_h: "سه مرحله، حدود پنج دقیقه",
    how_p: "این کار رو فقط یک بار انجام می‌دی. از اون به بعد StreamBridge رو باز کن و مثل همیشه روی PS5 دکمهٔ <span class=\"ltr\">Go Live</span> رو بزن.",
    s1_n: "روی کامپیوتر", s1_h: "StreamBridge رو باز کن", s1_p: "یه آدرس نشونت می‌ده. فقط همین رو باید کپی کنی.", s1_k: "DNS پلی‌استیشن (Primary)",
    s2_n: "روی PS5", s2_h: "اون آدرس رو توی تنظیمات DNS بذار", s2_p: "توی <span class=\"ltr\">Settings › Network</span> اتصالت رو انتخاب کن، بعد <span class=\"ltr\">Advanced Settings</span> و DNS رو بذار روی Manual.",
    s3_n: "روی PS5", s3_h: "مثل همیشه روی توییچ لایو برو", s3_p: "پخش می‌رسه به StreamBridge. با یک کلیک به OBS اضافه‌اش کن، یا مستقیم بفرستش به کیک یا یوتیوب.",
    shot_alt: "پنل کنترل StreamBridge وقتی PS5 لایوه: 1920×1080 با ۶۰ فریم، در حال ارسال به کیک و یوتیوب",
    shot_cap: "پنل کنترل وقتی PS5 لایوه و هم‌زمان به OBS، کیک و یوتیوب می‌فرسته.",
    spec_h: "چی بهت می‌ده",
    sp_pic: "تصویر", sp_pic_v: "دقیقاً همونی که PS5 می‌فرسته، تا 1080p با ۶۰ فریم", sp_pic_s: "دوباره انکود نمی‌شه، پس نه کیفیت کم می‌شه نه به کامپیوترت فشار میاد",
    sp_delay: "تأخیر", sp_delay_v: "کمتر از ۲ ثانیه تا کامپیوترت", sp_delay_s: "با یه PS5 واقعی روی شبکهٔ خونگی اندازه گرفته شده",
    sp_out: "ارسال به", sp_out_v: "OBS، Streamlabs، کیک، یوتیوب، توییچ", sp_out_s: "یا هر سرور RTMP یا RTMPS، چندتا با هم",
    sp_need: "لازم داری", sp_need_v: "یه کامپیوتر توی همون شبکه‌ای که PS5 هست", sp_need_s: "با کابل بهتره. نه کپچر کارت لازمه، نه <span class=\"ltr\">Remote Play</span>.",
    sp_os: "سیستم‌عامل", sp_os_v: "ویندوز ۱۰ و ۱۱، مک، لینوکس", sp_os_s: "رزبری‌پای هم پشتیبانی می‌شه",
    sp_cost: "قیمت", sp_cost_v: "رایگان", sp_cost_s: "نه اکانت، نه تبلیغ، نه اشتراک",
    f1_h: "با یک کلیک توی OBS یا Streamlabs",
    f1_p: "«افزودن به OBS» رو بزن تا PS5 توی صحنهٔ فعلیت بیاد، اندازهٔ صفحه بشه و زیر overlayهات قرار بگیره. وبکم، آلرت‌ها و چتت همون‌طور که هستن می‌مونن.",
    f1_l1: "با OBS نسخهٔ ۲۸ به بعد و <span class=\"ltr\">Streamlabs Desktop</span> کار می‌کنه", f1_l2: "راهنمای تصویری با فلش برای تنظیمات بار اول",
    f1_alt: "صفحهٔ OBS و Streamlabs با دکمه‌های افزودن و راهنمای تصویری",
    f2_h: "از گوشی هم کنترلش کن",
    f2_p: "یه کد QR رو اسکن کن تا پنل کنترل روی گوشیت باز بشه. ببین لایوی یا نه، کیک یا یوتیوب رو روشن و خاموش کن، همه از روی مبل.",
    f2_l1: "تا خودت روشنش نکنی خاموشه", f2_l2: "با رمز ۶ رقمی محافظت می‌شه و فقط روی وای‌فای خونه کار می‌کنه",
    f2_alt: "پنل کنترل StreamBridge روی گوشی",
    f3_h: "حالت ایران، برای وقتی توییچ فیلتره",
    f3_p: "PS5 برای لاگین و شروع پخش باید به توییچ برسه و روش نمی‌شه VPN زد. V2ray رو روی کامپیوترت وصل کن و حالت ایران رو روشن کن. StreamBridge فقط ترافیک توییچِ PS5 رو، همچنان رمزنگاری‌شده، از اون رد می‌کنه.",
    f3_l1: "روی PS5 جز DNS چیزی نصب یا عوض نمی‌کنی", f3_l2: "v2rayN، NekoRay، Hiddify یا Clash رو خودش پیدا می‌کنه",
    r1: "فقط DNS عوض شده", r2: "حالت ایران روشن", r3: "روی همون کامپیوتر", r4: "لاگین و <span class=\"ltr\">Go Live</span> کار می‌کنه",
    dl_h: "دانلود StreamBridge ۱.۳",
    dl_p: "از حالت فشرده درش بیار و بازش کن. نصب لازم نداره.",
    dl_mobile: "StreamBridge روی کامپیوتر اجرا می‌شه، نه روی گوشی. این صفحه رو روی کامپیوتری باز کن که با PS5 توی یه شبکه‌ست.",
    mine: "برای همین کامپیوتر",
    req_win: "ویندوز ۱۰ یا ۱۱، ۶۴ بیتی",
    req_mac: "Apple Silicon (M1 به بعد) و اینتل، هر دو توی همین zip",
    req_linux: "x86-64، ARM64 و ARMv7 (رزبری‌پای)، همه توی یه فایل",
    dl_zip: "دانلود فایل zip", dl_tar: "دانلود فایل tar.gz",
    first_win: "احتمالاً ویندوز بار اول یه پنجرهٔ آبی <span class=\"ltr\">Windows protected your PC</span> نشون می‌ده. <a href=\"#windows\">ببین کجا رو باید بزنی.</a>",
    first_mac: "بار اول: روی فایل راست‌کلیک کن و Open رو بزن. توی macOS 15 به بعد برو <span class=\"ltr\">System Settings</span>، بخش <span class=\"ltr\">Privacy &amp; Security</span> و <span class=\"ltr\">Open Anyway</span> رو بزن.",
    first_linux: "با <code class=\"ltr\">sudo</code> اجراش کن، چون پورت DNS (53) بهش نیاز داره.",
    dl_more: "اینا هم هست: <a href=\"/downloads/StreamBridge-1.3-source.zip\">سورس کد</a> (۰٫۳ مگابایت) و <a href=\"/downloads/SHA256SUMS.txt\">همهٔ checksumها</a>.",
    win_h: "پیام <span class=\"ltr\">Windows protected your PC</span> اومد؟",
    win_p: "ویندوز این پنجره رو برای برنامه‌های تازه‌ای نشون می‌ده که هنوز زیاد ندیده. هشدار ویروس نیست. با دو کلیک رد می‌شی و برای این فایل دیگه نمی‌بینیش.",
    w1_b: "۱. روی <span class=\"ltr\">More info</span> کلیک کن", w1_s: "همون لینک کوچیک زیرخط‌دار زیر متنه.",
    w2_b: "۲. روی <span class=\"ltr\">Run anyway</span> کلیک کن", w2_s: "StreamBridge اجرا می‌شه و پنل کنترلش رو توی مرورگرت باز می‌کنه.",
    why1_h: "بعد اجازهٔ فایروال رو بده",
    why1_p: "فایروال ویندوز یه بار می‌پرسه StreamBridge از شبکه استفاده کنه یا نه. با تیک <span class=\"ltr\">Private networks</span> روی Allow بزن. بدون این، PS5 به کامپیوترت نمی‌رسه.",
    why2_h: "می‌خوای مطمئن شی فایل مال ماست؟",
    why2_p: "PowerShell رو توی پوشهٔ Downloads باز کن و این رو بزن: <code>Get-FileHash StreamBridge-1.3-Windows-x64.zip</code>. نتیجه باید با SHA-256 زیر دانلود ویندوز یکی باشه.",
    faq_h: "سؤال‌ها",
    q1: "واقعاً کپچر کارت لازم نیست؟",
    a1: "واقعاً. PS5 خودش پخشش رو برای توییچ انکود می‌کنه. StreamBridge فقط به‌جای اینکه بذاره بره توییچ، روی کامپیوترت می‌گیرتش؛ پس تصویر خود PS5 رو بدون کابل و کپچر کارت داری.",
    q2: "اکانت توییچ لازمه؟",
    a2: "آره. PS5 فقط وقتی اکانت توییچ لینک شده باشه <span class=\"ltr\">Go Live</span> رو نشون می‌ده (Settings، <span class=\"ltr\">Users and Accounts</span>، <span class=\"ltr\">Linked Services</span>). تا «به توییچ خودم هم بفرست» رو توی StreamBridge روشن نکنی، چیزی به توییچ فرستاده نمی‌شه.",
    q3: "چقدر تأخیر داره؟",
    a3: "از PS5 تا کامپیوترت کمتر از دو ثانیه. کیک، یوتیوب یا توییچ هم مثل هر استریم دیگه‌ای تأخیر خودشون رو اضافه می‌کنن.",
    q4: "اینترنتم یا بقیهٔ دستگاه‌ها کند نمی‌شن؟",
    a4: "نه. StreamBridge فقط به سؤال PS5 دربارهٔ آدرس سرور ویدیوی توییچ جواب می‌ده. بقیهٔ درخواست‌ها به DNS معمولی (1.1.1.1) می‌رن و فقط PS5 از StreamBridge به‌عنوان DNS استفاده می‌کنه.",
    q5: "کامپیوتر باید روشن بمونه؟",
    a5: "آره، تا وقتی استریم می‌کنی. هر وقت دیگه نخواستی از StreamBridge استفاده کنی، DNS پلی‌استیشن رو برگردون روی Automatic.",
    q6: "استریم کی‌ام کجا ذخیره می‌شه؟",
    a6: "فقط روی کامپیوتر خودت، توی فایل تنظیمات StreamBridge. جز پلتفرمی که مال اونه جای دیگه‌ای فرستاده نمی‌شه.",
    q7: "توی ایران کار می‌کنه؟",
    a7: "آره. حالت ایران رو توی تنظیمات روشن کن و V2ray رو روی همون کامپیوتر وصل نگه دار. روی PS5 فقط مثل همیشه DNS رو روی کامپیوترت بذار.",
    q8: "PS4 یا ایکس‌باکس چی؟",
    a8: "StreamBridge برای PS5 ساخته و تست شده. کنسول‌های دیگه‌ای که روی توییچ استریم می‌کنن شاید کار کنن، ولی ما تستشون نکردیم.",
    q9: "چرا ویندوز برای این برنامه هشدار می‌ده؟",
    a9: "ویندوز برای هر برنامهٔ تازه‌ای که هنوز زیاد دانلود نشده هشدار می‌ده. <a href=\"#windows\">دو کلیکی که ازش ردت می‌کنه رو ببین.</a>",
    foot_by: "StreamBridge ساختهٔ", foot_ver: "نسخهٔ ۱٫۳، نوامبر ۲۰۲۶.",
    don_h: "حمایت از StreamBridge",
    don_p: "StreamBridge رایگانه و رایگان می‌مونه. اگه باعث شد کپچر کارت نخری و دوست داری کمک کنی، می‌تونی حمایت کنی. کاملاً اختیاریه.",
    don_warn: "فقط روی همون شبکه‌ای که نوشته شده بفرست؛ ارزی که روی شبکهٔ اشتباه فرستاده بشه برنمی‌گرده.",
    don_copy: "کپی", don_copied: "کپی شد", don_qr_show: "نمایش QR", don_qr_hide: "بستن QR", nav_support: "حمایت", nav_donate: "حمایت مالی",
    g_eyebrow: "اختیاری", g_evm: "یک آدرس برای همهٔ این شبکه‌ها:",
    g_s1: "شبکه‌ای رو انتخاب کن که کیف پولت باهاش می‌فرسته.",
    g_s2: "آدرس رو کپی کن، یا QR کدش رو اسکن کن.",
    g_s3: "قبل از فرستادن، چند کاراکتر اول و آخر آدرس رو با همین صفحه مقایسه کن.",
    g_thanks: "ممنون که از StreamBridge استفاده می‌کنی.",
    privacy: "حریم خصوصی", source: "سورس کد", back: "برگشت به صفحهٔ اصلی",
    legal: "PlayStation و PS5 علامت تجاری Sony Interactive Entertainment هستن. StreamBridge وابسته به سونی، توییچ، کیک، یوتیوب، OBS یا Streamlabs نیست.",
    /* privacy page */
    p_h: "حریم خصوصی",
    p_date: "آخرین به‌روزرسانی: ۱۶ نوامبر ۲۰۲۶",
    p_intro: "StreamBridge روی کامپیوتر خودت اجرا می‌شه. هیچ اطلاعات شخصی‌ای جمع نمی‌کنه، به ما نمی‌فرسته و نمی‌فروشه. نه اکانت داره، نه آمارگیری، نه تبلیغ.",
    p_h1: "چی روی کامپیوترت ذخیره می‌شه",
    p_t1: "تنظیماتت، از جمله استریم کی‌هایی که وارد می‌کنی، فقط توی یه فایل روی کامپیوتر خودت ذخیره می‌شن:",
    p_h2: "برنامه به کجا وصل می‌شه",
    p_l1: "پلتفرم‌هایی که خودت انتخاب می‌کنی (مثل کیک، یوتیوب یا توییچ) تا استریمت رو بفرسته.",
    p_l2: "سرورهای DNS عمومی (1.1.1.1 و 8.8.8.8) تا به بقیهٔ درخواست‌های PS5 جواب بده.",
    p_l3: "در حالت ایران، پروکسی خودت (مثلاً V2ray) روی همون کامپیوتر.",
    p_l4: "OBS یا Streamlabs روی همون کامپیوتر، فقط وقتی دکمهٔ افزودن رو بزنی.",
    p_t2: "برنامه آپدیت خودکار یا گزارش خطا به جایی نمی‌فرسته.",
    p_h3: "کنترل از گوشی",
    p_t3: "کنترل از گوشی تا خودت روشنش نکنی خاموشه. وقتی روشنه فقط توی شبکهٔ خونه‌ات در دسترسه و رمز ۶ رقمی یا کد QR لازم داره.",
    p_h4: "این وب‌سایت",
    p_t4: "این سایت کوکی، ابزار آمارگیری یا ردیاب نداره. فقط زبانی که انتخاب می‌کنی توی مرورگر خودت ذخیره می‌شه. سرور مثل هر وب‌سایتی ممکنه برای امنیت، آدرس IP و زمان درخواست‌ها رو مدت کوتاهی نگه داره.",
    p_h5: "تغییرات",
    p_t5: "اگه این متن عوض بشه، نسخهٔ جدید همین‌جا با تاریخ جدید منتشر می‌شه."
  };

  function q(sel, ctx) { return Array.prototype.slice.call((ctx || document).querySelectorAll(sel)); }
  function isPrivacy() { return !!document.querySelector(".doc"); }

  /* ---------- download picker ---------- */
  var FILES = {
    windows: { href: "/downloads/StreamBridge-1.3-Windows-x64.zip", key: "dl_win", after: "#windows" },
    mac: { href: "/downloads/StreamBridge-1.3-macOS.zip", key: "dl_mac", after: "#download" },
    linux: { href: "/downloads/StreamBridge-1.3-Linux.tar.gz", key: "dl_linux", after: "#download" }
  };
  function detectOS() {
    if (root.classList.contains("is-mobile")) return "mobile";
    var ua = navigator.userAgent || "";
    var p = (navigator.userAgentData && navigator.userAgentData.platform) || navigator.platform || "";
    if (/win/i.test(p) || /Windows/.test(ua)) return "windows";
    if (/mac/i.test(p) || /Mac OS X/.test(ua)) return "mac";
    if (/linux|x11/i.test(p) || /Linux|X11/.test(ua)) return "linux";
    return "unknown";
  }
  function setupDownload() {
    var btn = document.getElementById("hero-dl"), label = document.getElementById("hero-dl-label");
    if (!btn || !label) return;
    var os = detectOS(), f = FILES[os];
    if (f) {
      btn.href = f.href;
      btn.setAttribute("download", "");
      label.setAttribute("data-i18n", f.key);
      label.textContent = EN[f.key];
      var card = document.querySelector('.os[data-os="' + os + '"]');
      if (card) card.classList.add("mine");
      btn.addEventListener("click", function () {
        setTimeout(function () {
          var t = document.querySelector(f.after);
          if (t) t.scrollIntoView({ behavior: matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth" });
        }, 400);
      });
    } else if (os === "mobile") {
      label.setAttribute("data-i18n", "dl_see");
      label.textContent = EN.dl_see;
    }
  }

  /* ---------- language ---------- */
  function apply(lang) {
    var fa = lang === "fa";
    root.lang = fa ? "fa" : "en";
    root.dir = fa ? "rtl" : "ltr";
    q("[data-i18n]").forEach(function (el) {
      if (el.dataset.en === undefined) el.dataset.en = el.innerHTML;
      var v = fa ? FA[el.dataset.i18n] : null;
      el.innerHTML = v != null ? v : el.dataset.en;
    });
    q("[data-i18n-alt]").forEach(function (el) {
      if (el.dataset.enAlt === undefined) el.dataset.enAlt = el.getAttribute("alt") || "";
      var v = fa ? FA[el.dataset.i18nAlt] : null;
      el.setAttribute("alt", v != null ? v : el.dataset.enAlt);
    });
    q("[data-i18n-aria]").forEach(function (el) {
      if (el.dataset.enAria === undefined) el.dataset.enAria = el.getAttribute("aria-label") || "";
      var v = fa ? FA[el.dataset.i18nAria] : null;
      el.setAttribute("aria-label", v != null ? v : el.dataset.enAria);
    });
    q("img[data-src-fa]").forEach(function (img) {
      if (img.dataset.srcEn === undefined) img.dataset.srcEn = img.getAttribute("src");
      var want = fa ? img.dataset.srcFa : img.dataset.srcEn;
      if (img.getAttribute("src") !== want) img.setAttribute("src", want);
    });
    var isGive = !!document.querySelector(".give");
    var tkey = isPrivacy() ? "p_title" : isGive ? "g_title" : "title";
    document.title = (fa ? FA : EN)[tkey];
    var md = document.querySelector('meta[name="description"]');
    if (md && !isPrivacy() && !isGive) md.setAttribute("content", fa ? FA.desc : EN.desc);
    q(".seg button").forEach(function (b) { b.setAttribute("aria-pressed", String(b.dataset.lang === lang)); });
    root.classList.remove("pending");
  }
  function setLang(lang) {
    try { localStorage.setItem("sb-lang", lang); } catch (e) {}
    apply(lang);
  }

  /* ---------- hero signal chain: one run when it comes into view ---------- */
  function setupChain() {
    var chain = document.getElementById("chain");
    if (!chain) return;
    if (!root.classList.contains("anim") || !("IntersectionObserver" in window)) { chain.classList.add("played"); return; }
    var io = new IntersectionObserver(function (entries) {
      if (!entries[0].isIntersecting) return;
      io.disconnect();
      chain.classList.add("run");
      setTimeout(function () { chain.classList.add("played"); chain.classList.remove("run"); }, 2700);
    }, { threshold: 0.4 });
    io.observe(chain);
  }

  /* ---------- optional donations: copy address, show/hide QR ---------- */
  function setupDonate() {
    var live = document.querySelector(".don-live");
    q(".don-row").forEach(function (row) {
      var code = row.querySelector(".don-addr");
      var copy = row.querySelector(".don-copy");
      var qr = row.querySelector(".don-qr");
      var box = row.querySelector(".don-qrbox");
      var timer;
      copy.addEventListener("click", function () {
        var addr = code.textContent.trim();
        var done = function () {
          copy.classList.add("done");
          if (live) live.textContent = root.lang === "fa" ? FA.don_copied : "Copied";
          clearTimeout(timer);
          timer = setTimeout(function () { copy.classList.remove("done"); if (live) live.textContent = ""; }, 1800);
        };
        var select = function () {
          var r = document.createRange(); r.selectNodeContents(code);
          var sel = window.getSelection(); sel.removeAllRanges(); sel.addRange(r);
        };
        if (navigator.clipboard && navigator.clipboard.writeText) navigator.clipboard.writeText(addr).then(done, select);
        else select();
      });
      qr.addEventListener("click", function () {
        var open = box.hidden;
        box.hidden = !open;
        qr.setAttribute("aria-expanded", String(open));
      });
    });
  }

  function init() {
    setupDownload();
    setupDonate();
    apply(root.lang === "fa" ? "fa" : "en");
    q(".seg button").forEach(function (b) { b.addEventListener("click", function () { setLang(b.dataset.lang); }); });
    setupChain();
    var top = document.getElementById("top");
    if (top) {
      var onScroll = function () { top.classList.toggle("scrolled", window.scrollY > 8); };
      window.addEventListener("scroll", onScroll, { passive: true });
      onScroll();
    }
  }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init); else init();
})();
