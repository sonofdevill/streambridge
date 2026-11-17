package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Destination is a platform the stream is relayed to directly (without OBS).
type Destination struct {
	ID       string `json:"id"`
	Platform string `json:"platform"` // kick, youtube, twitch, custom
	Name     string `json:"name"`
	URL      string `json:"url"`
	Key      string `json:"key"`
	Enabled  bool   `json:"enabled"`
}

// Config is persisted as JSON in the user's config directory.
type Config struct {
	Lang            string        `json:"lang"`
	PreferredIP     string        `json:"preferredIP"`
	ForwardToTwitch bool          `json:"forwardToTwitch"`
	Upstream        []string      `json:"upstream"`
	Destinations    []Destination `json:"destinations"`

	// Phone control
	RemoteEnabled bool   `json:"remoteEnabled"`
	RemotePIN     string `json:"remotePIN,omitempty"`
	RemoteSecret  string `json:"remoteSecret,omitempty"`

	// One-click import
	OBSPort     int    `json:"obsPort,omitempty"`
	OBSPassword string `json:"obsPassword,omitempty"`
	SLPort      int    `json:"streamlabsPort,omitempty"`
	SLToken     string `json:"streamlabsToken,omitempty"`

	// Iran mode: forward Twitch through a local V2ray proxy
	TunnelEnabled    bool     `json:"tunnelEnabled"`
	TunnelProxy      string   `json:"tunnelProxy,omitempty"`
	TunnelExtra      []string `json:"tunnelExtraDomains,omitempty"`
	TunnelSkipRelays bool     `json:"tunnelSkipRelays,omitempty"`
}

type ConfigStore struct {
	mu   sync.RWMutex
	path string
	cfg  Config
}

func defaultConfig() Config {
	return Config{
		Lang:     "",
		Upstream: []string{"1.1.1.1:53", "8.8.8.8:53"},
	}
}

func configPath(override string) string {
	if override != "" {
		return override
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		exe, _ := os.Executable()
		return filepath.Join(filepath.Dir(exe), "streambridge-config.json")
	}
	return filepath.Join(dir, "StreamBridge", "config.json")
}

func LoadConfig(override string) *ConfigStore {
	cs := &ConfigStore{path: configPath(override), cfg: defaultConfig()}
	if b, err := os.ReadFile(cs.path); err == nil {
		var c Config
		if json.Unmarshal(b, &c) == nil {
			if len(c.Upstream) == 0 {
				c.Upstream = defaultConfig().Upstream
			}
			cs.cfg = c
		}
	}
	return cs
}

func (cs *ConfigStore) Get() Config {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	c := cs.cfg
	c.Destinations = append([]Destination(nil), cs.cfg.Destinations...)
	c.Upstream = append([]string(nil), cs.cfg.Upstream...)
	return c
}

// Update applies fn to the config under lock and saves it to disk.
func (cs *ConfigStore) Update(fn func(c *Config)) error {
	cs.mu.Lock()
	fn(&cs.cfg)
	b, err := json.MarshalIndent(cs.cfg, "", "  ")
	path := cs.path
	cs.mu.Unlock()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

func newID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
