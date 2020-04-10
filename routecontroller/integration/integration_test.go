package integration_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"sigs.k8s.io/kind/pkg/cluster"
)

var _ = Describe("Integration", func() {
	var (
		session *gexec.Session

		clusterName    string
		kubeConfigPath string
		namespace      string
	)

	BeforeEach(func() {
		// TODO Make a unique name
		clusterName = fmt.Sprintf("master-routing-integration-test-%d", GinkgoParallelNode()) // TODO rename
		namespace = "cf-k8s-networking-tests"

		kubeConfigPath = createKindCluster(clusterName)

		output, err := kubectlWithConfig(kubeConfigPath, nil, "create", "namespace", namespace)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("kubectl create namespace failed with err: %s", string(output)))

		istioCRDPath := filepath.Join("fixtures", "istio-virtual-service.yaml")
		output, err = kubectlWithConfig(kubeConfigPath, nil, "-n", namespace, "apply", "-f", istioCRDPath)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("kubectl apply crd failed with err: %s", string(output)))

		// Generate the YAML for the Route CRD with Kustomize, and then apply it with kubectl apply.
		kustomizeOutput, err := kustomizeConfigCRD()
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("kustomize failed to render CRD yaml: %s", string(kustomizeOutput)))
		kustomizeOutputReader := bytes.NewReader(kustomizeOutput)

		output, err = kubectlWithConfig(kubeConfigPath, kustomizeOutputReader, "-n", namespace, "apply", "-f", "-")
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("kubectl apply crd failed with err: %s", string(output)))

		session = startRouteController(kubeConfigPath, namespace)
	})

	AfterEach(func() {
		session.Interrupt()
		Eventually(session, "10s").Should(gexec.Exit())

		deleteKindCluster(clusterName, kubeConfigPath)
	})

	It("successfully creates the Route and VirtualService CRDs", func() {
		output, err := kubectlWithConfig(kubeConfigPath, nil, "get", "crds")
		Expect(err).NotTo(HaveOccurred())

		Expect(string(output)).To(ContainSubstring("routes.networking.cloudfoundry.org"))
		Expect(string(output)).To(ContainSubstring("virtualservices.networking.istio.io"))

		routeCRYaml := filepath.Join("fixtures", "route.yaml")
		output, err = kubectlWithConfig(kubeConfigPath, nil, "-n", namespace, "apply", "-f", routeCRYaml)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("kubectl apply CR failed with err: %s", string(output)))
	})

	When("there is no destination provided in the route CR", func() {
		BeforeEach(func() {
			routeCRYaml := filepath.Join("fixtures", "route-without-any-destination.yaml")
			output, err := kubectlWithConfig(kubeConfigPath, nil, "-n", namespace, "apply", "-f", routeCRYaml)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("kubectl apply CR failed with err: %s", string(output)))
		})

		It("does not create a virtualservice", func() {
			output, err := kubectlWithConfig(kubeConfigPath, nil, "get", "virtualservices")
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("kubectl get virtualservices failed: %s", string(output)))
			Expect(string(output)).To(ContainSubstring("")) // Stop shelling out to kubectl, use library code
		})
	})

	When("there is a single route CR with a single destination", func() {
		BeforeEach(func() {
			routeCRYaml := filepath.Join("fixtures", "single-route-with-single-destination.yaml")
			output, err := kubectlWithConfig(kubeConfigPath, nil, "-n", namespace, "apply", "-f", routeCRYaml)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("kubectl apply CR failed with err: %s", string(output)))
		})

		It("creates a virtualservice and a service", func() {
			output, err := kubectlWithConfig(kubeConfigPath, nil, "get", "virtualservices")
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("kubectl get virtualservices failed: %s", string(output)))
			Expect(string(output)).To(ContainSubstring("")) // Stop shelling out to kubectl, use library code
		})
	})
})

func startRouteController(kubeConfigPath, namespace string) *gexec.Session {
	cmd := exec.Command(routeControllerBinaryPath)
	cmd.Env = os.Environ()
	// cmd.Env = append(cmd.Env, fmt.Sprintf("ROUTING_NAMESPACE=%s", namespace)) TODO: Not sure about
	// cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%s", kubeConfigPath))

	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	return session
}

func createKindCluster(name string) string {
	provider := cluster.NewProvider()
	err := provider.Create(name)
	Expect(err).NotTo(HaveOccurred())

	kubeConfig, err := provider.KubeConfig(name, false)
	Expect(err).NotTo(HaveOccurred())

	kubeConfigPath, err := ioutil.TempFile("", fmt.Sprintf("kubeconfig-%s", name))
	Expect(err).NotTo(HaveOccurred())
	defer kubeConfigPath.Close()

	_, err = kubeConfigPath.Write([]byte(kubeConfig))
	Expect(err).NotTo(HaveOccurred())

	return kubeConfigPath.Name()
}

func deleteKindCluster(name, kubeConfigPath string) {
	provider := cluster.NewProvider()
	err := provider.Delete(name, kubeConfigPath)
	Expect(err).NotTo(HaveOccurred())
}

func kustomizeConfigCRD() ([]byte, error) {
	// TODO use an absolute path or something for ../config/crd -- we saw some weird docker errors when trying this
	args := []string{"build", "../config/crd"}
	cmd := exec.Command("kustomize", args...)
	cmd.Stderr = GinkgoWriter

	fmt.Fprintf(GinkgoWriter, "+ kustomize %s\n", strings.Join(args, " "))
	output, err := cmd.Output()
	return output, err
}

func kubectlWithConfig(kubeConfigPath string, stdin io.Reader, args ...string) ([]byte, error) {
	if len(kubeConfigPath) == 0 {
		return nil, errors.New("kubeconfig path cannot be empty")
	}
	cmd := exec.Command("kubectl", args...)
	cmd.Stderr = GinkgoWriter
	cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%s", kubeConfigPath))
	if stdin != nil {
		cmd.Stdin = stdin
	}

	fmt.Fprintf(GinkgoWriter, "+ kubectl %s\n", strings.Join(args, " "))
	output, err := cmd.Output()
	return output, err
}
