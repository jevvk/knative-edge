package edge

import (
	"context"
	"path/filepath"
	"testing"

	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	edgev1alpha1 "edge.jevv.dev/pkg/apis/edge/v1alpha1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

func TestRun(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Edge controller Integration Suite")
}

var stop context.CancelFunc

var edgeClusterCfg *rest.Config
var remoteClusterCfg *rest.Config

var edgeClusterTestEnv *envtest.Environment
var remoteClusterTestEnv *envtest.Environment

var edgeClusterClient client.Client
var remoteClusterClient client.Client

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	edgeClusterTestEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crd", "overlays", "edge")},
		ErrorIfCRDPathMissing: true,
	}

	remoteClusterTestEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crd", "overlays", "cloud")},
		ErrorIfCRDPathMissing: true,
	}

	var err error

	edgeClusterCfg, err = edgeClusterTestEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(edgeClusterCfg).NotTo(BeNil())

	remoteClusterCfg, err = remoteClusterTestEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(remoteClusterCfg).NotTo(BeNil())

	err = edgev1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = servingv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	edgeClusterClient, err = client.New(edgeClusterCfg, client.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).NotTo(HaveOccurred())

	remoteClusterClient, err = client.New(remoteClusterCfg, client.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).NotTo(HaveOccurred())

	var ctx context.Context
	ctx, stop = context.WithCancel(context.TODO())

	go func() {
		envs := []string{"testA", "testB"}

		defer GinkgoRecover()

		mgr, err := ctrl.NewManager(edgeClusterCfg, ctrl.Options{
			Scheme: scheme.Scheme,
		})
		Expect(err).ToNot(HaveOccurred())

		remoteCluster, err := cluster.New(remoteClusterCfg, func(o *cluster.Options) {
			o.Scheme = scheme.Scheme
			o.NewCache = EnvScopedCache(envs)
		})
		Expect(err).ToNot(HaveOccurred())

		err = mgr.Add(remoteCluster)
		Expect(err).ToNot(HaveOccurred())

		hasEdgeLabelPredicate := HasEdgeSyncLabelPredicate(envs)

		err = (&ConfigMapReconciler{
			Client:        mgr.GetClient(),
			Scheme:        mgr.GetScheme(),
			Log:           mgr.GetLogger().WithName("configmap-controller"),
			Recorder:      mgr.GetEventRecorderFor("configmap-controller"),
			RemoteCluster: remoteCluster,
			Envs:          envs,
		}).SetupWithManager(mgr, hasEdgeLabelPredicate)
		Expect(err).ToNot(HaveOccurred())

		err = mgr.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
	}()
})

var _ = AfterSuite(func() {
	stop()
	Expect(edgeClusterTestEnv.Stop()).To(Succeed())
	Expect(remoteClusterTestEnv.Stop()).To(Succeed())
})
