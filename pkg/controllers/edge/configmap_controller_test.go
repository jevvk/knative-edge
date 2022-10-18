package edge

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	edgecontrollers "edge.jevv.dev/pkg/controllers"
)

var _ = Describe("configmap controller", func() {
	const (
		timeout  = time.Second * 1
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("when creating new resource", func() {
		It("should replicate resources", func() {
			By("creating a configmap")
			ctx := context.Background()

			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "configmap-test",
					Namespace: "default",
					Labels: map[string]string{
						edgecontrollers.AppLabel:         "knative-edge",
						edgecontrollers.EnvironmentLabel: "testA",
					},
				},
				Data: map[string]string{
					"foo":  "bar",
					"fooo": "baar",
				},
			}

			Expect(remoteClusterClient.Create(ctx, configMap)).Should(Succeed())

			namespacedName := types.NamespacedName{Name: "configmap-test", Namespace: "default"}
			mirroredConfigMap := &corev1.ConfigMap{}

			Eventually(func() bool {
				err := edgeClusterClient.Get(ctx, namespacedName, mirroredConfigMap)
				return err == nil
			}, timeout, interval).Should(BeTrue())
		})
	})
})
