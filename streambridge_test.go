package main

import (
	"net"
	"testing"
)

func TestHijackNames(t *testing.T) {
	yes := []string{
		"live.twitch.tv.", "live-sea.twitch.tv.", "LIVE-SJC.TWITCH.TV.",
		"ingest.global-contribute.live-video.net.", "sea02.contribute.live-video.net.",
		"scl01.contribute.live-video.net.",
	}
	no := []string{
		"ingest.twitch.tv.", "www.twitch.tv.", "api.twitch.tv.", "id.twitch.tv.", "twitch.tv.",
		"live-video.net.", "playstation.net.", "notlive.twitch.tv.", "live.twitch.tv.evil.com.",
	}
	for _, n := range yes {
		if !isHijackName(n) {
			t.Errorf("expected hijack for %s", n)
		}
	}
	for _, n := range no {
		if isHijackName(n) {
			t.Errorf("did not expect hijack for %s", n)
		}
	}
}

func TestParsePublishPath(t *testing.T) {
	k, bw := parsePublishPath("/app/live_123_abc", "")
	if k != "live_123_abc" || bw {
		t.Fatalf("got %q %v", k, bw)
	}
	k, bw = parsePublishPath("/app/live_123_abc", "bandwidthtest=true")
	if k != "live_123_abc" || !bw {
		t.Fatalf("got %q %v", k, bw)
	}
	k, bw = parsePublishPath("/app/live_123_abc?bandwidthtest=true", "")
	if k != "live_123_abc" || !bw {
		t.Fatalf("got %q %v", k, bw)
	}
}

func TestTargetURL(t *testing.T) {
	cases := map[Destination]string{
		{URL: "rtmps://fa723fc1b171.global-contribute.live-video.net", Key: "sk_x"}:         "rtmps://fa723fc1b171.global-contribute.live-video.net/app/sk_x",
		{URL: "rtmps://fa723fc1b171.global-contribute.live-video.net:443/app", Key: "sk_x"}: "rtmps://fa723fc1b171.global-contribute.live-video.net:443/app/sk_x",
		{URL: "rtmp://a.rtmp.youtube.com/live2/", Key: " abc "}:                             "rtmp://a.rtmp.youtube.com/live2/abc",
	}
	for d, want := range cases {
		got, err := targetURL(d)
		if err != nil || got != want {
			t.Errorf("targetURL(%v) = %q, %v; want %q", d, got, err, want)
		}
	}
	if _, err := targetURL(Destination{URL: "http://x", Key: "k"}); err == nil {
		t.Error("expected error for non-rtmp URL")
	}
	if _, err := targetURL(Destination{URL: "rtmp://x/app"}); err == nil {
		t.Error("expected error for missing key")
	}
}

func TestDeviceClassification(t *testing.T) {
	d := NewDNSServer(&App{cfg: LoadConfig("/nonexistent/sb.json")}, 0)
	ps := net.ParseIP("192.168.1.50")
	d.record(ps, "www.google.com.", 1, false, "")
	d.record(ps, "ps4-system.sec.np.dl.playstation.net.", 1, false, "")
	d.record(ps, "live-sea.twitch.tv.", 1, true, "192.168.1.20")
	devs := d.Devices()
	if len(devs) != 1 || devs[0].Kind != "playstation" || !devs[0].AskedTwitch || devs[0].Queries != 3 {
		t.Fatalf("unexpected devices: %+v", devs)
	}
	if l := d.Log(10); len(l) != 3 || !l[0].Hijacked {
		t.Fatalf("unexpected log: %+v", l)
	}
}
