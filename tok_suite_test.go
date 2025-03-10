package tok_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTok(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tok Suite")
}

var ctx context.Context
var _ = BeforeEach(func() {
	ctx = context.Background()
})
