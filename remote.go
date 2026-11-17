package main

// Phone control: an optional second listener on the LAN that serves the same
// control panel, protected by a 6-digit PIN (or a QR code that logs in
// directly). Off by default; nothing listens on the LAN until it is enabled.

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

const sessionCookie = "sb_session"

type ctxKey int

const ctxRemote ctxKey = 1

func isRemoteReq(r *http.Request) bool {
	v, _ := r.Context().Value(ctxRemote).(bool)
	return v
}

func randomPIN() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%06d", n.Int64())
}

func randomSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func hmacHex(secret, msg string, n int) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(msg))
	s := hex.EncodeToString(m.Sum(nil))
	if n > 0 && n < len(s) {
		return s[:n]
	}
	return s
}

type RemoteServer struct {
	app      *App
	basePort int

	mu      sync.Mutex
	srv     *http.Server
	port    int
	running bool
	errMsg  string
	fails   map[string][]time.Time
}

func NewRemoteServer(app *App, basePort int) *RemoteServer {
	return &RemoteServer{app: app, basePort: basePort, fails: map[string][]time.Time{}}
}

// ensureSecrets creates the PIN and signing secret on first use.
func (rs *RemoteServer) ensureSecrets() {
	c := rs.app.cfg.Get()
	if c.RemotePIN != "" && c.RemoteSecret != "" {
		return
	}
	_ = rs.app.cfg.Update(func(c *Config) {
		if c.RemotePIN == "" {
			c.RemotePIN = randomPIN()
		}
		if c.RemoteSecret == "" {
			c.RemoteSecret = randomSecret()
		}
	})
}

// Reset issues a new PIN and secret, logging out every phone.
func (rs *RemoteServer) Reset() {
	_ = rs.app.cfg.Update(func(c *Config) {
		c.RemotePIN = randomPIN()
		c.RemoteSecret = randomSecret()
	})
}

func (rs *RemoteServer) sessionToken() string {
	return hmacHex(rs.app.cfg.Get().RemoteSecret, "sb-session-v1", 0)
}

func (rs *RemoteServer) qrToken() string {
	return hmacHex(rs.app.cfg.Get().RemoteSecret, "sb-qr-v1", 24)
}

// Sync starts or stops the LAN listener to match the config.
func (rs *RemoteServer) Sync() {
	enabled := rs.app.cfg.Get().RemoteEnabled
	rs.mu.Lock()
	running := rs.running
	rs.mu.Unlock()
	if enabled && !running {
		rs.ensureSecrets()
		rs.start()
	} else if !enabled && running {
		rs.stop()
	}
}

func (rs *RemoteServer) start() {
	var ln net.Listener
	var err error
	port := 0
	for p := rs.basePort; p < rs.basePort+10; p++ {
		ln, err = net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%d", p))
		if err == nil {
			port = p
			break
		}
	}
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if ln == nil {
		rs.errMsg = friendlyBindError(err)
		log.Printf("Phone control: could not listen: %v", err)
		return
	}
	rs.port = port
	rs.srv = &http.Server{Handler: rs.handler(rs.app.web.mux), ReadHeaderTimeout: 10 * time.Second}
	rs.running = true
	rs.errMsg = ""
	srv := rs.srv
	go func() { _ = srv.Serve(ln) }()
	log.Printf("Phone control: listening on port %d", port)
}

func (rs *RemoteServer) stop() {
	rs.mu.Lock()
	srv := rs.srv
	rs.srv = nil
	rs.running = false
	rs.mu.Unlock()
	if srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		log.Printf("Phone control: stopped")
	}
}

type RemoteView struct {
	Enabled bool   `json:"enabled"`
	Running bool   `json:"running"`
	Error   string `json:"error,omitempty"`
	URL     string `json:"url,omitempty"`
	PIN     string `json:"pin,omitempty"`
}

func (rs *RemoteServer) View(local bool) RemoteView {
	c := rs.app.cfg.Get()
	rs.mu.Lock()
	v := RemoteView{Enabled: c.RemoteEnabled, Running: rs.running, Error: rs.errMsg}
	port := rs.port
	rs.mu.Unlock()
	if v.Running {
		if ip := rs.app.ActiveIP(); ip != nil {
			v.URL = fmt.Sprintf("http://%s:%d", ip, port)
		}
	}
	if local {
		v.PIN = c.RemotePIN
	}
	return v
}

func (rs *RemoteServer) loginURL() string {
	v := rs.View(true)
	if v.URL == "" {
		return ""
	}
	return v.URL + "/?k=" + rs.qrToken()
}

// tooManyFails applies a simple rate limit to PIN attempts.
func (rs *RemoteServer) tooManyFails(ip string) bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	cut := time.Now().Add(-5 * time.Minute)
	total := 0
	for k, list := range rs.fails {
		kept := list[:0]
		for _, t := range list {
			if t.After(cut) {
				kept = append(kept, t)
			}
		}
		rs.fails[k] = kept
		total += len(kept)
	}
	return len(rs.fails[ip]) >= 5 || total >= 30
}

func (rs *RemoteServer) addFail(ip string) {
	rs.mu.Lock()
	rs.fails[ip] = append(rs.fails[ip], time.Now())
	rs.mu.Unlock()
}

func (rs *RemoteServer) setSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: rs.sessionToken(), Path: "/",
		MaxAge: 365 * 24 * 3600, HttpOnly: true, SameSite: http.SameSiteStrictMode,
	})
}

func (rs *RemoteServer) authed(r *http.Request) bool {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	return hmac.Equal([]byte(c.Value), []byte(rs.sessionToken()))
}

// localOnly lists paths a phone may never use.
var localOnly = []string{"/api/firewall", "/api/qr.png", "/api/remote", "/api/autostart", "/api/tunnel"}

func (rs *RemoteServer) handler(mux http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		// Only accept requests addressed to an IP (blocks DNS-rebinding tricks).
		if net.ParseIP(strings.Trim(host, "[]")) == nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		clientIP, _, _ := net.SplitHostPort(r.RemoteAddr)
		r = r.WithContext(context.WithValue(r.Context(), ctxRemote, true))
		w.Header().Set("X-Frame-Options", "DENY")

		switch {
		case r.URL.Path == "/login" || r.URL.Path == "/login.html":
			serveEmbedded(w, "web/login.html", "text/html; charset=utf-8")
			return
		case r.URL.Path == "/api/login":
			if r.Method != http.MethodPost || r.Header.Get("X-StreamBridge") != "1" {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			if rs.tooManyFails(clientIP) {
				writeJSON(w, map[string]any{"ok": false, "error": "locked"})
				return
			}
			var in struct {
				PIN string `json:"pin"`
			}
			_ = readJSON(r, &in)
			pin := strings.TrimSpace(in.PIN)
			want := rs.app.cfg.Get().RemotePIN
			if want != "" && hmac.Equal([]byte(pin), []byte(want)) {
				rs.setSession(w)
				writeJSON(w, map[string]any{"ok": true})
				return
			}
			rs.addFail(clientIP)
			time.Sleep(400 * time.Millisecond)
			writeJSON(w, map[string]any{"ok": false, "error": "wrong"})
			return
		}

		if k := r.URL.Query().Get("k"); k != "" && r.URL.Path == "/" {
			if hmac.Equal([]byte(k), []byte(rs.qrToken())) {
				rs.setSession(w)
			} else {
				rs.addFail(clientIP)
			}
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		if !rs.authed(r) {
			if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/live.flv" {
				http.Error(w, "login required", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		for _, p := range localOnly {
			if r.URL.Path == p || strings.HasPrefix(r.URL.Path, p+"/") {
				http.Error(w, "only on the computer", http.StatusForbidden)
				return
			}
		}
		mux.ServeHTTP(w, r)
	})
}

func serveEmbedded(w http.ResponseWriter, name, ctype string) {
	b, err := webFS.ReadFile(name)
	if err != nil {
		http.NotFound(w, nil)
		return
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(b)
}

func (rs *RemoteServer) qrPNG() ([]byte, error) {
	u := rs.loginURL()
	if u == "" {
		return nil, fmt.Errorf("phone control is off")
	}
	return qrcode.Encode(u, qrcode.Medium, 320)
}
