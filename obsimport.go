package main

// One-click import of the console stream into OBS Studio (via the built-in
// obs-websocket v5 server) and Streamlabs Desktop (via its Remote Control API).

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

const mediaSourceName = "PS5 (StreamBridge)"

// mediaSourceSettings are the OBS "ffmpeg_source" (Media Source) settings for
// the live console stream: network input, quick reconnect, small buffer for
// low latency, hardware decoding.
func mediaSourceSettings(url string) map[string]any {
	return map[string]any{
		"is_local_file":       false,
		"input":               url,
		"input_format":        "",
		"reconnect_delay_sec": 1,
		"buffering_mb":        1,
		"hw_decode":           true,
		"restart_on_activate": false,
		"close_when_inactive": false,
		"clear_on_media_end":  true,
	}
}

// ImportError carries a code the control panel can translate.
type ImportError struct {
	Code   string // not_running, need_password, bad_password, failed
	Detail string
}

func (e *ImportError) Error() string {
	if e.Detail != "" {
		return e.Code + ": " + e.Detail
	}
	return e.Code
}

func isConnRefused(err error) bool {
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}
	var op *net.OpError
	if errors.As(err, &op) {
		s := strings.ToLower(op.Error())
		return strings.Contains(s, "refused") || strings.Contains(s, "actively refused")
	}
	return strings.Contains(strings.ToLower(err.Error()), "refused")
}

// ---------------------------------------------------------------- OBS Studio

type obsClient struct {
	ws     *websocket.Conn
	nextID int
}

type obsEnvelope struct {
	Op int             `json:"op"`
	D  json.RawMessage `json:"d"`
}

func obsAuthString(password, salt, challenge string) string {
	h := sha256.Sum256([]byte(password + salt))
	secret := base64.StdEncoding.EncodeToString(h[:])
	h2 := sha256.Sum256([]byte(secret + challenge))
	return base64.StdEncoding.EncodeToString(h2[:])
}

func obsConnect(port int, password string) (*obsClient, error) {
	d := websocket.Dialer{HandshakeTimeout: 4 * time.Second, Subprotocols: []string{"obswebsocket.json"}}
	// OBS may listen on IPv4 or IPv6 only, depending on the system.
	var ws *websocket.Conn
	var err, firstErr error
	unreachable := true // every attempt was refused or the address family is missing
	for _, host := range []string{"127.0.0.1", "[::1]"} {
		ws, _, err = d.Dial(fmt.Sprintf("ws://%s:%d", host, port), nil)
		if err == nil {
			break
		}
		if firstErr == nil {
			firstErr = err
		}
		msg := strings.ToLower(err.Error())
		if !isConnRefused(err) && !strings.Contains(msg, "address family") && !strings.Contains(msg, "no route") && !strings.Contains(msg, "unreachable") {
			unreachable = false
		}
	}
	if err != nil {
		if unreachable {
			return nil, &ImportError{Code: "not_running"}
		}
		return nil, &ImportError{Code: "failed", Detail: firstErr.Error()}
	}
	c := &obsClient{ws: ws}
	_ = ws.SetReadDeadline(time.Now().Add(6 * time.Second))

	var hello struct {
		RPCVersion     int `json:"rpcVersion"`
		Authentication *struct {
			Challenge string `json:"challenge"`
			Salt      string `json:"salt"`
		} `json:"authentication"`
	}
	if err := c.readOp(0, &hello); err != nil {
		ws.Close()
		return nil, &ImportError{Code: "failed", Detail: "no hello from OBS: " + err.Error()}
	}
	identify := map[string]any{"rpcVersion": 1, "eventSubscriptions": 0}
	if hello.Authentication != nil {
		if password == "" {
			ws.Close()
			return nil, &ImportError{Code: "need_password"}
		}
		identify["authentication"] = obsAuthString(password, hello.Authentication.Salt, hello.Authentication.Challenge)
	}
	if err := ws.WriteJSON(map[string]any{"op": 1, "d": identify}); err != nil {
		ws.Close()
		return nil, &ImportError{Code: "failed", Detail: err.Error()}
	}
	if err := c.readOp(2, nil); err != nil {
		ws.Close()
		var ce *websocket.CloseError
		if errors.As(err, &ce) && ce.Code == 4009 {
			return nil, &ImportError{Code: "bad_password"}
		}
		return nil, &ImportError{Code: "failed", Detail: err.Error()}
	}
	return c, nil
}

func (c *obsClient) readOp(op int, out any) error {
	for {
		var env obsEnvelope
		if err := c.ws.ReadJSON(&env); err != nil {
			return err
		}
		if env.Op == op {
			if out != nil {
				return json.Unmarshal(env.D, out)
			}
			return nil
		}
	}
}

type obsStatus struct {
	Result  bool   `json:"result"`
	Code    int    `json:"code"`
	Comment string `json:"comment"`
}

func (c *obsClient) request(typ string, data map[string]any, out any) (obsStatus, error) {
	c.nextID++
	id := fmt.Sprintf("sb-%d", c.nextID)
	msg := map[string]any{"op": 6, "d": map[string]any{"requestType": typ, "requestId": id, "requestData": data}}
	_ = c.ws.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := c.ws.WriteJSON(msg); err != nil {
		return obsStatus{}, err
	}
	_ = c.ws.SetReadDeadline(time.Now().Add(8 * time.Second))
	for {
		var resp struct {
			RequestID     string          `json:"requestId"`
			RequestStatus obsStatus       `json:"requestStatus"`
			ResponseData  json.RawMessage `json:"responseData"`
		}
		if err := c.readOp(7, &resp); err != nil {
			return obsStatus{}, err
		}
		if resp.RequestID != id {
			continue
		}
		if out != nil && len(resp.ResponseData) > 0 {
			_ = json.Unmarshal(resp.ResponseData, out)
		}
		return resp.RequestStatus, nil
	}
}

func (c *obsClient) must(typ string, data map[string]any, out any) error {
	st, err := c.request(typ, data, out)
	if err != nil {
		return &ImportError{Code: "failed", Detail: typ + ": " + err.Error()}
	}
	if !st.Result {
		return &ImportError{Code: "failed", Detail: fmt.Sprintf("%s: %s (code %d)", typ, st.Comment, st.Code)}
	}
	return nil
}

// AddToOBS creates (or updates) the "PS5 (StreamBridge)" Media Source in the
// current scene, fits it to the canvas and puts it at the bottom, so overlays
// stay on top. It returns the scene name.
func AddToOBS(port int, password, url string) (string, error) {
	c, err := obsConnect(port, password)
	if err != nil {
		return "", err
	}
	defer c.ws.Close()

	var scene struct {
		CurrentProgramSceneName string `json:"currentProgramSceneName"`
	}
	if err := c.must("GetCurrentProgramScene", nil, &scene); err != nil {
		return "", err
	}
	sceneName := scene.CurrentProgramSceneName
	settings := mediaSourceSettings(url)

	var itemID int
	st, err := c.request("GetInputSettings", map[string]any{"inputName": mediaSourceName}, nil)
	if err != nil {
		return "", &ImportError{Code: "failed", Detail: err.Error()}
	}
	if st.Result {
		// Source exists already: refresh its settings and make sure it is in this scene.
		if err := c.must("SetInputSettings", map[string]any{"inputName": mediaSourceName, "inputSettings": settings, "overlay": true}, nil); err != nil {
			return "", err
		}
		var item struct {
			SceneItemID int `json:"sceneItemId"`
		}
		st, err := c.request("GetSceneItemId", map[string]any{"sceneName": sceneName, "sourceName": mediaSourceName}, &item)
		if err != nil {
			return "", &ImportError{Code: "failed", Detail: err.Error()}
		}
		if st.Result {
			itemID = item.SceneItemID
		} else if err := c.must("CreateSceneItem", map[string]any{"sceneName": sceneName, "sourceName": mediaSourceName, "sceneItemEnabled": true}, &item); err != nil {
			return "", err
		} else {
			itemID = item.SceneItemID
		}
	} else {
		var created struct {
			SceneItemID int `json:"sceneItemId"`
		}
		if err := c.must("CreateInput", map[string]any{
			"sceneName": sceneName, "inputName": mediaSourceName, "inputKind": "ffmpeg_source",
			"inputSettings": settings, "sceneItemEnabled": true,
		}, &created); err != nil {
			return "", err
		}
		itemID = created.SceneItemID
	}

	var video struct {
		BaseWidth  float64 `json:"baseWidth"`
		BaseHeight float64 `json:"baseHeight"`
	}
	if err := c.must("GetVideoSettings", nil, &video); err == nil && video.BaseWidth > 0 {
		_ = c.must("SetSceneItemTransform", map[string]any{
			"sceneName": sceneName, "sceneItemId": itemID,
			"sceneItemTransform": map[string]any{
				"positionX": 0, "positionY": 0, "alignment": 5, // top-left
				"boundsType": "OBS_BOUNDS_SCALE_INNER", "boundsAlignment": 0,
				"boundsWidth": video.BaseWidth, "boundsHeight": video.BaseHeight,
			},
		}, nil)
	}
	_ = c.must("SetSceneItemIndex", map[string]any{"sceneName": sceneName, "sceneItemId": itemID, "sceneItemIndex": 0}, nil)
	return sceneName, nil
}

// --------------------------------------------------------- Streamlabs Desktop

type slClient struct {
	ws     *websocket.Conn
	nextID int
}

type slResponse struct {
	ID     *int            `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *slClient) call(method, resource string, args ...any) (json.RawMessage, error) {
	c.nextID++
	id := c.nextID
	params := map[string]any{"resource": resource}
	if len(args) > 0 {
		params["args"] = args
	}
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	_ = c.ws.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := c.ws.WriteMessage(websocket.TextMessage, b); err != nil {
		return nil, err
	}
	_ = c.ws.SetReadDeadline(time.Now().Add(8 * time.Second))
	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var r slResponse
			if json.Unmarshal([]byte(line), &r) != nil || r.ID == nil || *r.ID != id {
				continue // events and other messages
			}
			if r.Error != nil {
				return nil, fmt.Errorf("%s", r.Error.Message)
			}
			return r.Result, nil
		}
	}
}

type slItem struct {
	ResourceID string `json:"resourceId"`
	SourceID   string `json:"sourceId"`
}

// AddToStreamlabs creates (or updates) the Media Source in the active scene of
// Streamlabs Desktop and fits it to the screen.
func AddToStreamlabs(port int, token, url string) (string, error) {
	if strings.TrimSpace(token) == "" {
		return "", &ImportError{Code: "need_password"}
	}
	d := websocket.Dialer{HandshakeTimeout: 4 * time.Second}
	ws, _, err := d.Dial(fmt.Sprintf("ws://127.0.0.1:%d/api/websocket", port), nil)
	if err != nil {
		if isConnRefused(err) {
			return "", &ImportError{Code: "not_running"}
		}
		return "", &ImportError{Code: "failed", Detail: err.Error()}
	}
	defer ws.Close()
	c := &slClient{ws: ws}

	if _, err := c.call("auth", "TcpServerService", strings.TrimSpace(token)); err != nil {
		return "", &ImportError{Code: "bad_password", Detail: err.Error()}
	}
	raw, err := c.call("activeScene", "ScenesService")
	if err != nil {
		return "", &ImportError{Code: "failed", Detail: "activeScene: " + err.Error()}
	}
	var scene struct {
		ResourceID string   `json:"resourceId"`
		Name       string   `json:"name"`
		Items      []slItem `json:"items"`
	}
	if err := json.Unmarshal(raw, &scene); err != nil || scene.ResourceID == "" {
		return "", &ImportError{Code: "failed", Detail: "unexpected scene data"}
	}
	settings := mediaSourceSettings(url)

	var item slItem
	raw, err = c.call("getSourcesByName", "SourcesService", mediaSourceName)
	var existing []slItem
	if err == nil {
		_ = json.Unmarshal(raw, &existing)
	}
	if len(existing) > 0 {
		src := existing[0]
		if _, err := c.call("updateSettings", src.ResourceID, settings); err != nil {
			return "", &ImportError{Code: "failed", Detail: "updateSettings: " + err.Error()}
		}
		for _, it := range scene.Items {
			if it.SourceID == src.SourceID {
				item = it
			}
		}
		if item.ResourceID == "" {
			raw, err := c.call("addSource", scene.ResourceID, src.SourceID)
			if err != nil {
				return "", &ImportError{Code: "failed", Detail: "addSource: " + err.Error()}
			}
			_ = json.Unmarshal(raw, &item)
		}
	} else {
		raw, err := c.call("createAndAddSource", scene.ResourceID, mediaSourceName, "ffmpeg_source", settings)
		if err != nil {
			return "", &ImportError{Code: "failed", Detail: "createAndAddSource: " + err.Error()}
		}
		_ = json.Unmarshal(raw, &item)
	}
	if item.ResourceID != "" {
		_, _ = c.call("fitToScreen", item.ResourceID)
	}
	return scene.Name, nil
}
