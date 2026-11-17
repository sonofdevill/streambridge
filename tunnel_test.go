package main

import (
	"bufio"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeSOCKS is a minimal SOCKS5 (no auth) server; route maps "host:port" to a real address.
func fakeSOCKS(t *testing.T, route map[string]string) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				b := make([]byte, 3)
				if _, err := io.ReadFull(c, b); err != nil || b[0] != 5 {
					return
				}
				c.Write([]byte{5, 0})
				h := make([]byte, 4)
				io.ReadFull(c, h)
				var host string
				switch h[3] {
				case 3:
					l := make([]byte, 1)
					io.ReadFull(c, l)
					hb := make([]byte, l[0])
					io.ReadFull(c, hb)
					host = string(hb)
				case 1:
					ip := make([]byte, 4)
					io.ReadFull(c, ip)
					host = net.IP(ip).String()
				}
				pb := make([]byte, 2)
				io.ReadFull(c, pb)
				target := fmt.Sprintf("%s:%d", host, binary.BigEndian.Uint16(pb))
				dst, ok := route[target]
				if !ok {
					c.Write([]byte{5, 4, 0, 1, 0, 0, 0, 0, 0, 0})
					return
				}
				up, err := net.Dial("tcp", dst)
				if err != nil {
					c.Write([]byte{5, 5, 0, 1, 0, 0, 0, 0, 0, 0})
					return
				}
				defer up.Close()
				c.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})
				go io.Copy(up, c)
				io.Copy(c, up)
			}(c)
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return ln.Addr().String()
}

// fakeHTTPProxy is a minimal HTTP CONNECT proxy.
func fakeHTTPProxy(t *testing.T, route map[string]string) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				br := bufio.NewReader(c)
				line, _ := br.ReadString('\n')
				for {
					l, err := br.ReadString('\n')
					if err != nil || l == "\r\n" {
						break
					}
				}
				f := strings.Fields(line)
				if len(f) < 2 || f[0] != "CONNECT" {
					c.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
					return
				}
				dst, ok := route[f[1]]
				if !ok {
					c.Write([]byte("HTTP/1.1 403 Forbidden\r\n\r\n"))
					return
				}
				up, err := net.Dial("tcp", dst)
				if err != nil {
					return
				}
				defer up.Close()
				c.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
				go io.Copy(up, br)
				io.Copy(c, up)
			}(c)
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return ln.Addr().String()
}

func TestReadClientHelloSNI(t *testing.T) {
	a, b := net.Pipe()
	go func() {
		tc := tls.Client(a, &tls.Config{ServerName: "id.twitch.tv"})
		_ = tc.Handshake()
	}()
	b.SetDeadline(time.Now().Add(3 * time.Second))
	sni, raw, err := readClientHello(b)
	if err != nil || sni != "id.twitch.tv" || len(raw) < 100 {
		t.Fatalf("sni=%q len=%d err=%v", sni, len(raw), err)
	}
	b.Close()
}

func TestTunnelDomains(t *testing.T) {
	app := &App{cfg: LoadConfig("/nonexistent/x.json")}
	app.tunnel = NewTunnelServer(app)
	app.cfg.cfg.TunnelEnabled = true
	app.cfg.cfg.TunnelExtra = []string{"playstation.net"}
	app.tunnel.running = true
	for _, h := range []string{"id.twitch.tv.", "api.twitch.tv", "irc-ws.chat.twitch.tv", "static-cdn.jtvnw.net", "usher.ttvnw.net", "ps4-system.sec.np.dl.playstation.net"} {
		if !app.tunnelDomain(h) {
			t.Errorf("expected tunnel for %s", h)
		}
	}
	for _, h := range []string{"www.google.com", "nottwitch.tv", "twitch.tv.evil.com", "kick.com"} {
		if app.tunnelDomain(h) {
			t.Errorf("did not expect tunnel for %s", h)
		}
	}
	app.cfg.cfg.TunnelEnabled = false
	if app.tunnelDomain("id.twitch.tv") {
		t.Error("tunnel must be off when disabled")
	}
}

// End to end: a "PS5" opens TLS to this computer with SNI id.twitch.tv; the
// forwarder sends it through a SOCKS (and an HTTP) proxy to the real server.
func TestForwarderEndToEnd(t *testing.T) {
	twitch := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello from twitch, host=%s", r.Host)
	}))
	defer twitch.Close()
	route := map[string]string{"id.twitch.tv:443": twitch.Listener.Addr().String()}

	for _, kind := range []string{"socks5", "http"} {
		var proxy string
		if kind == "socks5" {
			proxy = fakeSOCKS(t, route)
		} else {
			proxy = fakeHTTPProxy(t, route)
		}
		app := &App{cfg: LoadConfig("/nonexistent/x.json")}
		app.cfg.cfg.TunnelEnabled = true
		app.cfg.cfg.TunnelProxy = proxy
		app.tunnel = NewTunnelServer(app)
		ln, _ := net.Listen("tcp", "127.0.0.1:0")
		app.tunnel.httpsPort = ln.Addr().(*net.TCPAddr).Port
		ln.Close()
		ln80, _ := net.Listen("tcp", "127.0.0.1:0")
		app.tunnel.httpPort = ln80.Addr().(*net.TCPAddr).Port
		ln80.Close()
		app.tunnel.Sync()
		if !app.tunnel.Active() {
			t.Fatalf("%s: forwarder not active: %+v", kind, app.tunnel.View())
		}

		// Raw TLS so we control SNI (id.twitch.tv) while verifying the test cert.
		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", app.tunnel.httpsPort))
		if err != nil {
			t.Fatal(err)
		}
		tc := tls.Client(conn, &tls.Config{ServerName: "id.twitch.tv", InsecureSkipVerify: true})
		if err := tc.Handshake(); err != nil {
			t.Fatalf("%s: handshake through forwarder: %v", kind, err)
		}
		fmt.Fprintf(tc, "GET / HTTP/1.1\r\nHost: id.twitch.tv\r\nConnection: close\r\n\r\n")
		body, _ := io.ReadAll(tc)
		if !strings.Contains(string(body), "hello from twitch, host=id.twitch.tv") {
			t.Fatalf("%s: unexpected response: %q", kind, body)
		}
		if v := app.tunnel.View(); v.Total < 1 || v.LastHost != "id.twitch.tv" {
			t.Fatalf("%s: stats not updated: %+v", kind, v)
		}

		// A non-Twitch name must be refused (no open proxy).
		conn2, _ := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", app.tunnel.httpsPort))
		tc2 := tls.Client(conn2, &tls.Config{ServerName: "www.example.org", InsecureSkipVerify: true})
		tc2.SetDeadline(time.Now().Add(3 * time.Second))
		if err := tc2.Handshake(); err == nil {
			t.Fatalf("%s: non-Twitch SNI should be refused", kind)
		}
		app.tunnel.stopAll()
	}
}

func TestProbeAndDetectKinds(t *testing.T) {
	s := fakeSOCKS(t, nil)
	h := fakeHTTPProxy(t, nil)
	if k := probeProxy(s); k != "socks5" {
		t.Errorf("socks probe = %q", k)
	}
	if k := probeProxy(h); k != "http" {
		t.Errorf("http probe = %q", k)
	}
	if k := probeProxy("127.0.0.1:1"); k != "" {
		t.Errorf("closed port probe = %q", k)
	}
}
