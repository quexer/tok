package tok

import (
	"log"
	"time"
)

// BeforeReceiveFunc is a function type for preprocessing incoming data before OnReceive
type BeforeReceiveFunc func(dv *Device, data []byte) ([]byte, error)

// HubConfig config struct for creating new Hub
type HubConfig struct {
	actor              Actor                // actor implement dispatch logic
	beforeReceive      BeforeReceiveFunc    // optional preprocessing function for incoming data
	q                  Queue                // Message Queue, default is memory-based queue. if nil, message to offline user will not be cached
	sso                bool                 // Default true, if it's true, new connection  with same uid will kick off old ones
	serverPingInterval time.Duration        // Server ping interval, default 30 seconds
	authTimeout        time.Duration        // Auth timeout duration, default 5s
	writeTimeout       time.Duration        // Write timeout duration, default 1m
	readTimeout        time.Duration        // Read timeout duration, default 0s, means no read timeout
}

// NewHubConfig create new HubConfig
func NewHubConfig(actor Actor, opts ...HubConfigOption) *HubConfig {
	if actor == nil {
		log.Fatal("actor is needed")
	}

	hc := &HubConfig{
		actor:              actor,
		q:                  NewMemoryQueue(), // default
		sso:                true,             // default
		serverPingInterval: 30 * time.Second, // default
		authTimeout:        5 * time.Second,  // default
		writeTimeout:       time.Minute,      // default
		readTimeout:        0,
	}

	for _, opt := range opts {
		opt(hc)
	}

	return hc
}

type HubConfigOption func(*HubConfig)

// WithHubConfigQueue set queue for hub config. default is MemoryQueue
func WithHubConfigQueue(q Queue) HubConfigOption {
	return func(hc *HubConfig) {
		hc.q = q
	}
}

// WithHubConfigSso set sso for hub config. default is true
func WithHubConfigSso(sso bool) HubConfigOption {
	return func(hc *HubConfig) {
		hc.sso = sso
	}
}

// WithHubConfigServerPingInterval set ping interval for hub config, default is 30 seconds.
func WithHubConfigServerPingInterval(interval time.Duration) HubConfigOption {
	return func(hc *HubConfig) {
		hc.serverPingInterval = interval
	}
}

// WithHubConfigAuthTimeout set auth timeout for hub config, default is 5 seconds.
func WithHubConfigAuthTimeout(timeout time.Duration) HubConfigOption {
	return func(hc *HubConfig) {
		hc.authTimeout = timeout
	}
}

// WithHubConfigWriteTimeout set write timeout for hub config, default is 1 minute.
func WithHubConfigWriteTimeout(timeout time.Duration) HubConfigOption {
	return func(hc *HubConfig) {
		hc.writeTimeout = timeout
	}
}

// WithHubConfigReadTimeout set read timeout for hub config, default is 0 seconds, means no read timeout.
func WithHubConfigReadTimeout(timeout time.Duration) HubConfigOption {
	return func(hc *HubConfig) {
		hc.readTimeout = timeout
	}
}

// WithHubConfigBeforeReceive set optional BeforeReceive function for hub config.
func WithHubConfigBeforeReceive(beforeReceive BeforeReceiveFunc) HubConfigOption {
	return func(hc *HubConfig) {
		hc.beforeReceive = beforeReceive
	}
}
