package main

// Iran mode: the PS5 cannot reach Twitch (filtered) and cannot run a VPN.
// With only the PS5's DNS pointed here, StreamBridge answers Twitch's API/login
// hostnames with this computer's address, then forwards each connection —
// still encrypted, untouched — through the local V2ray (or similar) SOCKS5 or
// HTTP proxy. The server name is read from the TLS ClientHello (SNI) or the
// HTTP Host header; only allow-listed names are forwarded.

import (
	"bufio"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Twitch hostnames (API, login, chat, CDN, ingest list). RTMP ingest names are
// also covered; the PS5 reaches those on port 1935, which stays local.
var tunnelSuffixes = []string{"twitch.tv", "ttvnw.net", "jtvnw.net", "twitchcdn.net", "twitchsvc.net", "ext-twitch.tv", "live-video.net", "twitch.a2z.com"}

func normHost(h string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(h)), ".")
}

func matchSuffix(host, suffix string) bool {
	suffix = normHost(strings.TrimPrefix(strings.TrimSpace(suffix), "*."))
	return suffix != "" && (host == suffix || strings.HasSuffix(host, "."+suffix))
}

// tunnelDomain reports whether a hostname should go through the tunnel.
func (a *App) tunnelDomain(name string) bool {
	if a.tunnel == nil || !a.tunnel.Active() {
		return false
	}
	host := normHost(name)
	for _, s := range tunnelSuffixes {
		if matchSuffix(host, s) {
			return true
		}
	}
	for _, s := range a.cfg.Get().TunnelExtra {
		if matchSuffix(host, s) {
			return true
		}
	}
	return false
}

// ------------------------------------------------------------ proxy dialing

var proxyKind sync.Map // proxy addr -> "socks5" | "http"

func dialProxy(proxyAddr, target string, timeout time.Duration) (net.Conn, error) {
	if proxyAddr == "" {
		return nil, errors.New("no proxy set")
	}
	if k, ok := proxyKind.Load(proxyAddr); ok && k == "http" {
		return dialHTTPConnect(proxyAddr, target, timeout)
	}
	c, err := dialSOCKS5(proxyAddr, target, timeout)
	if err == errNotSOCKS {
		if c2, err2 := dialHTTPConnect(proxyAddr, target, timeout); err2 == nil {
			proxyKind.Store(proxyAddr, "http")
			return c2, nil
		} else {
			return nil, err2
		}
	}
	if err == nil {
		proxyKind.Store(proxyAddr, "socks5")
	}
	return c, err
}

var errNotSOCKS = errors.New("not a SOCKS5 proxy")

func dialSOCKS5(proxyAddr, target string, timeout time.Duration) (net.Conn, error) {
	host, portStr, err := net.SplitHostPort(target)
	if err != nil {
		return nil, err
	}
	port, _ := strconv.Atoi(portStr)
	c, err := net.DialTimeout("tcp", proxyAddr, timeout)
	if err != nil {
		return nil, fmt.Errorf("proxy %s not reachable: %w", proxyAddr, err)
	}
	// A SOCKS server answers the greeting at once; an HTTP proxy never does.
	greet := 2 * time.Second
	if timeout < greet {
		greet = timeout
	}
	_ = c.SetDeadline(time.Now().Add(greet))
	if _, err := c.Write([]byte{5, 1, 0}); err != nil {
		c.Close()
		return nil, err
	}
	hello := make([]byte, 2)
	if _, err := io.ReadFull(c, hello); err != nil {
		c.Close()
		return nil, errNotSOCKS
	}
	if hello[0] != 5 {
		c.Close()
		return nil, errNotSOCKS
	}
	if hello[1] != 0 {
		c.Close()
		return nil, errors.New("SOCKS proxy wants a password (not supported)")
	}
	_ = c.SetDeadline(time.Now().Add(timeout))
	req := []byte{5, 1, 0}
	if ip := net.ParseIP(host); ip != nil && ip.To4() != nil {
		req = append(req, 1)
		req = append(req, ip.To4()...)
	} else {
		if len(host) > 255 {
			c.Close()
			return nil, errors.New("hostname too long")
		}
		req = append(req, 3, byte(len(host)))
		req = append(req, host...)
	}
	req = binary.BigEndian.AppendUint16(req, uint16(port))
	if _, err := c.Write(req); err != nil {
		c.Close()
		return nil, err
	}
	head := make([]byte, 4)
	if _, err := io.ReadFull(c, head); err != nil {
		c.Close()
		return nil, err
	}
	if head[1] != 0 {
		c.Close()
		return nil, fmt.Errorf("proxy could not reach %s (SOCKS error %d)", target, head[1])
	}
	var skip int
	switch head[3] {
	case 1:
		skip = 4 + 2
	case 4:
		skip = 16 + 2
	case 3:
		l := make([]byte, 1)
		if _, err := io.ReadFull(c, l); err != nil {
			c.Close()
			return nil, err
		}
		skip = int(l[0]) + 2
	}
	if _, err := io.ReadFull(c, make([]byte, skip)); err != nil {
		c.Close()
		return nil, err
	}
	_ = c.SetDeadline(time.Time{})
	return c, nil
}

func dialHTTPConnect(proxyAddr, target string, timeout time.Duration) (net.Conn, error) {
	c, err := net.DialTimeout("tcp", proxyAddr, timeout)
	if err != nil {
		return nil, fmt.Errorf("proxy %s not reachable: %w", proxyAddr, err)
	}
	_ = c.SetDeadline(time.Now().Add(timeout))
	fmt.Fprintf(c, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", target, target)
	// Read the response head byte by byte so no tunnel data is consumed.
	var head []byte
	b := make([]byte, 1)
	for len(head) < 8192 {
		if _, err := c.Read(b); err != nil {
			c.Close()
			return nil, err
		}
		head = append(head, b[0])
		if strings.HasSuffix(string(head), "\r\n\r\n") {
			break
		}
	}
	line := strings.SplitN(string(head), "\r\n", 2)[0]
	f := strings.Fields(line)
	if len(f) < 2 || !strings.HasPrefix(f[0], "HTTP/") || f[1] != "200" {
		c.Close()
		return nil, fmt.Errorf("proxy refused %s: %s", target, line)
	}
	_ = c.SetDeadline(time.Time{})
	return c, nil
}

// probeProxy checks whether something at addr speaks SOCKS5 or HTTP CONNECT.
func probeProxy(addr string) string {
	c, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
	if err != nil {
		return ""
	}
	_ = c.SetDeadline(time.Now().Add(700 * time.Millisecond))
	_, _ = c.Write([]byte{5, 1, 0})
	b := make([]byte, 2)
	n, _ := io.ReadFull(c, b)
	c.Close()
	if n == 2 && b[0] == 5 {
		return "socks5"
	}
	c, err = net.DialTimeout("tcp", addr, 300*time.Millisecond)
	if err != nil {
		return ""
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(700 * time.Millisecond))
	fmt.Fprintf(c, "CONNECT example.com:443 HTTP/1.1\r\nHost: example.com:443\r\n\r\n")
	line, _ := bufio.NewReader(c).ReadString('\n')
	if strings.HasPrefix(line, "HTTP/") {
		return "http"
	}
	return ""
}

// Local proxy ports of common V2ray-family apps (v2rayN, NekoRay/NekoBox,
// Hiddify, Clash, v2rayA, generic SOCKS).
var commonProxyPorts = []int{10808, 10809, 2080, 12334, 7890, 7891, 20170, 20171, 1080, 10810}

type FoundProxy struct {
	Addr string `json:"addr"`
	Kind string `json:"kind"`
}

func detectProxies() []FoundProxy {
	var mu sync.Mutex
	var wg sync.WaitGroup
	found := []FoundProxy{}
	for _, p := range commonProxyPorts {
		addr := fmt.Sprintf("127.0.0.1:%d", p)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if k := probeProxy(addr); k != "" {
				mu.Lock()
				found = append(found, FoundProxy{Addr: addr, Kind: k})
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	// keep the order of commonProxyPorts (SOCKS ports first)
	order := map[string]int{}
	for i, p := range commonProxyPorts {
		order[fmt.Sprintf("127.0.0.1:%d", p)] = i
	}
	for i := 1; i < len(found); i++ {
		for j := i; j > 0 && order[found[j].Addr] < order[found[j-1].Addr]; j-- {
			found[j], found[j-1] = found[j-1], found[j]
		}
	}
	return found
}

// testTunnel opens a real TLS connection to Twitch through the proxy.
func testTunnel(proxyAddr string) (time.Duration, error) {
	start := time.Now()
	c, err := dialProxy(proxyAddr, "id.twitch.tv:443", 8*time.Second)
	if err != nil {
		return 0, err
	}
	defer c.Close()
	tc := tls.Client(c, &tls.Config{ServerName: "id.twitch.tv"})
	_ = tc.SetDeadline(time.Now().Add(8 * time.Second))
	if err := tc.Handshake(); err != nil {
		return 0, fmt.Errorf("Twitch did not answer through the proxy: %w", err)
	}
	return time.Since(start), nil
}

// ------------------------------------------------------------ SNI / Host

var errNotTLS = errors.New("not TLS")

// readClientHello reads the TLS ClientHello (possibly spread over several
// records) and returns the SNI plus every byte read, to be replayed upstream.
func readClientHello(r io.Reader) (string, []byte, error) {
	var raw, hs []byte
	hdr := make([]byte, 5)
	for {
		if _, err := io.ReadFull(r, hdr); err != nil {
			return "", raw, err
		}
		raw = append(raw, hdr...)
		if hdr[0] != 0x16 {
			return "", raw, errNotTLS
		}
		n := int(binary.BigEndian.Uint16(hdr[3:5]))
		if n == 0 || n > 18432 {
			return "", raw, errors.New("bad TLS record")
		}
		body := make([]byte, n)
		if _, err := io.ReadFull(r, body); err != nil {
			return "", raw, err
		}
		raw = append(raw, body...)
		hs = append(hs, body...)
		if len(hs) >= 4 {
			need := 4 + (int(hs[1])<<16 | int(hs[2])<<8 | int(hs[3]))
			if len(hs) >= need {
				break
			}
			if need > 65536 {
				return "", raw, errors.New("ClientHello too large")
			}
		}
	}
	sni, err := parseSNI(hs)
	return sni, raw, err
}

func parseSNI(hs []byte) (string, error) {
	bad := errors.New("malformed ClientHello")
	if len(hs) < 4 || hs[0] != 1 {
		return "", bad
	}
	p := hs[4:]
	if len(p) < 34 {
		return "", bad
	}
	p = p[34:] // version + random
	if len(p) < 1 || len(p) < 1+int(p[0]) {
		return "", bad
	}
	p = p[1+int(p[0]):] // session id
	if len(p) < 2 {
		return "", bad
	}
	cs := int(binary.BigEndian.Uint16(p))
	if len(p) < 2+cs {
		return "", bad
	}
	p = p[2+cs:] // cipher suites
	if len(p) < 1 || len(p) < 1+int(p[0]) {
		return "", bad
	}
	p = p[1+int(p[0]):] // compression
	if len(p) < 2 {
		return "", nil // no extensions
	}
	el := int(binary.BigEndian.Uint16(p))
	p = p[2:]
	if len(p) < el {
		return "", bad
	}
	p = p[:el]
	for len(p) >= 4 {
		typ := binary.BigEndian.Uint16(p)
		l := int(binary.BigEndian.Uint16(p[2:]))
		if len(p) < 4+l {
			return "", bad
		}
		data := p[4 : 4+l]
		p = p[4+l:]
		if typ != 0 || len(data) < 2 {
			continue
		}
		data = data[2:] // list length
		for len(data) >= 3 {
			nt := data[0]
			nl := int(binary.BigEndian.Uint16(data[1:]))
			if len(data) < 3+nl {
				return "", bad
			}
			if nt == 0 {
				return string(data[3 : 3+nl]), nil
			}
			data = data[3+nl:]
		}
	}
	return "", nil
}

// readHTTPHost reads an HTTP request head and returns the Host plus the raw bytes.
func readHTTPHost(r io.Reader) (string, []byte, error) {
	var raw []byte
	b := make([]byte, 1)
	for len(raw) < 16384 {
		if _, err := r.Read(b); err != nil {
			return "", raw, err
		}
		raw = append(raw, b[0])
		if len(raw) >= 4 && string(raw[len(raw)-4:]) == "\r\n\r\n" {
			break
		}
	}
	for _, line := range strings.Split(string(raw), "\r\n") {
		if len(line) > 5 && strings.EqualFold(line[:5], "host:") {
			h := strings.TrimSpace(line[5:])
			if hh, _, err := net.SplitHostPort(h); err == nil {
				h = hh
			}
			return h, raw, nil
		}
	}
	return "", raw, errors.New("no Host header")
}

// ------------------------------------------------------------ forwarder

type TunnelServer struct {
	app       *App
	httpsPort int
	httpPort  int

	mu        sync.Mutex
	lns       []net.Listener
	running   bool
	err443    string
	err80     string
	lastHost  string
	lastErr   string
	lastErrAt time.Time
	lastOKAt  time.Time
	proxyUp   bool
	proxyKind string
	stop      chan struct{}
	logged    map[string]bool

	active atomic.Int32
	total  atomic.Int64
}

func NewTunnelServer(app *App) *TunnelServer {
	return &TunnelServer{app: app, httpsPort: 443, httpPort: 80, logged: map[string]bool{}}
}

// Active is true when Iran mode is on and the HTTPS forwarder is listening.
func (t *TunnelServer) Active() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running && t.app.cfg.Get().TunnelEnabled
}

func (t *TunnelServer) Sync() {
	enabled := t.app.cfg.Get().TunnelEnabled
	t.mu.Lock()
	running := t.running
	t.mu.Unlock()
	if enabled && !running {
		t.start()
	} else if !enabled && running {
		t.stopAll()
	}
}

func (t *TunnelServer) start() {
	ln443, err := net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%d", t.httpsPort))
	t.mu.Lock()
	defer t.mu.Unlock()
	if err != nil {
		t.err443 = friendlyBindError(err)
		log.Printf("Iran mode: cannot listen on port 443: %v", err)
		return
	}
	t.err443 = ""
	t.lns = []net.Listener{ln443}
	go t.serve(ln443, true)
	if ln80, err := net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%d", t.httpPort)); err == nil {
		t.lns = append(t.lns, ln80)
		t.err80 = ""
		go t.serve(ln80, false)
	} else {
		t.err80 = friendlyBindError(err) // HTTP is rarely needed; HTTPS is what matters
	}
	t.running = true
	t.stop = make(chan struct{})
	go t.healthLoop(t.stop)
	log.Printf("Iran mode: forwarding Twitch through %s", t.app.cfg.Get().TunnelProxy)
}

func (t *TunnelServer) stopAll() {
	t.mu.Lock()
	for _, l := range t.lns {
		l.Close()
	}
	t.lns = nil
	t.running = false
	if t.stop != nil {
		close(t.stop)
		t.stop = nil
	}
	t.mu.Unlock()
	log.Printf("Iran mode: off")
}

func (t *TunnelServer) healthLoop(stop chan struct{}) {
	check := func() {
		addr := t.app.cfg.Get().TunnelProxy
		k := probeProxy(addr)
		if k != "" {
			proxyKind.Store(addr, k)
		}
		t.mu.Lock()
		t.proxyUp, t.proxyKind = k != "", k
		t.mu.Unlock()
	}
	check()
	tk := time.NewTicker(10 * time.Second)
	defer tk.Stop()
	for {
		select {
		case <-stop:
			return
		case <-tk.C:
			check()
		}
	}
}

func (t *TunnelServer) noteErr(msg string) {
	t.mu.Lock()
	t.lastErr, t.lastErrAt = msg, time.Now()
	t.mu.Unlock()
}

func (t *TunnelServer) serve(ln net.Listener, tlsPort bool) {
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		go t.handle(c, tlsPort)
	}
}

func (t *TunnelServer) handle(c net.Conn, tlsPort bool) {
	defer c.Close()
	_ = c.SetReadDeadline(time.Now().Add(10 * time.Second))
	var host string
	var raw []byte
	var err error
	port := "443"
	if tlsPort {
		host, raw, err = readClientHello(c)
	} else {
		port = "80"
		host, raw, err = readHTTPHost(c)
	}
	if err != nil || host == "" {
		return
	}
	host = normHost(host)
	if !t.app.tunnelDomain(host) {
		log.Printf("Iran mode: refused %s (not a Twitch address)", host)
		return
	}
	up, err := dialProxy(t.app.cfg.Get().TunnelProxy, net.JoinHostPort(host, port), 10*time.Second)
	if err != nil {
		t.noteErr(err.Error())
		log.Printf("Iran mode: %s: %v", host, err)
		return
	}
	defer up.Close()
	if _, err := up.Write(raw); err != nil {
		return
	}
	_ = c.SetReadDeadline(time.Time{})
	t.active.Add(1)
	t.total.Add(1)
	defer t.active.Add(-1)
	t.mu.Lock()
	t.lastHost, t.lastOKAt = host, time.Now()
	first := !t.logged[host]
	t.logged[host] = true
	t.mu.Unlock()
	if first {
		log.Printf("Iran mode: %s -> through the tunnel", host)
	}
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(up, c); done <- struct{}{} }()
	go func() { _, _ = io.Copy(c, up); done <- struct{}{} }()
	<-done
}

type TunnelView struct {
	Enabled    bool     `json:"enabled"`
	Proxy      string   `json:"proxy"`
	Extra      []string `json:"extra"`
	SkipRelays bool     `json:"skipRelays"`
	Running    bool     `json:"running"`
	Err443     string   `json:"err443,omitempty"`
	Err80      string   `json:"err80,omitempty"`
	ProxyUp    bool     `json:"proxyUp"`
	ProxyKind  string   `json:"proxyKind"`
	Active     int      `json:"active"`
	Total      int64    `json:"total"`
	LastHost   string   `json:"lastHost"`
	LastErr    string   `json:"lastErr,omitempty"`
	LastErrAgo float64  `json:"lastErrAgo"`
	LastOKAgo  float64  `json:"lastOkAgo"`
}

func (t *TunnelServer) View() TunnelView {
	c := t.app.cfg.Get()
	t.mu.Lock()
	defer t.mu.Unlock()
	v := TunnelView{Enabled: c.TunnelEnabled, Proxy: c.TunnelProxy, Extra: c.TunnelExtra, SkipRelays: c.TunnelSkipRelays,
		Running: t.running, Err443: t.err443, Err80: t.err80, ProxyUp: t.proxyUp, ProxyKind: t.proxyKind,
		Active: int(t.active.Load()), Total: t.total.Load(), LastHost: t.lastHost, LastErrAgo: -1, LastOKAgo: -1}
	if v.Extra == nil {
		v.Extra = []string{}
	}
	if !t.lastErrAt.IsZero() && time.Since(t.lastErrAt) < 10*time.Minute {
		v.LastErr, v.LastErrAgo = t.lastErr, time.Since(t.lastErrAt).Seconds()
	}
	if !t.lastOKAt.IsZero() {
		v.LastOKAgo = time.Since(t.lastOKAt).Seconds()
	}
	return v
}

// relayViaTunnel reports whether platform relays should use the proxy.
func (a *App) relayViaTunnel() (string, bool) {
	c := a.cfg.Get()
	if c.TunnelEnabled && !c.TunnelSkipRelays && c.TunnelProxy != "" {
		return c.TunnelProxy, true
	}
	return "", false
}
