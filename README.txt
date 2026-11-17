StreamBridge 1.3
Stream your PS5 to OBS, Kick, YouTube or anywhere - no capture card, no Remote Play.

====================================================================
 ENGLISH
====================================================================

WHAT IT DOES
The PS5 thinks it is streaming to Twitch. StreamBridge answers the PS5's
"where is the Twitch server?" question with this computer's address, so the
broadcast (encoded by the PS5 itself, up to 1080p60) arrives here instead.
From here you can:
  A) pull it into OBS (overlays, webcam, alerts), or
  B) send it straight to Kick / YouTube / Twitch / any RTMP server.

START IT
  Windows
    1. Unzip, then double-click StreamBridge.exe.
    2. If you see "Windows protected your PC": More info -> Run anyway.
    3. When Windows Firewall asks: allow it (tick Private networks).
    4. Your browser opens the control panel (http://127.0.0.1:8090).
       Follow the 3 steps shown there.
    StreamBridge runs in the tray next to the clock (no window). Click the
    icon for: Open control panel / Start with Windows / Quit. The icon gets
    a red dot while the PS5 is live.
  macOS
    Use "StreamBridge-macOS-AppleSilicon" (M1/M2/M3/M4) or "...-Intel".
    First time: right-click the file -> Open -> Open.
    If the panel says "no permission for port 53", run it in Terminal:
      sudo ./StreamBridge-macOS-AppleSilicon
  Linux / Raspberry Pi
    chmod +x streambridge-linux-*
    sudo ./streambridge-linux-amd64        (Pi: -arm64 or -armv7)
    (Port 53 needs root. Or once: sudo setcap cap_net_bind_service=+ep <file>)

PICTURE GUIDES
  The control panel shows every step with pictures and arrows on exactly
  what to press: Setup (PS5 DNS, going live) and OBS & Streamlabs
  (turning on the OBS connection, adding the source by hand, Streamlabs token).

ON THE PS5 (one time)
  Settings -> Network -> Settings -> Set Up Internet Connection
  -> highlight your network -> Options button -> Advanced Settings
  -> DNS Settings: Manual
     Primary DNS   = the address shown in the control panel
     Secondary DNS = 1.1.1.1
  Then: Create button -> Broadcast -> Twitch -> Go Live.
  (Twitch must be linked once: Settings -> Users and Accounts -> Linked Services.)

OBS / STREAMLABS - ONE CLICK
  In the control panel press "Add to OBS" or "Add to Streamlabs".
  The PS5 is added to your current scene, fitted to the screen.
  First time only:
    OBS:        Tools -> WebSocket Server Settings -> tick "Enable WebSocket
                server" -> Show Connect Info -> copy the Server Password.
    Streamlabs: Settings -> Remote Control -> click the QR code ->
                Show details -> copy the API Token.
  By hand: Media Source, untick "Local File",
           Input: rtmp://127.0.0.1:1935/live/ps5

IRAN MODE (Twitch blocked where you live)
  The PS5 must reach Twitch to sign in and to start a broadcast, and it
  can't run a VPN. Run V2ray (v2rayN, NekoRay, Hiddify, Clash...) on this
  computer and connect, then: Settings -> "Iran mode" -> on.
  StreamBridge finds V2ray's local proxy by itself (or type its port,
  e.g. 10808) and passes only the PS5's Twitch connections through it,
  still encrypted. The PS5 only needs its DNS pointed here, as usual.
  Port 443 on this computer must be free.

CONTROL FROM YOUR PHONE
  Control panel -> This computer -> "Control from your phone".
  Scan the QR code with your phone (same Wi-Fi), or open the address
  shown and enter the 6-digit PIN.

GOOD TO KNOW
  - Keep this computer on and StreamBridge open while you stream.
  - Nothing is sent to Twitch unless you switch on "Also send to my Twitch".
    Without it, the PS5's own chat/viewer count stay empty.
  - When you stop using StreamBridge, set the PS5's DNS back to Automatic.
  - Stream keys are stored only on this computer:
      Windows: %AppData%\StreamBridge\config.json
      macOS:   ~/Library/Application Support/StreamBridge/config.json
      Linux:   ~/.config/StreamBridge/config.json
  - Some games block broadcasting of certain scenes; that also applies here.
  - Network ports: 53 (DNS), 1935 (RTMP). The control panel listens on
    127.0.0.1 only; phone control (port 8190) is off until you turn it on,
    and then needs the QR code or PIN.


====================================================================
 فارسی
====================================================================

این برنامه چی کار می‌کنه؟
PS5 فکر می‌کنه داره برای توییچ استریم می‌کنه. StreamBridge وقتی PS5 می‌پرسه
«سرور توییچ کجاست؟» آدرس همین کامپیوتر رو جواب می‌ده، پس پخش (که خود PS5 تا
1080p60 انکودش می‌کنه) به‌جای توییچ میاد اینجا. بعد می‌تونی:
  الف) توی OBS ازش استفاده کنی (overlay، وبکم، آلرت)، یا
  ب) مستقیم بفرستیش به کیک / یوتیوب / توییچ / هر سرور RTMP.

اجرا
  ویندوز
    ۱. فایل zip رو باز کن و روی StreamBridge.exe دوبار کلیک کن.
    ۲. اگه پیام «Windows protected your PC» اومد: More info و بعد Run anyway.
    ۳. وقتی فایروال ویندوز پرسید: اجازه بده (تیک Private networks).
    ۴. مرورگر خودش پنل کنترل رو باز می‌کنه (http://127.0.0.1:8090).
       سه مرحله‌ای که اونجا نوشته رو انجام بده.
  مک
    برای M1/M2/M3/M4 فایل StreamBridge-macOS-AppleSilicon و برای مک‌های
    اینتلی فایل ...-Intel رو بردار. بار اول: راست‌کلیک ← Open ← Open.
    اگه پنل گفت «اجازهٔ پورت 53 نیست»، توی ترمینال اجرا کن:
      sudo ./StreamBridge-macOS-AppleSilicon
  لینوکس / رزبری‌پای
    chmod +x streambridge-linux-*
    sudo ./streambridge-linux-amd64        (رزبری: -arm64 یا -armv7)

راهنمای تصویری
  پنل کنترل همهٔ مرحله‌ها رو با تصویر و فلش روی دقیقاً همون جایی که باید
  بزنی نشون می‌ده: «راه‌اندازی» (DNS پلی‌استیشن، شروع پخش) و «OBS و Streamlabs»
  (روشن کردن اتصال OBS، افزودن دستی، توکن Streamlabs).

روی PS5 (فقط یک بار)
  Settings -> Network -> Settings -> Set Up Internet Connection
  -> شبکه‌ات رو انتخاب کن -> دکمهٔ Options -> Advanced Settings
  -> DNS Settings: Manual
     Primary DNS   = آدرسی که توی پنل کنترل نوشته
     Secondary DNS = 1.1.1.1
  بعد: دکمهٔ Create -> Broadcast -> Twitch -> Go Live.
  (توییچ باید یک بار لینک بشه: Settings -> Users and Accounts -> Linked Services.)

OBS / STREAMLABS با یک کلیک
  توی پنل کنترل «افزودن به OBS» یا «افزودن به Streamlabs» رو بزن.
  PS5 به صحنهٔ فعلی اضافه می‌شه و اندازهٔ صفحه می‌شه.
  فقط بار اول:
    OBS:        Tools -> WebSocket Server Settings -> تیک Enable WebSocket
                server -> Show Connect Info -> کپی Server Password.
    Streamlabs: Settings -> Remote Control -> کلیک روی کد QR ->
                Show details -> کپی API Token.

حالت ایران (وقتی توییچ فیلتره)
  PS5 برای لاگین و شروع پخش باید به توییچ برسه و روش VPN نمی‌شه زد.
  V2ray (v2rayN، NekoRay، Hiddify، Clash...) رو روی همین کامپیوتر وصل کن،
  بعد: تنظیمات -> «حالت ایران» -> روشن.
  StreamBridge خودش پروکسی محلی V2ray رو پیدا می‌کنه (یا پورتش رو بنویس،
  مثلاً 10808) و فقط اتصال‌های توییچ PS5 رو، رمزنگاری‌شده، از اون رد می‌کنه.
  روی PS5 فقط مثل همیشه DNS رو روی این کامپیوتر بذار.
  پورت 443 این کامپیوتر باید آزاد باشه.

کنترل از گوشی
  پنل کنترل -> این کامپیوتر -> «کنترل از گوشی».
  کد QR رو با گوشی اسکن کن (همون وای‌فای)، یا آدرسی که نوشته رو باز کن و
  رمز ۶ رقمی رو بزن.

نکته‌ها
  - تا وقتی استریم می‌کنی، کامپیوتر روشن و StreamBridge باز باشه.
  - چیزی به توییچ نمی‌ره، مگه این‌که «به توییچ خودم هم بفرست» رو روشن کنی.
    بدون اون، چت و تعداد بیننده روی خود PS5 خالی می‌مونه.
  - وقتی دیگه از StreamBridge استفاده نمی‌کنی، DNS پلی‌استیشن رو برگردون روی Automatic.
  - استریم کی‌ها فقط روی همین کامپیوتر ذخیره می‌شن (مسیرهاش توی بخش انگلیسی بالا هست).
  - بعضی بازی‌ها پخش بعضی صحنه‌ها رو قفل می‌کنن؛ اینجا هم همین‌طوره.
  - پورت‌ها: 53 (DNS) و 1935 (RTMP). پنل کنترل فقط روی 127.0.0.1 بازه؛
    کنترل از گوشی (پورت 8190) تا روشنش نکنی خاموشه و بعدش هم کد QR یا رمز می‌خواد.


--------------------------------------------------------------------
Built with datarhei/joy4 (RTMP), miekg/dns, gorilla/websocket, skip2/go-qrcode
and fyne-io/systray. Redirect patterns follow
Lightstream's open-source Prism DNS and Aioros/console-streaming-server.
See THIRD_PARTY_LICENSES.txt.
