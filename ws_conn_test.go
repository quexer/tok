package tok_test

import (
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/quexer/tok"
)

var _ = Describe("WsConn", func() {
	var auth tok.WsAuthFunc
	BeforeEach(func() {
		auth = func(r *http.Request) (*tok.Device, error) {
			return tok.CreateDevice(fmt.Sprintf("%p", r), ""), nil
		}
	})

	It("CreateWsHandler", func() {
		hub, hdl := tok.CreateWsHandler(auth,
			tok.WithWsHandlerTxt(true),
			tok.WithWsHandlerHubConfig(tok.NewHubConfig(actor, 
				tok.WithHubConfigSso(true),
				tok.WithHubConfigPingProducer(&testPingGenerator{}))))
		Ω(hub).ToNot(BeNil())
		Ω(hdl).ToNot(BeNil())
	})
})
