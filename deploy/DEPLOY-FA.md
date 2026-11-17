# بالا آوردن stream.voxocast.com

محتوای پوشهٔ `site` یه سایت استاتیکه (HTML/CSS/JS، فونت‌ها و فایل‌های دانلود همه داخلشه). هیچ چیزی از گوگل یا CDN بیرونی لود نمی‌شه، پس از ایران هم کامل باز می‌شه.

## ۱. DNS

برای `stream.voxocast.com` یه رکورد بساز، دقیقاً مثل بقیهٔ ساب‌دامین‌هات (mafia، ledger و...).

## ۲. کپی فایل‌ها روی سرور

```bash
sudo mkdir -p /srv/strem
sudo cp -r site/* /srv/strem/
```

اگه Caddy داخل Podman اجرا می‌شه، پوشه رو به کانتینر mount کن:
`-v /srv/strem:/srv/strem:ro,Z`

## ۳. اضافه کردن به Caddy

بلوک داخل `Caddyfile.stream` رو به ته Caddyfile اضافه کن، بعد:

```bash
caddy fmt --overwrite /etc/caddy/Caddyfile
sudo systemctl reload caddy
# یا اگه کانتینره:
podman exec caddy caddy reload --config /etc/caddy/Caddyfile
```

Caddy خودش گواهی HTTPS رو می‌گیره.

**اگه از Cloudflare Tunnel استفاده می‌کنی:** خط اول بلوک رو بکن `http://stream.voxocast.com {`، خط `Strict-Transport-Security` رو پاک کن (TLS دست Cloudflare‌ه) و توی تونل یه Public Hostname برای `stream.voxocast.com` بساز که به `http://localhost:80` اشاره کنه.

## ۴. تست

```bash
curl -I https://stream.voxocast.com
curl -I https://stream.voxocast.com/downloads/StreamBridge-1.3-Windows-x64.zip
```

جواب اولی باید `200` باشه و دومی `Content-Disposition: attachment`.

## موقع انتشار نسخهٔ جدید

1. فایل‌های جدید رو با اسم نسخهٔ جدید بذار توی `/srv/strem/downloads` (اسم جدید یعنی کش Cloudflare مشکلی درست نمی‌کنه).
2. checksumها رو دوباره بساز:
   `cd /srv/strem/downloads && sha256sum StreamBridge-* > SHA256SUMS.txt`
3. توی `index.html` اسم فایل‌ها، حجم‌ها، SHA-256ها و شمارهٔ نسخه رو عوض کن. متن‌های فارسی شمارهٔ نسخه (`hero_meta`، `dl_h`، `foot_ver`) توی `assets/site.js` هستن.
