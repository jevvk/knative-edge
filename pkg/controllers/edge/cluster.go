package edge

import (
	"fmt"
	"net/http"
	"net/url"

	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

func NewRemoteClusterOrDie(opts ...cluster.Option) cluster.Cluster {
	kubeconfigPath := fmt.Sprintf("%s/%s", ConfigPath, KubeconfigFile)

	loader := clientcmd.NewDefaultClientConfigLoadingRules()
	loader.Precedence = append(loader.Precedence, kubeconfigPath)

	kubeconfig, err :=
		clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, nil).ClientConfig()

	if err != nil {
		panic(fmt.Errorf("couldn't retrieve kubeconfig: %w", err))
	}

	cluster, err := cluster.New(kubeconfig, opts...)

	if err != nil {
		panic(fmt.Errorf("couldn't create remote cluster: %w", err))
	}

	return cluster
}

func NewRemoteClusterWithProxyOrDie(proxy *url.URL, opts ...cluster.Option) cluster.Cluster {
	kubeconfigPath := fmt.Sprintf("%s/%s", ConfigPath, KubeconfigFile)

	loader := clientcmd.NewDefaultClientConfigLoadingRules()
	loader.Precedence = append(loader.Precedence, kubeconfigPath)

	kubeconfig, err :=
		clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, nil).ClientConfig()

	if err != nil {
		panic(fmt.Errorf("couldn't retrieve kubeconfig: %w", err))
	}

	kubeconfig.Proxy = func(req *http.Request) (*url.URL, error) {
		return proxy, nil
	}

	cluster, err := cluster.New(kubeconfig, opts...)

	if err != nil {
		panic(fmt.Errorf("couldn't create remote cluster: %w", err))
	}

	return cluster
}
