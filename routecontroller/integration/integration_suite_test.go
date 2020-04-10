package integration_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var (
	routeControllerBinaryPath string
)

var _ = SynchronizedBeforeSuite(func() []byte {
	binPath, err := gexec.Build(
		// TODO: Fix this compilation
		"code.cloudfoundry.org/cf-k8s-networking/routecontroller",
		"--race",
		// "--ldflags",
		// "-s -X github.com/pivotal/ingress-router/pkg/version.IngressRouterVersion=0.5.0",
	)
	Expect(err).NotTo(HaveOccurred())
	return []byte(binPath)
}, func(data []byte) {
	routeControllerBinaryPath = string(data)
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})
