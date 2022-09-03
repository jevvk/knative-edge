package operator

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

type operatorContextKey string

var (
	kubeconfigSecretKey operatorContextKey = "kubeconfig-ref-secret"
)

func withKubeconfigInContext(ctx context.Context, kubeconfigSecret *corev1.Secret) context.Context {
	if kubeconfigSecret == nil {
		return ctx
	}

	return context.WithValue(ctx, kubeconfigSecretKey, kubeconfigSecret)
}

func withKubeconfigFromContext(ctx context.Context, refSecret *corev1.Secret) {
	kubeconfigSecret := ctx.Value(kubeconfigSecretKey)

	if kubeconfigSecret != nil {
		if secret, ok := kubeconfigSecret.(*corev1.Secret); ok {
			*refSecret = *secret
		}
	}
}
