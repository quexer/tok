package tok

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
