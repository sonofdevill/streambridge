package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/datarhei/joy4/av/avutil"
	"github.com/datarhei/joy4/format/rtmp"
)

const twitchForwardID = "twitch-ps5"

type RelayStatus struct {
	State string    `json:"state"` // idle, connecting, live, retrying, error
	Error string    `json:"error,omitempty"`
	Since time.Time `json:"-"`
}

type relayRun struct {
	target string
	cancel context.CancelFunc
	sessID int64
}

type RelayManager struct {
	app    *App
	mu     sync.Mutex
	runs   map[string]*relayRun
	status map[string]*RelayStatus
}

func NewRelayManager(app *App) *RelayManager {
	return &RelayManager{app: app, runs: map[string]*relayRun{}, status: map[string]*RelayStatus{}}
}

// targetURL builds the full publish URL (server URL + stream key).
func targetURL(d Destination) (string, error) {
	raw := strings.TrimSpace(d.URL)
	if raw == "" {
		return "", fmt.Errorf("missing server URL")
	}
	if !strings.HasPrefix(raw, "rtmp://") && !strings.HasPrefix(raw, "rtmps://") {
		return "", fmt.Errorf("server URL must start with rtmp:// or rtmps://")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid server URL")
	}
	// Kick shows its URL without the /app path, which encoders need.
	if strings.Trim(u.Path, "/") == "" {
		u.Path = "/app"
	}
	key := strings.TrimSpace(d.Key)
	if key == "" {
		return "", fmt.Errorf("missing stream key")
	}
	return strings.TrimRight(u.String(), "/") + "/" + key, nil
}

func (m *RelayManager) setStatus(id, state, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.status[id]
	if st == nil {
		st = &RelayStatus{}
		m.status[id] = st
	}
	if st.State != state || st.Error != errMsg {
		st.Since = time.Now()
	}
	st.State, st.Error = state, errMsg
}

func (m *RelayManager) Status(id string) RelayStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	if st := m.status[id]; st != nil {
		return *st
	}
	return RelayStatus{State: "idle"}
}

// Sync starts/stops relays so they match the config and the live session.
func (m *RelayManager) Sync() {
	sess := m.app.stream.CurrentSession()
	cfg := m.app.cfg.Get()

	desired := map[string]string{} // id -> target URL
	targetErr := map[string]string{}
	if sess != nil && !sess.BandwidthTest {
		for _, d := range cfg.Destinations {
			if !d.Enabled {
				continue
			}
			t, err := targetURL(d)
			if err != nil {
				targetErr[d.ID] = err.Error()
				continue
			}
			desired[d.ID] = t
		}
		if cfg.ForwardToTwitch && sess.TwitchKey != "" {
			desired[twitchForwardID] = "rtmp://live.twitch.tv/app/" + sess.TwitchKey
		}
	}

	m.mu.Lock()
	var toStart []string
	for id, run := range m.runs {
		t, ok := desired[id]
		if !ok || t != run.target || sess == nil || run.sessID != sess.ID {
			run.cancel()
			delete(m.runs, id)
			if st := m.status[id]; st != nil {
				st.State, st.Error = "idle", ""
			}
		}
	}
	for id := range desired {
		if _, ok := m.runs[id]; !ok {
			toStart = append(toStart, id)
		}
	}
	// Clear stale statuses of destinations that are not running.
	for id, st := range m.status {
		if _, running := m.runs[id]; !running {
			if _, wanted := desired[id]; !wanted {
				st.State, st.Error = "idle", ""
			}
		}
	}
	for id, e := range targetErr {
		m.status[id] = &RelayStatus{State: "error", Error: e, Since: time.Now()}
	}
	for _, id := range toStart {
		ctx, cancel := context.WithCancel(sess.ctx)
		run := &relayRun{target: desired[id], cancel: cancel, sessID: sess.ID}
		m.runs[id] = run
		go m.run(ctx, id, run.target, sess)
	}
	m.mu.Unlock()
}

func (m *RelayManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, run := range m.runs {
		run.cancel()
		delete(m.runs, id)
	}
}

func redact(target string) string {
	if i := strings.LastIndex(target, "/"); i > 0 {
		return target[:i] + "/****"
	}
	return target
}

// checkLoop prevents relaying to ourselves (when this PC uses StreamBridge as
// its own DNS, twitch/kick hostnames would resolve back to this computer).
func checkLoop(target string, ownPort int) error {
	u, err := rtmp.ParseURL(target)
	if err != nil {
		return err
	}
	ips, err := net.LookupIP(u.Hostname())
	if err != nil {
		return fmt.Errorf("cannot find server %s (check internet connection)", u.Hostname())
	}
	if u.Port() != fmt.Sprint(ownPort) {
		return nil
	}
	for _, ip := range ips {
		if isLocalAddress(ip) {
			return fmt.Errorf("loop: %s points to this computer. Set this computer's DNS back to automatic", u.Hostname())
		}
	}
	return nil
}

// dialRTMPViaProxy opens an RTMP(S) publish connection through the tunnel proxy.
func dialRTMPViaProxy(target, proxy string) (*rtmp.Conn, error) {
	u, err := rtmp.ParseURL(target)
	if err != nil {
		return nil, err
	}
	nc, err := dialProxy(proxy, u.Host, 12*time.Second)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "rtmps" {
		tc := tls.Client(nc, &tls.Config{ServerName: u.Hostname()})
		_ = tc.SetDeadline(time.Now().Add(12 * time.Second))
		if err := tc.Handshake(); err != nil {
			nc.Close()
			return nil, err
		}
		_ = tc.SetDeadline(time.Time{})
		nc = tc
	}
	conn := rtmp.NewConn(nc)
	conn.URL = u
	return conn, nil
}

func (m *RelayManager) run(ctx context.Context, id, target string, sess *Session) {
	backoff := 2 * time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		m.setStatus(id, "connecting", "")
		var conn *rtmp.Conn
		var err error
		if proxy, ok := m.app.relayViaTunnel(); ok {
			conn, err = dialRTMPViaProxy(target, proxy)
		} else {
			err = checkLoop(target, m.app.rtmpPort)
			if err == nil {
				conn, err = rtmp.DialTimeout(target, 10*time.Second, rtmp.DialOptions{SetupTimeout: 15 * time.Second})
			}
		}
		if err == nil {
			// Close the connection when the session or relay is cancelled.
			stop := make(chan struct{})
			go func() {
				select {
				case <-ctx.Done():
					conn.Close()
				case <-stop:
				}
			}()
			started := time.Now()
			cursor := sess.Queue.Latest()
			streams, serr := cursor.Streams()
			if serr == nil {
				err = conn.WriteHeader(streams)
			} else {
				err = serr
			}
			if err == nil {
				m.setStatus(id, "live", "")
				log.Printf("Relay: sending to %s", redact(target))
				err = avutil.CopyPackets(conn, cursor)
				if err == nil {
					_ = conn.WriteTrailer()
				}
			}
			close(stop)
			conn.Close()
			if time.Since(started) > 30*time.Second {
				backoff = 2 * time.Second
			}
		}
		if ctx.Err() != nil {
			m.setStatus(id, "idle", "")
			return
		}
		select {
		case <-sess.Done():
			m.setStatus(id, "idle", "")
			return
		default:
		}
		msg := "connection closed"
		if err != nil {
			msg = err.Error()
		}
		log.Printf("Relay: %s: %s (retrying in %s)", redact(target), msg, backoff)
		m.setStatus(id, "retrying", msg)
		select {
		case <-ctx.Done():
			m.setStatus(id, "idle", "")
			return
		case <-time.After(backoff):
		}
		if backoff < 20*time.Second {
			backoff *= 2
		}
	}
}
