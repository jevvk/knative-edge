package edge

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"edge.jevv.dev/pkg/controllers"
	edgecontrollers "edge.jevv.dev/pkg/controllers"
	"edge.jevv.dev/pkg/controllers/utils"
)

var _ = Describe("knative service controller", func() {
	const (
		timeout  = time.Second * 1
		duration = time.Second * 10
		interval = time.Millisecond * 250

		revisionTimeout = time.Second * 10
	)

	var _ = BeforeEach(func() {
		time.Sleep(time.Millisecond * 200)
	})

	Context("when creating new resource", func() {
		It("should replicate resources", func() {
			var mirroredService, service *servingv1.Service
			ctx := context.Background()

			By("creating a service")
			namespacedName := types.NamespacedName{Name: "service-test-1", Namespace: "default"}

			mirroredService = &servingv1.Service{}
			service = &servingv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namespacedName.Name,
					Namespace: namespacedName.Namespace,
					Labels: map[string]string{
						edgecontrollers.AppLabel:         "knative-edge",
						edgecontrollers.EnvironmentLabel: "testA",
					},
				},
				Spec: servingv1.ServiceSpec{
					ConfigurationSpec: servingv1.ConfigurationSpec{
						Template: servingv1.RevisionTemplateSpec{
							Spec: servingv1.RevisionSpec{
								PodSpec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Image: "gcr.io/knative-samples/helloworld-go",
											Env: []corev1.EnvVar{
												{
													Name:  "TARGET",
													Value: "World",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			Expect(remoteClusterClient.Create(ctx, service)).Should(Succeed())
			DeferCleanup(func() {
				Expect(remoteClusterClient.Delete(ctx, service)).Should(Succeed())
			})

			Eventually(func() error {
				return edgeClusterClient.Get(ctx, namespacedName, mirroredService)
			}, timeout, interval).Should(Not(Succeed()))
		})

		It("should update replicated resources", func() {
			var mirroredService, service *servingv1.Service
			ctx := context.Background()

			By("creating a service")
			namespacedName := types.NamespacedName{Name: "service-test-2", Namespace: "default"}

			mirroredService = &servingv1.Service{}
			service = &servingv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namespacedName.Name,
					Namespace: namespacedName.Namespace,
					Labels: map[string]string{
						edgecontrollers.AppLabel:         "knative-edge",
						edgecontrollers.EnvironmentLabel: "testA",
						"check":                          "before",
					},
				},
				Spec: servingv1.ServiceSpec{
					ConfigurationSpec: servingv1.ConfigurationSpec{
						Template: servingv1.RevisionTemplateSpec{
							Spec: servingv1.RevisionSpec{
								PodSpec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Image: "gcr.io/knative-samples/helloworld-go",
											Env: []corev1.EnvVar{
												{
													Name:  "TARGET",
													Value: "World",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			Expect(remoteClusterClient.Create(ctx, service)).Should(Succeed())
			DeferCleanup(func() {
				Expect(remoteClusterClient.Delete(ctx, service)).Should(Succeed())
			})

			Eventually(func(g Gomega) {
				g.Expect(edgeClusterClient.Get(ctx, namespacedName, mirroredService)).Should(Succeed())
				g.Expect(mirroredService.Labels["check"]).To(Equal("before"))
				g.Expect(mirroredService.Spec.Template.Spec.Containers[0].Env[0].Value).To(Equal("World"))
			}, timeout, interval).Should(Succeed())

			By("updating the service")
			service.Labels["check"] = "after"
			service.Spec.Template.Spec.Containers[0].Env[0].Value = "world"
			mirroredService = &servingv1.Service{}

			Expect(remoteClusterClient.Update(ctx, service)).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(edgeClusterClient.Get(ctx, namespacedName, mirroredService)).Should(Succeed())
				g.Expect(mirroredService.Labels["check"]).To(Equal("after"))
				g.Expect(mirroredService.Spec.Template.Spec.Containers[0].Env[0].Value).To(Equal("world"))
			}, timeout, interval).Should(Succeed())
		})

		It("should delete replicated resources", func() {
			var mirroredService, service *servingv1.Service
			ctx := context.Background()

			By("creating a service")
			namespacedName := types.NamespacedName{Name: "service-test-3", Namespace: "default"}

			mirroredService = &servingv1.Service{}
			service = &servingv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namespacedName.Name,
					Namespace: namespacedName.Namespace,
					Annotations: map[string]string{
						"check": "before",
					},
					Labels: map[string]string{
						edgecontrollers.AppLabel:         "knative-edge",
						edgecontrollers.EnvironmentLabel: "testA",
					},
				},
				Spec: servingv1.ServiceSpec{
					ConfigurationSpec: servingv1.ConfigurationSpec{
						Template: servingv1.RevisionTemplateSpec{
							Spec: servingv1.RevisionSpec{
								PodSpec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Image: "gcr.io/knative-samples/helloworld-go",
											Env: []corev1.EnvVar{
												{
													Name:  "TARGET",
													Value: "World",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			Expect(remoteClusterClient.Create(ctx, service)).Should(Succeed())

			Eventually(func() error {
				return edgeClusterClient.Get(ctx, namespacedName, mirroredService)
			}, timeout, interval).Should(Succeed())

			By("deleting the service")
			Expect(remoteClusterClient.Delete(ctx, service)).Should(Succeed())

			Eventually(func() error {
				return edgeClusterClient.Get(ctx, namespacedName, mirroredService)
			}, timeout, interval).Should(Not(Succeed()))

			// Eventually(func() error {
			// 	return fmt.Errorf("hello world")
			// }, timeout, interval).Should(Succeed())
		})
	})

	Context("when offloading compute", func() {
		It("should add edge proxy route", func() {
			var mirroredService, service *servingv1.Service
			ctx := context.Background()

			By("creating a service")
			namespacedName := types.NamespacedName{Name: "service-test-4", Namespace: "default"}

			mirroredService = &servingv1.Service{}
			service = &servingv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namespacedName.Name,
					Namespace: namespacedName.Namespace,
					Labels: map[string]string{
						edgecontrollers.EdgeOffloadLabel: "true",
						edgecontrollers.EnvironmentLabel: "testA",
					},
				},
				Spec: servingv1.ServiceSpec{
					ConfigurationSpec: servingv1.ConfigurationSpec{
						Template: servingv1.RevisionTemplateSpec{
							Spec: servingv1.RevisionSpec{
								PodSpec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Image: "gcr.io/knative-samples/helloworld-go",
											Env: []corev1.EnvVar{
												{
													Name:  "TARGET",
													Value: "World",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			Expect(remoteClusterClient.Create(ctx, service)).Should(Succeed())
			DeferCleanup(func() {
				Expect(remoteClusterClient.Delete(ctx, service)).Should(Succeed())
			})

			Eventually(func(g Gomega) {
				g.Expect(edgeClusterClient.Get(ctx, namespacedName, mirroredService)).Should(Succeed())
				g.Expect(mirroredService.Spec.RouteSpec.Traffic).Should(HaveLen(2))
			}, revisionTimeout, interval).Should(Succeed())

			revision := &servingv1.Revision{}

			Eventually(func() error {
				return edgeClusterClient.Get(ctx, utils.GetConfigurationNamespacedName(namespacedName), revision)
			}, timeout, interval).Should(Succeed())
		})

		It("should delete edge proxy route", func() {
			var mirroredService, service *servingv1.Service
			ctx := context.Background()

			By("creating a service")
			namespacedName := types.NamespacedName{Name: "service-test-4", Namespace: "default"}

			mirroredService = &servingv1.Service{}
			service = &servingv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namespacedName.Name,
					Namespace: namespacedName.Namespace,
					Labels: map[string]string{
						edgecontrollers.EdgeOffloadLabel: "true",
						edgecontrollers.EnvironmentLabel: "testA",
					},
				},
				Spec: servingv1.ServiceSpec{
					ConfigurationSpec: servingv1.ConfigurationSpec{
						Template: servingv1.RevisionTemplateSpec{
							Spec: servingv1.RevisionSpec{
								PodSpec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Image: "gcr.io/knative-samples/helloworld-go",
											Env: []corev1.EnvVar{
												{
													Name:  "TARGET",
													Value: "World",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			Expect(remoteClusterClient.Create(ctx, service)).Should(Succeed())
			DeferCleanup(func() {
				Expect(remoteClusterClient.Delete(ctx, service)).Should(Succeed())
			})

			Eventually(func(g Gomega) {
				g.Expect(edgeClusterClient.Get(ctx, namespacedName, mirroredService)).Should(Succeed())
				g.Expect(mirroredService.Spec.RouteSpec.Traffic).Should(HaveLen(2))
			}, revisionTimeout, interval).Should(Succeed())

			revision := &servingv1.Revision{}

			Eventually(func() error {
				return edgeClusterClient.Get(ctx, utils.GetConfigurationNamespacedName(namespacedName), revision)
			}, timeout, interval).Should(Succeed())

			Expect(remoteClusterClient.Get(ctx, namespacedName, service)).Should(Succeed())
			service = service.DeepCopy()

			By("disabling offload label")
			service.Labels[controllers.EdgeOffloadLabel] = "false"
			Expect(remoteClusterClient.Update(ctx, service)).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(edgeClusterClient.Get(ctx, namespacedName, mirroredService)).Should(Succeed())
				g.Expect(mirroredService.Spec.RouteSpec.Traffic).Should(HaveLen(0))
			}, revisionTimeout, interval).Should(Succeed())

			Eventually(func() error {
				if err := edgeClusterClient.Get(ctx, utils.GetConfigurationNamespacedName(namespacedName), revision); err != nil && apierrors.IsNotFound(err) {
					return nil
				}

				return fmt.Errorf("should not exist")
			}, timeout, interval).Should(Succeed())
		})
	})
})
