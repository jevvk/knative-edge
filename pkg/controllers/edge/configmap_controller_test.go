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
			var mirroredConfigMap, configMap *corev1.ConfigMap
			ctx := context.Background()

			By("creating a configmap")
			namespacedName := types.NamespacedName{Name: "configmap-test-1", Namespace: "default"}

			mirroredConfigMap = &corev1.ConfigMap{}
			configMap = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namespacedName.Name,
					Namespace: namespacedName.Namespace,
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
			DeferCleanup(func() {
				Expect(remoteClusterClient.Delete(ctx, configMap)).Should(Succeed())
			})

			Eventually(func() error {
				return edgeClusterClient.Get(ctx, namespacedName, mirroredConfigMap)
			}, timeout, interval).Should(Succeed())
		})

		It("should update replicated resources", func() {
			var mirroredConfigMap, configMap *corev1.ConfigMap
			ctx := context.Background()

			By("creating a configmap")
			namespacedName := types.NamespacedName{Name: "configmap-test-2", Namespace: "default"}

			mirroredConfigMap = &corev1.ConfigMap{}
			configMap = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namespacedName.Name,
					Namespace: namespacedName.Namespace,
					Labels: map[string]string{
						edgecontrollers.AppLabel:         "knative-edge",
						edgecontrollers.EnvironmentLabel: "testA",
					},
				},
				Data: map[string]string{
					"check": "before",
				},
			}

			Expect(remoteClusterClient.Create(ctx, configMap)).Should(Succeed())
			DeferCleanup(func() {
				Expect(remoteClusterClient.Delete(ctx, configMap)).Should(Succeed())
			})

			Eventually(func(g Gomega) {
				g.Expect(edgeClusterClient.Get(ctx, namespacedName, mirroredConfigMap)).To(Succeed())
				g.Expect(mirroredConfigMap.Data["check"]).To(Equal("before"))
			}, timeout, interval).Should(Succeed())

			By("updating the configmap")
			configMap.Data["check"] = "after"
			mirroredConfigMap = &corev1.ConfigMap{}

			Expect(remoteClusterClient.Update(ctx, configMap)).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(edgeClusterClient.Get(ctx, namespacedName, mirroredConfigMap)).To(Succeed())
				g.Expect(mirroredConfigMap.Data["check"]).To(Equal("after"))
			}, timeout, interval).Should(Succeed())
		})

		It("should delete replicated resources", func() {
			var mirroredConfigMap, configMap *corev1.ConfigMap
			ctx := context.Background()

			By("creating a configmap")
			namespacedName := types.NamespacedName{Name: "configmap-test-3", Namespace: "default"}

			mirroredConfigMap = &corev1.ConfigMap{}
			configMap = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namespacedName.Name,
					Namespace: namespacedName.Namespace,
					Labels: map[string]string{
						edgecontrollers.AppLabel:         "knative-edge",
						edgecontrollers.EnvironmentLabel: "testA",
					},
				},
				Data: map[string]string{
					"foo": "bar",
				},
			}

			Expect(remoteClusterClient.Create(ctx, configMap)).Should(Succeed())

			Eventually(func() error {
				return edgeClusterClient.Get(ctx, namespacedName, mirroredConfigMap)
			}, timeout, interval).Should(Succeed())

			By("deleting the configmap")
			Expect(remoteClusterClient.Delete(ctx, configMap)).Should(Succeed())

			Eventually(func() error {
				return edgeClusterClient.Get(ctx, namespacedName, mirroredConfigMap)
			}, timeout, interval).Should(Not(Succeed()))
		})
	})
})
