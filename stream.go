package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/datarhei/joy4/av"
	"github.com/datarhei/joy4/av/avutil"
	"github.com/datarhei/joy4/av/pubsub"
	"github.com/datarhei/joy4/format/rtmp"
)

// Session is one broadcast from the console.
type Session struct {
	ID            int64
	Start         time.Time
	Publisher     *rtmp.Conn
	RemoteIP      string
	Path          string
	TwitchKey     string
	BandwidthTest bool
	Queue         *pubsub.Queue
	Streams       []av.CodecData
	Stats         *Stats
	Viewers       atomic.Int32
	ctx           context.Context
	cancel        context.CancelFunc
}

func (s *Session) Done() <-chan struct{} { return s.ctx.Done() }

type Stats struct {
	mu          sync.Mutex
	Width       int
	Height      int
	VideoCodec  string
	AudioCodec  string
	SampleRate  int
	FPS         float64
	Kbps        float64
	TotalBytes  int64
	winStart    time.Time
	winFrames   int
	winBytes    int64
	videoIdx    int8
	lastPacket  time.Time
}

func newStats(streams []av.CodecData) *Stats {
	st := &Stats{videoIdx: -1, winStart: time.Now()}
	for i, c := range streams {
		t := c.Type()
		if t.IsVideo() {
			st.videoIdx = int8(i)
			st.VideoCodec = codecName(t)
			if vc, ok := c.(av.VideoCodecData); ok {
				st.Width, st.Height = vc.Width(), vc.Height()
			}
		} else if t.IsAudio() {
			st.AudioCodec = codecName(t)
			if ac, ok := c.(av.AudioCodecData); ok {
				st.SampleRate = ac.SampleRate()
			}
		}
	}
	return st
}

func codecName(t av.CodecType) string {
	s := strings.ToUpper(t.String())
	switch s {
	case "H264":
		return "H.264"
	case "HEVC", "H265":
		return "HEVC"
	}
	return s
}

func (st *Stats) add(pkt av.Packet) {
	st.mu.Lock()
	defer st.mu.Unlock()
	now := time.Now()
	st.lastPacket = now
	n := int64(len(pkt.Data))
	st.TotalBytes += n
	st.winBytes += n
	if pkt.Idx == st.videoIdx {
		st.winFrames++
	}
	if el := now.Sub(st.winStart); el >= time.Second {
		secs := el.Seconds()
		st.FPS = float64(st.winFrames) / secs
		st.Kbps = float64(st.winBytes*8) / secs / 1000
		st.winFrames, st.winBytes, st.winStart = 0, 0, now
	}
}

type StreamStatus struct {
	Live          bool    `json:"live"`
	BandwidthTest bool    `json:"bandwidthTest"`
	Since         string  `json:"since,omitempty"`
	DurationSec   float64 `json:"durationSec"`
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	FPS           float64 `json:"fps"`
	Kbps          float64 `json:"kbps"`
	VideoCodec    string  `json:"videoCodec"`
	AudioCodec    string  `json:"audioCodec"`
	SampleRate    int     `json:"sampleRate"`
	From          string  `json:"from"`
	Viewers       int     `json:"viewers"`
	Stalled       bool    `json:"stalled"`
	HasTwitchKey  bool    `json:"hasTwitchKey"`
}

type LastStream struct {
	EndedAt     time.Time `json:"-"`
	EndedAgoSec float64   `json:"endedAgoSec"`
	DurationSec float64   `json:"durationSec"`
	From        string    `json:"from"`
}

type StreamServer struct {
	app  *App
	port int
	srv  *rtmp.Server

	mu      sync.Mutex
	running bool
	errMsg  string
	session *Session
	last    *LastStream
	nextID  int64
}

func NewStreamServer(app *App, port int) *StreamServer {
	return &StreamServer{app: app, port: port}
}

func (s *StreamServer) Start() {
	addr := fmt.Sprintf("0.0.0.0:%d", s.port)
	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		s.mu.Lock()
		s.errMsg = friendlyBindError(err)
		s.mu.Unlock()
		log.Printf("RTMP: could not listen on %s: %v", addr, err)
		return
	}
	s.srv = &rtmp.Server{
		HandlePublish:       s.handlePublish,
		HandlePlay:          s.handlePlay,
		SkipInvalidMessages: true,
	}
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()
	log.Printf("RTMP: listening on %s", addr)
	go func() {
		if err := s.srv.Serve(ln); err != nil && err != rtmp.ErrServerClosed {
			log.Printf("RTMP: server stopped: %v", err)
		}
	}()
}

func (s *StreamServer) Stop() {
	if s.srv != nil {
		s.srv.Close()
	}
}

func remoteIP(c *rtmp.Conn) string {
	if a, ok := c.RemoteAddr().(*net.TCPAddr); ok {
		return a.IP.String()
	}
	return c.RemoteAddr().String()
}

func parsePublishPath(path, rawQuery string) (key string, bwTest bool) {
	p := strings.TrimPrefix(path, "/")
	if i := strings.Index(p, "/"); i >= 0 {
		key = p[i+1:]
	}
	if i := strings.Index(key, "?"); i >= 0 {
		rawQuery += "&" + key[i+1:]
		key = key[:i]
	}
	bwTest = strings.Contains(strings.ToLower(rawQuery), "bandwidthtest=true")
	return
}

func (s *StreamServer) handlePublish(conn *rtmp.Conn) {
	defer conn.Close()
	from := remoteIP(conn)
	streams, err := conn.Streams()
	if err != nil {
		log.Printf("RTMP: publish from %s failed while reading stream info: %v", from, err)
		return
	}
	key, bw := parsePublishPath(conn.URL.Path, conn.URL.RawQuery)
	ctx, cancel := context.WithCancel(context.Background())
	sess := &Session{
		Start:         time.Now(),
		Publisher:     conn,
		RemoteIP:      from,
		Path:          conn.URL.Path,
		TwitchKey:     key,
		BandwidthTest: bw,
		Queue:         pubsub.NewQueue(),
		Streams:       streams,
		Stats:         newStats(streams),
		ctx:           ctx,
		cancel:        cancel,
	}
	_ = sess.Queue.WriteHeader(streams)

	s.mu.Lock()
	s.nextID++
	sess.ID = s.nextID
	old := s.session
	s.session = sess
	s.mu.Unlock()
	if old != nil {
		// A new broadcast replaces a stale one (e.g. console reconnected).
		old.cancel()
		old.Publisher.Close()
	}

	st := sess.Stats
	log.Printf("RTMP: broadcast started from %s (%dx%d %s / %s)%s", from, st.Width, st.Height, st.VideoCodec, st.AudioCodec, map[bool]string{true: " [bandwidth test]", false: ""}[bw])
	s.app.OnSessionStart(sess)

	for {
		pkt, err := conn.ReadPacket()
		if err != nil {
			break
		}
		sess.Stats.add(pkt)
		if err := sess.Queue.WritePacket(pkt); err != nil {
			break
		}
	}

	_ = sess.Queue.Close()
	sess.cancel()
	s.mu.Lock()
	if s.session == sess {
		s.session = nil
		s.last = &LastStream{EndedAt: time.Now(), DurationSec: time.Since(sess.Start).Seconds(), From: from}
	}
	s.mu.Unlock()
	log.Printf("RTMP: broadcast from %s ended after %s", from, time.Since(sess.Start).Round(time.Second))
	s.app.OnSessionEnd(sess)
}

// CurrentSession returns the live session or nil.
func (s *StreamServer) CurrentSession() *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.session
}

// WaitSession blocks until a session is live, ctx is done, or timeout passes.
func (s *StreamServer) WaitSession(ctx context.Context, timeout time.Duration) *Session {
	deadline := time.Now().Add(timeout)
	t := time.NewTicker(250 * time.Millisecond)
	defer t.Stop()
	for {
		if sess := s.CurrentSession(); sess != nil {
			return sess
		}
		if time.Now().After(deadline) {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

// handlePlay serves the live console stream to any player (OBS uses
// rtmp://127.0.0.1:1935/live/ps5, but any path works). If nothing is live
// yet, the connection waits so OBS shows the picture as soon as the console
// starts broadcasting.
func (s *StreamServer) handlePlay(conn *rtmp.Conn) {
	defer conn.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sess := s.WaitSession(ctx, 6*time.Hour)
	if sess == nil {
		return
	}
	sess.Viewers.Add(1)
	defer sess.Viewers.Add(-1)
	log.Printf("RTMP: player connected from %s", remoteIP(conn))
	_ = avutil.CopyFile(conn, sess.Queue.Latest())
}

func (s *StreamServer) Status() (StreamStatus, *LastStream, bool, string) {
	s.mu.Lock()
	sess := s.session
	last := s.last
	running, errMsg := s.running, s.errMsg
	s.mu.Unlock()

	var st StreamStatus
	if sess != nil {
		sess.Stats.mu.Lock()
		x := sess.Stats
		st = StreamStatus{
			Live:          true,
			BandwidthTest: sess.BandwidthTest,
			Since:         sess.Start.Format(time.RFC3339),
			DurationSec:   time.Since(sess.Start).Seconds(),
			Width:         x.Width,
			Height:        x.Height,
			FPS:           x.FPS,
			Kbps:          x.Kbps,
			VideoCodec:    x.VideoCodec,
			AudioCodec:    x.AudioCodec,
			SampleRate:    x.SampleRate,
			From:          sess.RemoteIP,
			Viewers:       int(sess.Viewers.Load()),
			Stalled:       time.Since(x.lastPacket) > 3*time.Second,
			HasTwitchKey:  sess.TwitchKey != "",
		}
		sess.Stats.mu.Unlock()
	}
	var lastCopy *LastStream
	if last != nil {
		l := *last
		l.EndedAgoSec = time.Since(l.EndedAt).Seconds()
		lastCopy = &l
	}
	return st, lastCopy, running, errMsg
}
