package edge

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	edgecontrollers "edge.jevv.dev/pkg/controllers"
)

var _ = Describe("knative service controller", func() {
	const (
		timeout  = time.Second * 1
		duration = time.Second * 10
		interval = time.Millisecond * 250
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

			Eventually(func() bool {
				err := edgeClusterClient.Get(ctx, namespacedName, mirroredService)
				return err == nil
			}, timeout, interval).Should(BeTrue())
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
			DeferCleanup(func() {
				Expect(remoteClusterClient.Delete(ctx, service)).Should(Succeed())
			})

			Eventually(func() bool {
				if err := edgeClusterClient.Get(ctx, namespacedName, mirroredService); err != nil {
					return false
				}

				if value, ok := mirroredService.Annotations["check"]; ok {
					return value == "before"
				}

				return false
			}, timeout, interval).Should(BeTrue())

			By("updating the service")
			service.Annotations["check"] = "after"
			service.Spec.Template.Spec.Containers[0].Env[0].Value = "world"
			mirroredService = &servingv1.Service{}

			Expect(remoteClusterClient.Update(ctx, service)).Should(Succeed())

			Eventually(func() bool {
				if err := edgeClusterClient.Get(ctx, namespacedName, mirroredService); err != nil {
					return false
				}

				if value, ok := mirroredService.Annotations["check"]; ok && value != "after" {
					return false
				}

				return mirroredService.Spec.Template.Spec.Containers[0].Env[0].Value == "world"
			}, timeout, interval).Should(BeTrue())
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

			Eventually(func() bool {
				if err := edgeClusterClient.Get(ctx, namespacedName, mirroredService); err != nil {
					return false
				}

				return true
			}, timeout, interval).Should(BeTrue())

			By("deleting the service")
			Expect(remoteClusterClient.Delete(ctx, service)).Should(Succeed())

			Eventually(func() bool {
				if err := edgeClusterClient.Get(ctx, namespacedName, mirroredService); err != nil {
					return true
				}

				return false
			}, timeout, interval).Should(BeTrue())
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
					Annotations: map[string]string{
						edgecontrollers.OffloadToRemoteAnnotation: "true",
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
			DeferCleanup(func() {
				Expect(remoteClusterClient.Delete(ctx, service)).Should(Succeed())
			})

			Eventually(func() bool {
				err := edgeClusterClient.Get(ctx, namespacedName, mirroredService)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(len(mirroredService.Spec.RouteSpec.Traffic)).Should(BeNumerically(">", 0))
			Expect(len(mirroredService.Spec.RouteSpec.Traffic)).Should(BeNumerically("<=", 2))

			Eventually(func() bool {
				err := edgeClusterClient.Get(ctx, namespacedName, mirroredService)

				if err != nil {
					return false
				}

				return len(mirroredService.Spec.RouteSpec.Traffic) == 2
			}, timeout, interval).Should(BeTrue())

			revision := &servingv1.Revision{}

			Eventually(func() bool {
				err := edgeClusterClient.Get(ctx, getRevisionNamespacedName(namespacedName), revision)
				return err == nil
			}, timeout, interval).Should(BeTrue())
		})
	})
})
