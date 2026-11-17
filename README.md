# StreamBridge

Stream your PS5 to OBS, Streamlabs, Kick, YouTube or any RTMP server, without a capture card and without Remote Play.

Website and downloads: **https://stream.voxocast.com**

## How it works

The PS5 can only broadcast to Twitch (and YouTube). StreamBridge runs a small DNS server on your computer. When you point the PS5's DNS at it, it answers the PS5's "where is Twitch's ingest server?" lookup with your computer's own address and passes every other lookup on to normal DNS. The PS5 then sends its broadcast, already encoded at up to 1080p60, to the RTMP server built into StreamBridge. From there you can:

- add it to OBS or Streamlabs with one click (a media source pulling `rtmp://127.0.0.1:1935/live/ps5`), or
- relay it straight to Kick, YouTube, Twitch or any RTMP/RTMPS server, several at once, without re-encoding.

Delay from the PS5 to the computer is under 2 seconds.

Other features:

- Web control panel in English and Persian, with picture guides for the one-time setup
- Tray icon and start with Windows
- Phone control over your home Wi-Fi (off by default, PIN or QR code)
- Iran mode: passes only the PS5's Twitch traffic through a local V2ray / SOCKS / HTTP proxy, so the PS5 can sign in to Twitch where it's blocked

## Build from source

Needs Go 1.24 or newer.

```bash
go test ./...
./build.sh           # all platforms -> dist/ (+ SHA256SUMS.txt)
./build.sh --site    # same, and copies the packages into site/downloads
```

Run it locally without root by moving the DNS port:

```bash
go run . -dns-port 5353 -no-browser
```

Flags: `-web-port` (8090), `-remote-port` (8190), `-dns-port` (53), `-rtmp-port` (1935), `-no-browser`, `-background`, `-config <file>`.

`go.mod` points the `golang.org/x/*` modules at their official GitHub mirrors (`github.com/golang/*`), so builds work where `golang.org` isn't reachable.

## Layout

| Path | What |
|---|---|
| `*.go` | The app: DNS (`dnsserver.go`), RTMP server (`stream.go`), relays (`relay.go`), control panel API (`web.go`), OBS/Streamlabs import (`obsimport.go`), phone control (`remote.go`), Iran mode (`tunnel.go`), tray and Windows integration (`tray_windows.go`, `platform_*.go`) |
| `web/` | Control panel UI, embedded in the binary |
| `assets/` | Tray icons |
| `winres/`, `rsrc_windows_amd64.syso` | Windows icon, version info and manifest (regenerate with `goversioninfo -64 -o rsrc_windows_amd64.syso winres/versioninfo.json`) |
| `site/` | The stream.voxocast.com website (static). Release files go in `site/downloads/` and aren't tracked in git |
| `deploy/` | Caddy config and deploy notes for the website |

## Releases

Release packages (Windows, macOS, Linux and source) are attached to [GitHub Releases](../../releases) and served from the website. Their SHA-256 checksums are in `site/downloads/SHA256SUMS.txt`.

---

## فارسی

StreamBridge پخش توییچِ PS5 رو با یه DNS محلی به کامپیوتر خودت می‌کشونه. از اونجا با یه کلیک به OBS یا Streamlabs اضافه‌ش می‌کنی، یا بدون انکود دوباره مستقیم به Kick، YouTube یا هر سرور RTMP دیگه می‌فرستیش. نه کپچر کارت لازمه نه Remote Play، و تأخیرش کمتر از ۲ ثانیه‌ست.

دانلود و راهنما: https://stream.voxocast.com

ساختن از سورس: `./build.sh` (با Go نسخهٔ ۱.۲۴ به بعد)

PlayStation and PS5 are trademarks of Sony Interactive Entertainment. StreamBridge isn't affiliated with Sony, Twitch, Kick, YouTube, OBS or Streamlabs. Third-party licenses: `THIRD_PARTY_LICENSES.txt`.
