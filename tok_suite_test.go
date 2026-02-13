package tok_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func TestTok(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tok Suite")
}

var _ = BeforeSuite(func() {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
})

var ctx context.Context
var ctl *gomock.Controller
var _ = BeforeEach(func() {
	ctx = context.Background()
	ctl = gomock.NewController(GinkgoT())
})

var _ = AfterEach(func() {
	ctl.Finish()
})
