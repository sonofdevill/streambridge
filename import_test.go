package main

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
)

// fakeStreamlabs mimics the Streamlabs Desktop Remote Control JSON-RPC API
// (method names and resource ids as in stream-labs/desktop external-api).
func fakeStreamlabs(t *testing.T, token string, existing bool) (int, *[]string, func()) {
	var mu sync.Mutex
	calls := []string{}
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/websocket", func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		authed := false
		for {
			_, data, err := c.ReadMessage()
			if err != nil {
				return
			}
			var req struct {
				ID     int    `json:"id"`
				Method string `json:"method"`
				Params struct {
					Resource string            `json:"resource"`
					Args     []json.RawMessage `json:"args"`
				} `json:"params"`
			}
			_ = json.Unmarshal(data, &req)
			mu.Lock()
			calls = append(calls, req.Method+"@"+req.Params.Resource)
			mu.Unlock()
			reply := func(result any) {
				b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
				_ = c.WriteMessage(websocket.TextMessage, b)
			}
			fail := func(msg string) {
				b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": map[string]any{"code": -32600, "message": msg}})
				_ = c.WriteMessage(websocket.TextMessage, b)
			}
			// an unrelated event, which the client must skip
			_ = c.WriteMessage(websocket.TextMessage, []byte(`{"jsonrpc":"2.0","result":{"_type":"EVENT"}}`))
			if req.Method == "auth" {
				var tok string
				_ = json.Unmarshal(req.Params.Args[0], &tok)
				if tok != token {
					fail("INVALID_TOKEN")
					continue
				}
				authed = true
				reply(true)
				continue
			}
			if !authed {
				fail("Authorization required")
				continue
			}
			switch req.Method {
			case "activeScene":
				items := []any{}
				reply(map[string]any{"_type": "HELPER", "resourceId": `Scene["scene_1"]`, "id": "scene_1", "name": "Main", "items": items})
			case "getSourcesByName":
				if existing {
					reply([]any{map[string]any{"resourceId": `Source["ffmpeg_source_9"]`, "sourceId": "ffmpeg_source_9"}})
				} else {
					reply([]any{})
				}
			case "createAndAddSource", "addSource":
				reply(map[string]any{"resourceId": `SceneItem["scene_1", "item_1", "ffmpeg_source_9"]`, "sourceId": "ffmpeg_source_9"})
			case "updateSettings", "fitToScreen":
				reply(nil)
			default:
				fail("unknown method " + req.Method)
			}
		}
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	return ln.Addr().(*net.TCPAddr).Port, &calls, func() { srv.Close() }
}

func TestStreamlabsCreate(t *testing.T) {
	port, calls, stop := fakeStreamlabs(t, "tok123", false)
	defer stop()
	scene, err := AddToStreamlabs(port, "tok123", "rtmp://127.0.0.1:1935/live/ps5")
	if err != nil || scene != "Main" {
		t.Fatalf("got %q %v", scene, err)
	}
	got := strings.Join(*calls, ",")
	want := `auth@TcpServerService,activeScene@ScenesService,getSourcesByName@SourcesService,createAndAddSource@Scene["scene_1"],fitToScreen@SceneItem["scene_1", "item_1", "ffmpeg_source_9"]`
	if got != want {
		t.Fatalf("calls:\n%s\nwant:\n%s", got, want)
	}
}

func TestStreamlabsExistingSource(t *testing.T) {
	port, calls, stop := fakeStreamlabs(t, "tok123", true)
	defer stop()
	if _, err := AddToStreamlabs(port, "tok123", "rtmp://x"); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(*calls, ",")
	if !strings.Contains(got, `updateSettings@Source["ffmpeg_source_9"]`) || !strings.Contains(got, "addSource@") || strings.Contains(got, "createAndAddSource") {
		t.Fatalf("unexpected calls: %s", got)
	}
}

func TestStreamlabsErrors(t *testing.T) {
	port, _, stop := fakeStreamlabs(t, "tok123", false)
	defer stop()
	_, err := AddToStreamlabs(port, "wrong", "rtmp://x")
	if ie, ok := err.(*ImportError); !ok || ie.Code != "bad_password" {
		t.Fatalf("want bad_password, got %v", err)
	}
	_, err = AddToStreamlabs(port, "", "rtmp://x")
	if ie, ok := err.(*ImportError); !ok || ie.Code != "need_password" {
		t.Fatalf("want need_password, got %v", err)
	}
	// nothing listening
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	free := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	_, err = AddToStreamlabs(free, "tok123", "rtmp://x")
	if ie, ok := err.(*ImportError); !ok || ie.Code != "not_running" {
		t.Fatalf("want not_running, got %v", err)
	}
}

func TestOBSAuthString(t *testing.T) {
	// Example from the obs-websocket v5 protocol documentation.
	got := obsAuthString("supersecretpassword", "lM1GncleQOaCu9lT1yeUZhFYnqhsLLP1G5lAGo3ixaI=", "+IxH4CnCiqpX1rM9scsNynZzbOe4KhDeYcTNS3PDaeY=")
	if got != "1Ct943GAT+6YQUUX47Ia/ncufilbe6+oD6lY+5kaCu4=" {
		t.Fatalf("auth string %s", got)
	}
}
