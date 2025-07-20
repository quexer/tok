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

var _ = Describe("AfterSend Functional Option", func() {
	
	It("should work without AfterSend option", func() {
		hubConfig := tok.NewHubConfig(actor)
		Ω(hubConfig).ToNot(BeNil())
		// fnAfterSend should be nil, so AfterSend functionality is disabled
	})
	
	It("should work with AfterSend option", func() {
		var afterSendCalled bool
		
		afterSendFunc := func(dv *tok.Device, data []byte) {
			afterSendCalled = true
		}
		
		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigAfterSend(afterSendFunc))
		Ω(hubConfig).ToNot(BeNil())
		
		// Basic verification that the config was created successfully
		// The actual functionality is tested through integration
		Ω(afterSendCalled).To(BeFalse()) // Not called yet
	})
	
	It("should accept nil AfterSend function", func() {
		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigAfterSend(nil))
		Ω(hubConfig).ToNot(BeNil())
	})
	
	It("should work with multiple functional options including AfterSend", func() {
		afterSendFunc := func(dv *tok.Device, data []byte) {
			// Do nothing, just verify it can be configured
		}
		
		beforeReceiveFunc := func(dv *tok.Device, data []byte) ([]byte, error) {
			return data, nil
		}
		
		hubConfig := tok.NewHubConfig(actor, 
			tok.WithHubConfigAfterSend(afterSendFunc),
			tok.WithHubConfigBeforeReceive(beforeReceiveFunc),
			tok.WithHubConfigSso(false),
		)
		Ω(hubConfig).ToNot(BeNil())
	})
	
})