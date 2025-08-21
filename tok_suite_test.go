package tok_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func TestTok(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tok Suite")
}

var ctx context.Context
var ctl *gomock.Controller
var _ = BeforeEach(func() {
	ctx = context.Background()
	ctl = gomock.NewController(GinkgoT())
})

var _ = AfterEach(func() {
	ctl.Finish()
})

// Test PingGenerator implementations
type testPingGenerator struct{}

func (p *testPingGenerator) Ping() []byte {
	return []byte("ping")
}
