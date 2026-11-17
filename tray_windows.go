//go:build windows

package main

import (
	_ "embed"
	"fmt"
	"time"

	"fyne.io/systray"
)

//go:embed assets/tray_idle.ico
var trayIdleIcon []byte

//go:embed assets/tray_live.ico
var trayLiveIcon []byte

var trayText = map[string]map[string]string{
	"en": {"open": "Open control panel", "auto": "Start with Windows", "quit": "Quit StreamBridge",
		"idle": "StreamBridge — waiting for the PS5", "live": "StreamBridge — LIVE %s", "ready": "StreamBridge — PS5 connected"},
	"fa": {"open": "باز کردن پنل کنترل", "auto": "اجرا با روشن شدن ویندوز", "quit": "خروج از StreamBridge",
		"idle": "StreamBridge — منتظر PS5", "live": "StreamBridge — زنده %s", "ready": "StreamBridge — PS5 وصله"},
}

func trT(lang, k string) string {
	if m, ok := trayText[lang]; ok {
		return m[k]
	}
	return trayText["en"][k]
}

// runUI shows the tray icon and blocks until the user quits.
func runUI(app *App) {
	systray.Run(func() { trayReady(app) }, func() {})
}

func trayReady(app *App) {
	lang := app.cfg.Get().Lang
	systray.SetIcon(trayIdleIcon)
	systray.SetTooltip(trT(lang, "idle"))
	mOpen := systray.AddMenuItem(trT(lang, "open"), "")
	systray.AddSeparator()
	mAuto := systray.AddMenuItemCheckbox(trT(lang, "auto"), "", autostartEnabled())
	systray.AddSeparator()
	mQuit := systray.AddMenuItem(trT(lang, "quit"), "")

	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		wasLive := false
		lastTip := ""
		for {
			select {
			case <-mOpen.ClickedCh:
				openBrowser(app.panelURL)
			case <-mAuto.ClickedCh:
				_ = setAutostart(!autostartEnabled())
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			case <-app.quit:
				systray.Quit()
				return
			case <-t.C:
			}
			// Refresh language, checkbox, icon and tooltip.
			if l := app.cfg.Get().Lang; l != lang {
				lang = l
				mOpen.SetTitle(trT(lang, "open"))
				mAuto.SetTitle(trT(lang, "auto"))
				mQuit.SetTitle(trT(lang, "quit"))
			}
			if autostartEnabled() {
				mAuto.Check()
			} else {
				mAuto.Uncheck()
			}
			st, _, _, _ := app.stream.Status()
			tip := trT(lang, "idle")
			if st.Live {
				q := ""
				if st.Height > 0 {
					q = fmt.Sprintf("%dp%.0f", st.Height, st.FPS)
				}
				tip = fmt.Sprintf(trT(lang, "live"), q)
			} else {
				for _, d := range app.dns.Devices() {
					if d.Kind == "playstation" && time.Since(d.LastSeen) < 3*time.Minute {
						tip = trT(lang, "ready")
						break
					}
				}
			}
			if st.Live != wasLive {
				wasLive = st.Live
				if st.Live {
					systray.SetIcon(trayLiveIcon)
				} else {
					systray.SetIcon(trayIdleIcon)
				}
			}
			if tip != lastTip {
				lastTip = tip
				systray.SetTooltip(tip)
			}
		}
	}()
}
