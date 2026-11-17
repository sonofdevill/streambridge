package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/datarhei/joy4/av/avutil"
	"github.com/datarhei/joy4/format/flv"
)

//go:embed web
var webFS embed.FS

type IPView struct {
	IP    string `json:"ip"`
	Iface string `json:"iface"`
}

type DestView struct {
	ID        string `json:"id"`
	Platform  string `json:"platform"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	KeyMasked string `json:"keyMasked"`
	Enabled   bool   `json:"enabled"`
	State     string `json:"state"`
	Error     string `json:"error,omitempty"`
}

type DeviceView struct {
	IP          string  `json:"ip"`
	Kind        string  `json:"kind"`
	LastSeenAgo float64 `json:"lastSeenAgo"`
	Queries     int     `json:"queries"`
	AskedTwitch bool    `json:"askedTwitch"`
	TwitchAgo   float64 `json:"twitchAgo"`
}

type SettingsView struct {
	Lang            string   `json:"lang"`
	PreferredIP     string   `json:"preferredIP"`
	ForwardToTwitch bool     `json:"forwardToTwitch"`
	Upstream        []string `json:"upstream"`
}

type ToggleView struct {
	Supported bool `json:"supported"`
	Enabled   bool `json:"enabled"`
}

type ImportView struct {
	HasOBSPassword bool `json:"hasObsPassword"`
	HasSLToken     bool `json:"hasStreamlabsToken"`
}

type StatusResp struct {
	App          string       `json:"app"`
	Version      string       `json:"version"`
	OS           string       `json:"os"`
	IsRemote     bool         `json:"isRemote"`
	IPs          []IPView     `json:"ips"`
	ActiveIP     string       `json:"activeIP"`
	DNS          DNSStatus    `json:"dns"`
	RTMPRunning  bool         `json:"rtmpRunning"`
	RTMPError    string       `json:"rtmpError"`
	RTMPPort     int          `json:"rtmpPort"`
	Devices      []DeviceView `json:"devices"`
	Stream       StreamStatus `json:"stream"`
	LastStream   *LastStream  `json:"lastStream"`
	OBSURL       string       `json:"obsURL"`
	LanOBSURL    string       `json:"lanObsURL"`
	Destinations []DestView   `json:"destinations"`
	TwitchFwd    RelayStatus  `json:"twitchForward"`
	Settings     SettingsView `json:"settings"`
	FirewallFix  bool         `json:"firewallFix"`
	Autostart    ToggleView   `json:"autostart"`
	Remote       RemoteView   `json:"remote"`
	Import       ImportView   `json:"import"`
	Tunnel       TunnelView   `json:"tunnel"`
	Warnings     []string     `json:"warnings"`
}

func maskKey(k string) string {
	k = strings.TrimSpace(k)
	if k == "" {
		return ""
	}
	if len(k) <= 4 {
		return "••••"
	}
	return "••••••" + k[len(k)-4:]
}

type WebServer struct {
	app  *App
	port int
	srv  *http.Server
	mux  *http.ServeMux
}

// localGuard protects the panel on 127.0.0.1: only localhost Host headers are
// accepted (blocks DNS rebinding).
func localGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		if host != "127.0.0.1" && host != "localhost" && host != "[::1]" && host != "::1" {
			http.Error(rw, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(rw, r)
	})
}

// post wraps state-changing endpoints: they need POST plus a custom header that
// browsers won't send cross-origin without a CORS preflight (never allowed).
func post(next http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("X-StreamBridge") != "1" {
			http.Error(rw, "forbidden", http.StatusForbidden)
			return
		}
		next(rw, r)
	}
}

func localOnlyH(next http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		if isRemoteReq(r) {
			http.Error(rw, "only on the computer", http.StatusForbidden)
			return
		}
		next(rw, r)
	}
}

func writeJSON(rw http.ResponseWriter, v any) {
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(rw).Encode(v)
}

func readJSON(r *http.Request, v any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(v)
}

func (w *WebServer) buildMux() {
	mux := http.NewServeMux()
	sub, _ := fs.Sub(webFS, "web")
	static := http.FileServer(http.FS(sub))
	mux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Cache-Control", "no-store")
		rw.Header().Set("X-Frame-Options", "DENY")
		static.ServeHTTP(rw, r)
	})
	mux.HandleFunc("/api/ping", func(rw http.ResponseWriter, r *http.Request) {
		writeJSON(rw, map[string]string{"app": "streambridge", "version": version})
	})
	mux.HandleFunc("/api/status", w.handleStatus)
	mux.HandleFunc("/api/dnslog", func(rw http.ResponseWriter, r *http.Request) {
		writeJSON(rw, w.app.dns.Log(120))
	})
	mux.HandleFunc("/api/settings", post(w.handleSettings))
	mux.HandleFunc("/api/destinations", post(w.handleDestUpsert))
	mux.HandleFunc("/api/destinations/delete", post(w.handleDestDelete))
	mux.HandleFunc("/api/firewall", localOnlyH(post(func(rw http.ResponseWriter, r *http.Request) {
		if err := fixFirewall(); err != nil {
			writeJSON(rw, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(rw, map[string]any{"ok": true})
	})))
	mux.HandleFunc("/api/autostart", localOnlyH(post(func(rw http.ResponseWriter, r *http.Request) {
		var in struct {
			Enabled bool `json:"enabled"`
		}
		_ = readJSON(r, &in)
		err := setAutostart(in.Enabled)
		writeJSON(rw, map[string]any{"ok": err == nil, "error": errString(err)})
	})))
	mux.HandleFunc("/api/remote", localOnlyH(post(func(rw http.ResponseWriter, r *http.Request) {
		var in struct {
			Enabled bool `json:"enabled"`
		}
		_ = readJSON(r, &in)
		_ = w.app.cfg.Update(func(c *Config) { c.RemoteEnabled = in.Enabled })
		w.app.remote.Sync()
		writeJSON(rw, map[string]any{"ok": true})
	})))
	mux.HandleFunc("/api/remote/reset", localOnlyH(post(func(rw http.ResponseWriter, r *http.Request) {
		w.app.remote.Reset()
		writeJSON(rw, map[string]any{"ok": true})
	})))
	mux.HandleFunc("/api/qr.png", localOnlyH(func(rw http.ResponseWriter, r *http.Request) {
		png, err := w.app.remote.qrPNG()
		if err != nil {
			http.Error(rw, err.Error(), http.StatusServiceUnavailable)
			return
		}
		rw.Header().Set("Content-Type", "image/png")
		rw.Header().Set("Cache-Control", "no-store")
		_, _ = rw.Write(png)
	}))
	mux.HandleFunc("/api/tunnel", localOnlyH(post(w.handleTunnel)))
	mux.HandleFunc("/api/tunnel/detect", localOnlyH(post(func(rw http.ResponseWriter, r *http.Request) {
		found := detectProxies()
		resp := map[string]any{"ok": true, "found": found}
		if len(found) > 0 {
			if d, err := testTunnel(found[0].Addr); err == nil {
				resp["twitchOK"], resp["ms"] = true, d.Milliseconds()
			} else {
				resp["twitchOK"], resp["error"] = false, err.Error()
			}
		}
		writeJSON(rw, resp)
	})))
	mux.HandleFunc("/api/tunnel/test", localOnlyH(post(func(rw http.ResponseWriter, r *http.Request) {
		var in struct {
			Proxy string `json:"proxy"`
		}
		_ = readJSON(r, &in)
		p := strings.TrimSpace(in.Proxy)
		if p == "" {
			p = w.app.cfg.Get().TunnelProxy
		}
		d, err := testTunnel(p)
		if err != nil {
			writeJSON(rw, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(rw, map[string]any{"ok": true, "ms": d.Milliseconds()})
	})))
	mux.HandleFunc("/api/import/obs", post(w.handleImportOBS))
	mux.HandleFunc("/api/import/streamlabs", post(w.handleImportSL))
	mux.HandleFunc("/api/quit", post(func(rw http.ResponseWriter, r *http.Request) {
		writeJSON(rw, map[string]any{"ok": true})
		go func() { time.Sleep(300 * time.Millisecond); w.app.quit <- struct{}{} }()
	}))
	mux.HandleFunc("/live.flv", w.handleFLV)
	w.mux = mux
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (w *WebServer) Start(ln net.Listener) {
	w.buildMux()
	w.srv = &http.Server{Handler: localGuard(w.mux), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := w.srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("Web: %v", err)
		}
	}()
}

func (w *WebServer) Stop() {
	if w.srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = w.srv.Shutdown(ctx)
	}
}

func (w *WebServer) handleStatus(rw http.ResponseWriter, r *http.Request) {
	a := w.app
	cfg := a.cfg.Get()
	stream, last, rtmpRunning, rtmpErr := a.stream.Status()
	remote := isRemoteReq(r)

	resp := StatusResp{
		App:         "StreamBridge",
		Version:     version,
		OS:          runtime.GOOS,
		IsRemote:    remote,
		DNS:         a.dns.Status(),
		RTMPRunning: rtmpRunning,
		RTMPError:   rtmpErr,
		RTMPPort:    a.rtmpPort,
		Stream:      stream,
		LastStream:  last,
		TwitchFwd:   a.relays.Status(twitchForwardID),
		Settings:    SettingsView{Lang: cfg.Lang, PreferredIP: cfg.PreferredIP, ForwardToTwitch: cfg.ForwardToTwitch, Upstream: cfg.Upstream},
		FirewallFix: firewallSupported() && !remote,
		Autostart:   ToggleView{Supported: autostartSupported() && !remote, Enabled: autostartEnabled()},
		Remote:      a.remote.View(!remote),
		Import:      ImportView{HasOBSPassword: cfg.OBSPassword != "", HasSLToken: cfg.SLToken != ""},
		Tunnel:      a.tunnel.View(),
		Warnings:    []string{},
	}
	for _, lip := range LocalIPv4s() {
		resp.IPs = append(resp.IPs, IPView{IP: lip.IP.String(), Iface: lip.Iface})
	}
	if ip := a.ActiveIP(); ip != nil {
		resp.ActiveIP = ip.String()
	}
	resp.OBSURL = fmt.Sprintf("rtmp://127.0.0.1:%d/live/ps5", a.rtmpPort)
	if resp.ActiveIP != "" {
		resp.LanOBSURL = fmt.Sprintf("rtmp://%s:%d/live/ps5", resp.ActiveIP, a.rtmpPort)
	}
	now := time.Now()
	for _, d := range a.dns.Devices() {
		dv := DeviceView{IP: d.IP, Kind: d.Kind, LastSeenAgo: now.Sub(d.LastSeen).Seconds(), Queries: d.Queries, AskedTwitch: d.AskedTwitch, TwitchAgo: -1}
		if d.AskedTwitch {
			dv.TwitchAgo = now.Sub(d.LastTwitch).Seconds()
		}
		if d.Kind == "this-pc" && now.Sub(d.LastSeen) < 2*time.Minute {
			resp.Warnings = append(resp.Warnings, "self_dns")
		}
		resp.Devices = append(resp.Devices, dv)
	}
	for _, d := range cfg.Destinations {
		st := a.relays.Status(d.ID)
		resp.Destinations = append(resp.Destinations, DestView{
			ID: d.ID, Platform: d.Platform, Name: d.Name, URL: d.URL,
			KeyMasked: maskKey(d.Key), Enabled: d.Enabled, State: st.State, Error: st.Error,
		})
	}
	if resp.Destinations == nil {
		resp.Destinations = []DestView{}
	}
	if resp.Devices == nil {
		resp.Devices = []DeviceView{}
	}
	if len(resp.IPs) == 0 {
		resp.Warnings = append(resp.Warnings, "no_network")
	}
	writeJSON(rw, resp)
}

func (w *WebServer) handleSettings(rw http.ResponseWriter, r *http.Request) {
	var in struct {
		Lang            *string `json:"lang"`
		PreferredIP     *string `json:"preferredIP"`
		ForwardToTwitch *bool   `json:"forwardToTwitch"`
	}
	if err := readJSON(r, &in); err != nil {
		http.Error(rw, "bad request", http.StatusBadRequest)
		return
	}
	err := w.app.cfg.Update(func(c *Config) {
		if in.Lang != nil && (*in.Lang == "en" || *in.Lang == "fa") {
			c.Lang = *in.Lang
		}
		if in.PreferredIP != nil {
			c.PreferredIP = strings.TrimSpace(*in.PreferredIP)
		}
		if in.ForwardToTwitch != nil {
			c.ForwardToTwitch = *in.ForwardToTwitch
		}
	})
	w.app.relays.Sync()
	writeJSON(rw, map[string]any{"ok": err == nil})
}

var platformDefaults = map[string]struct{ Name, URL string }{
	"kick":    {"Kick", ""},
	"youtube": {"YouTube", "rtmp://a.rtmp.youtube.com/live2"},
	"twitch":  {"Twitch", "rtmp://live.twitch.tv/app"},
	"custom":  {"Custom", ""},
}

func (w *WebServer) handleDestUpsert(rw http.ResponseWriter, r *http.Request) {
	var in Destination
	if err := readJSON(r, &in); err != nil {
		http.Error(rw, "bad request", http.StatusBadRequest)
		return
	}
	in.Platform = strings.ToLower(strings.TrimSpace(in.Platform))
	in.URL = strings.TrimSpace(in.URL)
	in.Key = strings.TrimSpace(in.Key)
	in.Name = strings.TrimSpace(in.Name)

	var errMsg string
	_ = w.app.cfg.Update(func(c *Config) {
		// Update: only fields that were sent are changed.
		for i := range c.Destinations {
			if c.Destinations[i].ID == in.ID && in.ID != "" {
				d := &c.Destinations[i]
				d.Enabled = in.Enabled
				if in.URL != "" {
					d.URL = in.URL
				}
				if in.Key != "" {
					d.Key = in.Key
				}
				if in.Name != "" {
					d.Name = in.Name
				}
				return
			}
		}
		// New destination: fill in platform defaults.
		if _, ok := platformDefaults[in.Platform]; !ok {
			in.Platform = "custom"
		}
		if in.Name == "" {
			in.Name = platformDefaults[in.Platform].Name
		}
		if in.URL == "" {
			in.URL = platformDefaults[in.Platform].URL
		}
		if in.URL == "" || in.Key == "" {
			errMsg = "missing"
			return
		}
		in.ID = newID()
		c.Destinations = append(c.Destinations, in)
	})
	w.app.relays.Sync()
	if errMsg != "" {
		writeJSON(rw, map[string]any{"ok": false, "error": errMsg})
		return
	}
	writeJSON(rw, map[string]any{"ok": true, "id": in.ID})
}

func (w *WebServer) handleDestDelete(rw http.ResponseWriter, r *http.Request) {
	var in struct {
		ID string `json:"id"`
	}
	if err := readJSON(r, &in); err != nil {
		http.Error(rw, "bad request", http.StatusBadRequest)
		return
	}
	_ = w.app.cfg.Update(func(c *Config) {
		out := c.Destinations[:0]
		for _, d := range c.Destinations {
			if d.ID != in.ID {
				out = append(out, d)
			}
		}
		c.Destinations = out
	})
	w.app.relays.Sync()
	writeJSON(rw, map[string]any{"ok": true})
}

func (w *WebServer) handleTunnel(rw http.ResponseWriter, r *http.Request) {
	var in struct {
		Enabled    *bool    `json:"enabled"`
		Proxy      *string  `json:"proxy"`
		Extra      []string `json:"extra"`
		SkipRelays *bool    `json:"skipRelays"`
	}
	if err := readJSON(r, &in); err != nil {
		http.Error(rw, "bad request", http.StatusBadRequest)
		return
	}
	_ = w.app.cfg.Update(func(c *Config) {
		if in.Proxy != nil {
			p := strings.TrimSpace(*in.Proxy)
			p = strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(p, "socks5://"), "http://"), "socks://")
			if p != "" && !strings.Contains(p, ":") {
				p = "127.0.0.1:" + p // just a port number
			}
			c.TunnelProxy = p
		}
		if in.Extra != nil {
			var out []string
			for _, e := range in.Extra {
				if e = normHost(e); e != "" {
					out = append(out, e)
				}
			}
			c.TunnelExtra = out
		}
		if in.SkipRelays != nil {
			c.TunnelSkipRelays = *in.SkipRelays
		}
		if in.Enabled != nil {
			c.TunnelEnabled = *in.Enabled
			if c.TunnelEnabled && c.TunnelProxy == "" {
				if f := detectProxies(); len(f) > 0 {
					c.TunnelProxy = f[0].Addr
				}
			}
		}
	})
	w.app.tunnel.Sync()
	// Reconnect running relays so they use the new route.
	w.app.relays.StopAll()
	w.app.relays.Sync()
	writeJSON(rw, map[string]any{"ok": true})
}

func importResult(rw http.ResponseWriter, scene string, err error) {
	if err == nil {
		writeJSON(rw, map[string]any{"ok": true, "scene": scene})
		return
	}
	var ie *ImportError
	if errors.As(err, &ie) {
		log.Printf("Import: %v", ie)
		writeJSON(rw, map[string]any{"ok": false, "code": ie.Code, "detail": ie.Detail})
		return
	}
	writeJSON(rw, map[string]any{"ok": false, "code": "failed", "detail": err.Error()})
}

func (w *WebServer) handleImportOBS(rw http.ResponseWriter, r *http.Request) {
	var in struct {
		Password *string `json:"password"`
	}
	_ = readJSON(r, &in)
	cfg := w.app.cfg.Get()
	pw := cfg.OBSPassword
	if in.Password != nil {
		pw = strings.TrimSpace(*in.Password)
	}
	port := cfg.OBSPort
	if port == 0 {
		port = 4455
	}
	scene, err := AddToOBS(port, pw, fmt.Sprintf("rtmp://127.0.0.1:%d/live/ps5", w.app.rtmpPort))
	if err == nil && in.Password != nil {
		_ = w.app.cfg.Update(func(c *Config) { c.OBSPassword = pw })
	}
	importResult(rw, scene, err)
}

func (w *WebServer) handleImportSL(rw http.ResponseWriter, r *http.Request) {
	var in struct {
		Token *string `json:"token"`
	}
	_ = readJSON(r, &in)
	cfg := w.app.cfg.Get()
	tok := cfg.SLToken
	if in.Token != nil {
		tok = strings.TrimSpace(*in.Token)
	}
	port := cfg.SLPort
	if port == 0 {
		port = 59650
	}
	scene, err := AddToStreamlabs(port, tok, fmt.Sprintf("rtmp://127.0.0.1:%d/live/ps5", w.app.rtmpPort))
	if err == nil && in.Token != nil {
		_ = w.app.cfg.Update(func(c *Config) { c.SLToken = tok })
	}
	importResult(rw, scene, err)
}

type flushWriter struct {
	w http.ResponseWriter
	f http.Flusher
}

func (fw flushWriter) Write(p []byte) (int, error) { return fw.w.Write(p) }
func (fw flushWriter) Flush() error                { fw.f.Flush(); return nil }

// handleFLV serves the live stream as HTTP-FLV for the in-page preview.
func (w *WebServer) handleFLV(rw http.ResponseWriter, r *http.Request) {
	sess := w.app.stream.WaitSession(r.Context(), 2*time.Second)
	if sess == nil {
		http.Error(rw, "not live", http.StatusServiceUnavailable)
		return
	}
	flusher, ok := rw.(http.Flusher)
	if !ok {
		http.Error(rw, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "video/x-flv")
	rw.Header().Set("Cache-Control", "no-store")
	rw.WriteHeader(http.StatusOK)
	flusher.Flush()
	sess.Viewers.Add(1)
	defer sess.Viewers.Add(-1)
	muxer := flv.NewMuxerWriteFlusher(flushWriter{w: rw, f: flusher})
	_ = avutil.CopyFile(muxer, sess.Queue.Latest())
}
