package reboot

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zapcore"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	selfnoderemediationv1alpha1 "github.com/medik8s/self-node-remediation/api/v1alpha1"
)

var testEnv *envtest.Environment
var k8sClient client.Client
var cfg *rest.Config
var cancelFunc context.CancelFunc

func TestWatchdog(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rebooter Suite")
}

var _ = BeforeSuite(func() {
	By("bootstrapping test environment")
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339NanoTimeEncoder,
	}
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseFlagOptions(&opts)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = selfnoderemediationv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	gracefulShutdown := 0 * time.Second
	Expect(err).ToNot(HaveOccurred())
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                  scheme.Scheme,
		LeaderElection:          false,
		MetricsBindAddress:      "0",
		GracefulShutdownTimeout: &gracefulShutdown,
	})
	Expect(err).ToNot(HaveOccurred())

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())

	var ctx context.Context
	ctx, cancelFunc = context.WithCancel(ctrl.SetupSignalHandler())
	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
	}()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancelFunc()
	Expect(testEnv.Stop()).To(Succeed())
})
