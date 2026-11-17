package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/miekg/dns"
)

// Hostnames the console uses to reach a Twitch ingest server. These are the same
// patterns used by Lightstream's open-source Prism DNS (live(.*).twitch.tv and
// (.*).contribute.live-video.net) and Aioros/console-streaming-server.
// ingest.twitch.tv (the HTTPS API that lists ingest servers) is deliberately NOT matched.
var hijackPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^live[a-z0-9-]*\.twitch\.tv\.$`),
	regexp.MustCompile(`(?i)^([a-z0-9-]+\.)*[a-z0-9-]*contribute\.live-video\.net\.$`),
}

func isHijackName(name string) bool {
	for _, re := range hijackPatterns {
		if re.MatchString(name) {
			return true
		}
	}
	return false
}

type DNSLogEntry struct {
	Time     time.Time `json:"time"`
	Client   string    `json:"client"`
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	Hijacked bool      `json:"hijacked"`
	Tunnel   bool      `json:"tunnel,omitempty"`
	Answer   string    `json:"answer,omitempty"`
}

type DeviceInfo struct {
	IP          string    `json:"ip"`
	Kind        string    `json:"kind"` // playstation, xbox, this-pc, device
	FirstSeen   time.Time `json:"firstSeen"`
	LastSeen    time.Time `json:"lastSeen"`
	Queries     int       `json:"queries"`
	AskedTwitch bool      `json:"askedTwitch"`
	LastTwitch  time.Time `json:"lastTwitch"`
}

type DNSServer struct {
	app  *App
	port int

	mu      sync.Mutex
	running bool
	errMsg  string
	addrs   []string
	servers []*dns.Server
	logBuf  []DNSLogEntry
	devices map[string]*DeviceInfo
}

func NewDNSServer(app *App, port int) *DNSServer {
	return &DNSServer{app: app, port: port, devices: map[string]*DeviceInfo{}}
}

func friendlyBindError(err error) string {
	switch {
	case strings.Contains(err.Error(), "forbidden by its access permissions"):
		// Windows: port held exclusively by another program or reserved by Hyper-V/WSL.
		return "in_use"
	case errors.Is(err, syscall.EACCES) || strings.Contains(strings.ToLower(err.Error()), "permission"):
		return "permission"
	case errors.Is(err, syscall.EADDRINUSE) || strings.Contains(strings.ToLower(err.Error()), "address already in use") || strings.Contains(err.Error(), "Only one usage"):
		return "in_use"
	}
	return err.Error()
}

// Start binds UDP (and TCP when possible) on port 53. It first tries all
// interfaces; if that fails (e.g. systemd-resolved owns 127.0.0.53 on Linux)
// it binds each LAN address separately.
func (d *DNSServer) Start() {
	var bound []string
	var firstErr error

	tryBind := func(host string) bool {
		addr := net.JoinHostPort(host, fmt.Sprint(d.port))
		pc, err := net.ListenPacket("udp4", addr)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return false
		}
		udp := &dns.Server{PacketConn: pc, Handler: d}
		go func() { _ = udp.ActivateAndServe() }()
		d.servers = append(d.servers, udp)
		if ln, err := net.Listen("tcp4", addr); err == nil {
			tcp := &dns.Server{Listener: ln, Handler: d}
			go func() { _ = tcp.ActivateAndServe() }()
			d.servers = append(d.servers, tcp)
		}
		bound = append(bound, addr)
		return true
	}

	if !tryBind("0.0.0.0") {
		for _, lip := range LocalIPv4s() {
			tryBind(lip.IP.String())
		}
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	d.addrs = bound
	if len(bound) == 0 {
		d.running = false
		if firstErr != nil {
			d.errMsg = friendlyBindError(firstErr)
		} else {
			d.errMsg = "no_network"
		}
		log.Printf("DNS: could not start on port %d: %v", d.port, firstErr)
		return
	}
	d.running = true
	d.errMsg = ""
	log.Printf("DNS: listening on %s", strings.Join(bound, ", "))
}

func (d *DNSServer) Stop() {
	for _, s := range d.servers {
		_ = s.Shutdown()
	}
}

func classifyQuery(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "playstation"), strings.Contains(n, "sonyentertainmentnetwork"), strings.Contains(n, ".scea.com"):
		return "playstation"
	case strings.Contains(n, "xboxlive"), strings.Contains(n, "xbox.com"), strings.Contains(n, "xboxab.com"):
		return "xbox"
	}
	return ""
}

func (d *DNSServer) record(client net.IP, name string, qtype uint16, hijacked bool, answer string) {
	d.recordX(client, name, qtype, hijacked, false, answer)
}

func (d *DNSServer) recordX(client net.IP, name string, qtype uint16, hijacked, tunnel bool, answer string) {
	now := time.Now()
	ipStr := client.String()
	d.mu.Lock()
	defer d.mu.Unlock()
	dev := d.devices[ipStr]
	if dev == nil {
		dev = &DeviceInfo{IP: ipStr, Kind: "device", FirstSeen: now}
		if isLocalAddress(client) {
			dev.Kind = "this-pc"
		}
		d.devices[ipStr] = dev
	}
	dev.LastSeen = now
	dev.Queries++
	if k := classifyQuery(name); k != "" && dev.Kind == "device" {
		dev.Kind = k
	}
	if hijacked {
		dev.AskedTwitch = true
		dev.LastTwitch = now
	}
	d.logBuf = append(d.logBuf, DNSLogEntry{Time: now, Client: ipStr, Name: strings.TrimSuffix(name, "."), Type: dns.TypeToString[qtype], Hijacked: hijacked, Tunnel: tunnel, Answer: answer})
	if len(d.logBuf) > 300 {
		d.logBuf = d.logBuf[len(d.logBuf)-300:]
	}
}

func (d *DNSServer) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	var client net.IP
	switch a := w.RemoteAddr().(type) {
	case *net.UDPAddr:
		client = a.IP
	case *net.TCPAddr:
		client = a.IP
	}
	if len(r.Question) == 0 || r.Opcode != dns.OpcodeQuery {
		m := new(dns.Msg)
		m.SetRcode(r, dns.RcodeNotImplemented)
		_ = w.WriteMsg(m)
		return
	}
	q := r.Question[0]

	rtmpName := isHijackName(q.Name)
	tunnelName := !rtmpName && d.app.tunnelDomain(q.Name)
	if rtmpName || tunnelName {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true
		m.RecursionAvailable = true
		answer := ""
		if q.Qtype == dns.TypeA || q.Qtype == dns.TypeANY {
			ip := d.app.IPForClient(client)
			if ip != nil {
				answer = ip.String()
				m.Answer = append(m.Answer, &dns.A{
					Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 5},
					A:   ip.To4(),
				})
			}
		}
		// AAAA / HTTPS / anything else: empty answer, so the console uses IPv4 (our A record).
		d.recordX(client, q.Name, q.Qtype, true, tunnelName, answer)
		if answer != "" && rtmpName {
			log.Printf("DNS: %s asked for %s -> redirected to %s", client, strings.TrimSuffix(q.Name, "."), answer)
		}
		_ = w.WriteMsg(m)
		return
	}

	d.record(client, q.Name, q.Qtype, false, "")
	resp, err := d.forward(r)
	if err != nil || resp == nil {
		m := new(dns.Msg)
		m.SetRcode(r, dns.RcodeServerFailure)
		_ = w.WriteMsg(m)
		return
	}
	resp.Id = r.Id
	if _, isUDP := w.RemoteAddr().(*net.UDPAddr); isUDP {
		size := dns.MinMsgSize
		if opt := r.IsEdns0(); opt != nil && int(opt.UDPSize()) > size {
			size = int(opt.UDPSize())
		}
		resp.Truncate(size)
	}
	_ = w.WriteMsg(resp)
}

func (d *DNSServer) forward(r *dns.Msg) (*dns.Msg, error) {
	upstreams := d.app.cfg.Get().Upstream
	var lastErr error
	for _, up := range upstreams {
		if _, _, err := net.SplitHostPort(up); err != nil {
			up = net.JoinHostPort(up, "53")
		}
		c := &dns.Client{Net: "udp", Timeout: 2500 * time.Millisecond}
		resp, _, err := c.Exchange(r, up)
		if err == nil && resp != nil && resp.Truncated {
			tc := &dns.Client{Net: "tcp", Timeout: 3 * time.Second}
			if r2, _, err2 := tc.Exchange(r, up); err2 == nil {
				resp = r2
			}
		}
		if err == nil && resp != nil {
			return resp, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

type DNSStatus struct {
	Running bool     `json:"running"`
	Error   string   `json:"error"`
	Addrs   []string `json:"addrs"`
	Port    int      `json:"port"`
}

func (d *DNSServer) Status() DNSStatus {
	d.mu.Lock()
	defer d.mu.Unlock()
	return DNSStatus{Running: d.running, Error: d.errMsg, Addrs: append([]string(nil), d.addrs...), Port: d.port}
}

func (d *DNSServer) Devices() []DeviceInfo {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]DeviceInfo, 0, len(d.devices))
	for _, dev := range d.devices {
		out = append(out, *dev)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastSeen.After(out[j].LastSeen) })
	return out
}

func (d *DNSServer) Log(n int) []DNSLogEntry {
	d.mu.Lock()
	defer d.mu.Unlock()
	if n <= 0 || n > len(d.logBuf) {
		n = len(d.logBuf)
	}
	src := d.logBuf[len(d.logBuf)-n:]
	out := make([]DNSLogEntry, n)
	for i := range src {
		out[i] = src[len(src)-1-i] // newest first
	}
	return out
}
