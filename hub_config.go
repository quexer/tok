package tok

import (
	"errors"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// PartialSendPolicy controls how Send with ttl > 0 behaves when sending to multiple online connections partially fails.
type PartialSendPolicy int

const (
	// PartialSendCacheOnAnyFailure caches the message if any online connection send failed.
	// This is the default behavior for backward compatibility.
	PartialSendCacheOnAnyFailure PartialSendPolicy = iota
	// PartialSendCacheWhenAllFailed caches the message only when all online connections failed.
	PartialSendCacheWhenAllFailed
	// PartialSendNoCache never caches online send failures; partial/all failures are returned directly.
	PartialSendNoCache
)

// HubConfig config struct for creating new Hub
type HubConfig struct {
	actor              Actor                // actor implement dispatch logic
	pingProducer       PingGenerator        // optional server-side ping generator for auto-ping feature
	byeGenerator       ByeGenerator         // optional bye generator for connection close notifications
	hdlBeforeReceive   BeforeReceiveHandler // optional preprocessing handler for incoming data
	hdlBeforeSend      BeforeSendHandler    // optional preprocessing handler for outgoing data
	hdlAfterSend       AfterSendHandler     // optional AfterSend handler
	closeHandler       CloseHandler         // optional CloseHandler for connection close events
	q                  Queue                // Message Queue, default is memory-based queue. if nil, message to offline user will not be cached
	sso                bool                 // Default true, if it's true, new connection  with same uid will kick off old ones
	serverPingInterval time.Duration        // Server ping interval, default 30 seconds
	authTimeout        time.Duration        // Auth timeout duration, default 5s
	writeTimeout       time.Duration        // Write timeout duration, default 1m
	readTimeout        time.Duration        // Read timeout duration, default 0s, means no read timeout
	partialSendPolicy  PartialSendPolicy    // strategy for online partial send failures when ttl > 0
	meterProvider      metric.MeterProvider // optional OTel MeterProvider, defaults to global
	tracerProvider     trace.TracerProvider // optional OTel TracerProvider, defaults to global
}

// NewHubConfig create new HubConfig
func NewHubConfig(actor Actor, opts ...HubConfigOption) (*HubConfig, error) {
	if actor == nil {
		return nil, errors.New("[tok] actor is required")
	}

	hc := &HubConfig{
		actor:              actor,
		q:                  NewMemoryQueue(), // default
		sso:                true,             // default
		serverPingInterval: 30 * time.Second, // default
		authTimeout:        5 * time.Second,  // default
		writeTimeout:       time.Minute,      // default
		readTimeout:        0,
		partialSendPolicy:  PartialSendCacheOnAnyFailure,
	}

	for _, opt := range opts {
		opt(hc)
	}

	return hc, nil
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

// WithHubConfigBeforeReceive set optional BeforeReceive handler for hub config.
func WithHubConfigBeforeReceive(hdl BeforeReceiveHandler) HubConfigOption {
	return func(hc *HubConfig) {
		hc.hdlBeforeReceive = hdl
	}
}

// WithHubConfigBeforeSend set optional BeforeSend handler for hub config.
func WithHubConfigBeforeSend(beforeSend BeforeSendHandler) HubConfigOption {
	return func(hc *HubConfig) {
		hc.hdlBeforeSend = beforeSend
	}
}

// WithHubConfigAfterSend set optional AfterSend handler for hub config.
func WithHubConfigAfterSend(afterSend AfterSendHandler) HubConfigOption {
	return func(hc *HubConfig) {
		hc.hdlAfterSend = afterSend
	}
}

// WithHubConfigCloseHandler set optional CloseHandler for hub config.
func WithHubConfigCloseHandler(closeHandler CloseHandler) HubConfigOption {
	return func(hc *HubConfig) {
		hc.closeHandler = closeHandler
	}
}

// WithHubConfigPingProducer set optional PingGenerator for hub config to enable auto-ping feature.
// if this is not set, server-side auto-ping feature is disabled.
func WithHubConfigPingProducer(pingProducer PingGenerator) HubConfigOption {
	return func(hc *HubConfig) {
		hc.pingProducer = pingProducer
	}
}

// WithHubConfigByeGenerator set optional ByeGenerator for hub config to enable bye message generation.
// if this is not set, no bye messages will be sent when closing connections.
func WithHubConfigByeGenerator(byeGenerator ByeGenerator) HubConfigOption {
	return func(hc *HubConfig) {
		hc.byeGenerator = byeGenerator
	}
}

// WithHubConfigPartialSendPolicy configures how Send with ttl > 0 behaves on online partial failures.
func WithHubConfigPartialSendPolicy(policy PartialSendPolicy) HubConfigOption {
	return func(hc *HubConfig) {
		hc.partialSendPolicy = policy
	}
}

// WithMeterProvider sets the OTel MeterProvider for hub metrics.
// If not set, the global MeterProvider is used (noop by default).
func WithMeterProvider(mp metric.MeterProvider) HubConfigOption {
	return func(hc *HubConfig) {
		hc.meterProvider = mp
	}
}

// WithTracerProvider sets the OTel TracerProvider for hub tracing.
// If not set, the global TracerProvider is used (noop by default).
func WithTracerProvider(tp trace.TracerProvider) HubConfigOption {
	return func(hc *HubConfig) {
		hc.tracerProvider = tp
	}
}
