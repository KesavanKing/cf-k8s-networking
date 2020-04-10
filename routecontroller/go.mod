module code.cloudfoundry.org/cf-k8s-networking/routecontroller

go 1.13

require (
	github.com/go-logr/logr v0.1.0
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/sirupsen/logrus v1.4.2
	istio.io/api v0.0.0-20200409210158-852f8fa8e3f4 // indirect
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v0.0.0-20191014070654-bd505ee787b2
	k8s.io/cluster-registry v0.0.6
	sigs.k8s.io/controller-runtime v0.4.0
	sigs.k8s.io/kind v0.7.0
)
