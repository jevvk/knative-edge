package reflector

import (
	"fmt"
	"os"

	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

func NewRemoteClusterOrDie(opts ...cluster.Option) cluster.Cluster {
	bytes, err := os.ReadFile(fmt.Sprintf("%s/%s", ConfigPath, KubeconfigFile))

	if err != nil {
		panic(fmt.Errorf("couldn't read remote cluster kubeconfig: %w", err))
	}

	config, err := clientcmd.NewClientConfigFromBytes(bytes)

	if err != nil {
		panic(fmt.Errorf("couldn't parse kubeconfig: %w", err))
	}

	kubeconfig, err := config.ClientConfig()

	if err != nil {
		panic(fmt.Errorf("couldn't retrieve kubeconfig: %w", err))
	}

	cluster, err := cluster.New(kubeconfig)

	if err != nil {
		panic(fmt.Errorf("couldn't create remote cluster: %w", err))
	}

	return cluster
}
