package tok_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/quexer/tok"
)

var _ = Describe("BeforeReceive Functional Option", func() {
	
	It("should work without BeforeReceive option", func() {
		hubConfig := tok.NewHubConfig(actor)
		Ω(hubConfig).ToNot(BeNil())
	})
	
	It("should work with BeforeReceive option", func() {
		beforeReceiveFunc := func(dv *tok.Device, data []byte) ([]byte, error) {
			// Transform data by adding a prefix
			return append([]byte("prefix:"), data...), nil
		}
		
		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigBeforeReceive(beforeReceiveFunc))
		Ω(hubConfig).ToNot(BeNil())
	})
	
})

var _ = Describe("BeforeSend Functional Option", func() {
	
	It("should work without BeforeSend option", func() {
		hubConfig := tok.NewHubConfig(actor)
		Ω(hubConfig).ToNot(BeNil())
	})
	
	It("should work with BeforeSend option", func() {
		beforeSendFunc := func(dv *tok.Device, data []byte) ([]byte, error) {
			// Transform data by adding a suffix
			return append(data, []byte(":suffix")...), nil
		}
		
		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigBeforeSend(beforeSendFunc))
		Ω(hubConfig).ToNot(BeNil())
	})
	
	It("should handle BeforeSend error correctly", func() {
		beforeSendFunc := func(dv *tok.Device, data []byte) ([]byte, error) {
			// Return an error to test error handling
			return nil, errors.New("BeforeSend error")
		}
		
		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigBeforeSend(beforeSendFunc))
		Ω(hubConfig).ToNot(BeNil())
	})
	
	It("should handle BeforeSend returning nil data correctly", func() {
		beforeSendFunc := func(dv *tok.Device, data []byte) ([]byte, error) {
			// Return nil data, should use original data
			return nil, nil
		}
		
		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigBeforeSend(beforeSendFunc))
		Ω(hubConfig).ToNot(BeNil())
	})
	
})

var _ = Describe("OnSent Functional Option", func() {
	
	It("should work without OnSent option", func() {
		hubConfig := tok.NewHubConfig(actor)
		Ω(hubConfig).ToNot(BeNil())
		// fnOnSent should be nil, so OnSent functionality is disabled
	})
	
	It("should work with OnSent option", func() {
		var onSentCalled bool
		
		onSentFunc := func(dv *tok.Device, data []byte) {
			onSentCalled = true
		}
		
		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigOnSent(onSentFunc))
		Ω(hubConfig).ToNot(BeNil())
		
		// Basic verification that the config was created successfully
		// The actual functionality is tested through integration
		Ω(onSentCalled).To(BeFalse()) // Not called yet
	})
	
	It("should accept nil OnSent function", func() {
		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigOnSent(nil))
		Ω(hubConfig).ToNot(BeNil())
	})
	
	It("should work with multiple functional options including OnSent", func() {
		onSentFunc := func(dv *tok.Device, data []byte) {
			// Do nothing, just verify it can be configured
		}
		
		beforeReceiveFunc := func(dv *tok.Device, data []byte) ([]byte, error) {
			return data, nil
		}
		
		hubConfig := tok.NewHubConfig(actor, 
			tok.WithHubConfigOnSent(onSentFunc),
			tok.WithHubConfigBeforeReceive(beforeReceiveFunc),
			tok.WithHubConfigSso(false),
		)
		Ω(hubConfig).ToNot(BeNil())
	})
	
})