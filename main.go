// StreamBridge — stream from a PS5 (or any console with Twitch broadcasting)
// to OBS, Kick, YouTube or anywhere, without a capture card.
//
// How it works: the console's DNS is pointed at this computer. When the console
// looks up a Twitch ingest server, StreamBridge answers with this computer's IP,
// so the broadcast arrives at the built-in RTMP server. From there it is available
// to OBS (rtmp://127.0.0.1:1935/live/ps5) and can be relayed directly to platforms.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

var version = "1.3.0"

type App struct {
	cfg      *ConfigStore
	dns      *DNSServer
	stream   *StreamServer
	relays   *RelayManager
	web      *WebServer
	remote   *RemoteServer
	tunnel   *TunnelServer
	rtmpPort int
	panelURL string
	quit     chan struct{}
}

// ActiveIP is the address the console should use as its DNS server.
func (a *App) ActiveIP() net.IP {
	ips := LocalIPv4s()
	pref := a.cfg.Get().PreferredIP
	if pref != "" {
		for _, l := range ips {
			if l.IP.String() == pref {
				return l.IP
			}
		}
	}
	if len(ips) > 0 {
		return ips[0].IP
	}
	return nil
}

// IPForClient picks the local address on the same subnet as the console.
func (a *App) IPForClient(client net.IP) net.IP {
	if client != nil {
		for _, l := range LocalIPv4s() {
			if l.Net.Contains(client) {
				return l.IP
			}
		}
		if client.IsLoopback() {
			return net.IPv4(127, 0, 0, 1)
		}
	}
	return a.ActiveIP()
}

func (a *App) OnSessionStart(s *Session) { a.relays.Sync() }
func (a *App) OnSessionEnd(s *Session)   { a.relays.Sync() }

func (a *App) Shutdown() {
	log.Println("Stopping...")
	a.relays.StopAll()
	a.stream.Stop()
	a.dns.Stop()
	a.remote.stop()
	if a.tunnel.Active() {
		a.tunnel.stopAll()
	}
	a.web.Stop()
}

func alreadyRunning(port int) bool {
	c := http.Client{Timeout: 800 * time.Millisecond}
	resp, err := c.Get(fmt.Sprintf("http://127.0.0.1:%d/api/ping", port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var v map[string]string
	if json.NewDecoder(resp.Body).Decode(&v) != nil {
		return false
	}
	return v["app"] == "streambridge"
}

func explainBindErr(code string, port int) string {
	switch code {
	case "permission":
		return fmt.Sprintf("permission denied for port %d (Linux: run with sudo)", port)
	case "in_use":
		return fmt.Sprintf("port %d is used by another program (close other DNS/streaming servers)", port)
	}
	return code
}

func main() {
	webPort := flag.Int("web-port", 8090, "control panel port (localhost only)")
	remotePort := flag.Int("remote-port", 8190, "phone control port (LAN, only when enabled)")
	dnsPort := flag.Int("dns-port", 53, "DNS port (consoles always use 53)")
	rtmpPort := flag.Int("rtmp-port", 1935, "RTMP port (consoles always use 1935)")
	noBrowser := flag.Bool("no-browser", false, "do not open the control panel automatically")
	background := flag.Bool("background", false, "start quietly (used when starting with the computer)")
	cfgPath := flag.String("config", "", "path to config file")
	flag.Parse()

	cfg := LoadConfig(*cfgPath)
	setupLogging(cfg.path)
	log.SetFlags(log.Ltime)
	printBanner()

	if alreadyRunning(*webPort) {
		if !*background {
			fmt.Println("  StreamBridge is already running. Opening the control panel...")
			openBrowser(fmt.Sprintf("http://127.0.0.1:%d", *webPort))
			time.Sleep(1500 * time.Millisecond)
		}
		return
	}

	app := &App{cfg: cfg, rtmpPort: *rtmpPort, quit: make(chan struct{}, 1)}
	app.dns = NewDNSServer(app, *dnsPort)
	app.stream = NewStreamServer(app, *rtmpPort)
	app.relays = NewRelayManager(app)
	app.web = &WebServer{app: app}
	app.remote = NewRemoteServer(app, *remotePort)
	app.tunnel = NewTunnelServer(app)

	app.dns.Start()
	app.stream.Start()
	app.tunnel.Sync()

	var ln net.Listener
	var err error
	for p := *webPort; p < *webPort+10; p++ {
		ln, err = net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", p))
		if err == nil {
			app.web.port = p
			break
		}
	}
	if ln == nil {
		showFatal("Could not open the StreamBridge control panel: " + errString(err))
		return
	}
	app.web.Start(ln)
	app.remote.Sync()
	refreshAutostartPath()
	app.panelURL = fmt.Sprintf("http://127.0.0.1:%d", app.web.port)

	ip := app.ActiveIP()
	ipStr := "(no network found)"
	if ip != nil {
		ipStr = ip.String()
	}
	fmt.Println("  Control panel :  " + app.panelURL)
	fmt.Println("  PS5 DNS       :  " + ipStr + "   (Primary DNS on the PS5)")
	fmt.Printf("  OBS source    :  rtmp://127.0.0.1:%d/live/ps5\n", *rtmpPort)
	fmt.Println()
	log.Printf("Control panel %s, PS5 DNS %s", app.panelURL, ipStr)
	if st := app.dns.Status(); !st.Running {
		fmt.Println("  !! DNS server could not start: " + explainBindErr(st.Error, *dnsPort))
	}
	if _, _, ok, e := app.stream.Status(); !ok {
		fmt.Println("  !! RTMP server could not start: " + explainBindErr(e, *rtmpPort))
	}
	fmt.Println("  Keep this window open while streaming. Close it to stop.")
	fmt.Println()
	fmt.Println("  ---- log ----")

	if !*noBrowser && !*background {
		go func() { time.Sleep(400 * time.Millisecond); openBrowser(app.panelURL) }()
	}

	runUI(app) // blocks until quit (tray on Windows, console elsewhere)
	app.Shutdown()
}

func printBanner() {
	fmt.Println()
	fmt.Println("  =====================================================")
	fmt.Printf("   StreamBridge %s  -  PS5 to OBS / Kick / YouTube\n", version)
	fmt.Println("  =====================================================")
	fmt.Println()
}
