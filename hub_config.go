package tok

import (
	"log"
	"time"
)

// HubConfig config struct for creating new Hub
type HubConfig struct {
	Actor              Actor         // Actor implement dispatch logic
	Q                  Queue         // Message Queue, if nil, message to offline user will not be cached
	Sso                bool          // Default true, if it's true, new connection  with same uid will kick off old ones
	ServerPingInterval time.Duration // Server ping interval, default 30 seconds
}

// NewHubConfig create new HubConfig
func NewHubConfig(actor Actor, opts ...HubConfigOption) *HubConfig {
	hc := &HubConfig{
		Actor:              actor,
		Q:                  NewMemoryQueue(), // default
		Sso:                true,             // default
		ServerPingInterval: 30 * time.Second, // default
	}

	for _, opt := range opts {
		opt(hc)
	}

	if hc.Actor == nil {
		log.Fatal("actor is needed")
	}

	return hc
}

type HubConfigOption func(*HubConfig)

// WithHubConfigQueue set queue for hub config. default is MemoryQueue
func WithHubConfigQueue(q Queue) HubConfigOption {
	return func(hc *HubConfig) {
		hc.Q = q
	}
}

// WithHubConfigSso set sso for hub config. default is true
func WithHubConfigSso(sso bool) HubConfigOption {
	return func(hc *HubConfig) {
		hc.Sso = sso
	}
}

// WithHubConfigServerPingInterval set ping interval for hub config, default is 30 seconds.
func WithHubConfigServerPingInterval(interval time.Duration) HubConfigOption {
	return func(hc *HubConfig) {
		hc.ServerPingInterval = interval
	}
}
