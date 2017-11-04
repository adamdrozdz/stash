/*
Copyright 2017 The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file was automatically generated by informer-gen

package v1alpha1

import (
	stash_v1alpha1 "github.com/appscode/stash/apis/stash/v1alpha1"
	client "github.com/appscode/stash/client"
	internalinterfaces "github.com/appscode/stash/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/appscode/stash/listers/stash/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
	time "time"
)

// ResticInformer provides access to a shared informer and lister for
// Restics.
type ResticInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.ResticLister
}

type resticInformer struct {
	factory internalinterfaces.SharedInformerFactory
}

// NewResticInformer constructs a new informer for Restic type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewResticInformer(client client.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				return client.StashV1alpha1().Restics(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return client.StashV1alpha1().Restics(namespace).Watch(options)
			},
		},
		&stash_v1alpha1.Restic{},
		resyncPeriod,
		indexers,
	)
}

func defaultResticInformer(client client.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewResticInformer(client, v1.NamespaceAll, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
}

func (f *resticInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&stash_v1alpha1.Restic{}, defaultResticInformer)
}

func (f *resticInformer) Lister() v1alpha1.ResticLister {
	return v1alpha1.NewResticLister(f.Informer().GetIndexer())
}