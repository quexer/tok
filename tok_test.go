package tok_test

import (
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
	
})