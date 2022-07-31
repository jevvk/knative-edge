// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	"context"
	time "time"

	edgeknativedevv1 "github.com/jevvk/knative-edge/pkg/apis/edge.knative.dev/v1"
	versioned "github.com/jevvk/knative-edge/pkg/client/clientset/versioned"
	internalinterfaces "github.com/jevvk/knative-edge/pkg/client/informers/externalversions/internalinterfaces"
	v1 "github.com/jevvk/knative-edge/pkg/client/listers/edge.knative.dev/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// EdgeClusterInformer provides access to a shared informer and lister for
// EdgeClusters.
type EdgeClusterInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.EdgeClusterLister
}

type edgeClusterInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewEdgeClusterInformer constructs a new informer for EdgeCluster type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewEdgeClusterInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredEdgeClusterInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredEdgeClusterInformer constructs a new informer for EdgeCluster type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredEdgeClusterInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.EdgeV1().EdgeClusters().List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.EdgeV1().EdgeClusters().Watch(context.TODO(), options)
			},
		},
		&edgeknativedevv1.EdgeCluster{},
		resyncPeriod,
		indexers,
	)
}

func (f *edgeClusterInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredEdgeClusterInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *edgeClusterInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&edgeknativedevv1.EdgeCluster{}, f.defaultInformer)
}

func (f *edgeClusterInformer) Lister() v1.EdgeClusterLister {
	return v1.NewEdgeClusterLister(f.Informer().GetIndexer())
}
