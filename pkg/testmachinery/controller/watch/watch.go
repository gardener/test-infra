// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package watch

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

type WatchFunc func(*tmv1beta1.Testrun) (bool, error)

type Watch interface {
	// Watch registers a watch for a testrun resource until the testrun is completed
	Watch(namespace, name string, f WatchFunc) error

	WatchUntil(timeout time.Duration, namespace, name string, f WatchFunc) error

	Client() client.Client

	// Start starts watcher adn its cache
	Start(<-chan struct{}) error

	// WaitForCacheSync waits for all the caches to sync.  Returns false if it could not sync a cache.
	WaitForCacheSync(stop <-chan struct{}) bool
}

type Options struct {
	// Scheme is the scheme
	Scheme *runtime.Scheme

	// SyncPeriod is the minimum time where resources are reconciled
	SyncPeriod *time.Duration

	// Namespace restrict the namespace to watch.
	// Leave this value empty to watch all namespaces.
	Namespace string
}

type watch struct {
	log logr.Logger

	cache  cache.Cache
	queue  workqueue.RateLimitingInterface
	client client.Client

	eventbus EventBus
}

// WaitForCacheSyncWithTimeout waits for all the caches to sync. Returns false if it could not sync a cache.
func WaitForCacheSyncWithTimeout(w Watch, d time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	if ok := w.WaitForCacheSync(ctx.Done()); !ok {
		return errors.New("error while waiting for cache")
	}
	return nil
}

// NewFromFile creates a new watch client from a kubeconfig file
func NewFromFile(log logr.Logger, kubeconfig string, options *Options) (Watch, error) {
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
		&clientcmd.ConfigOverrides{},
	)

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	return New(log, restConfig, options)
}

// New creates a new watch client
func New(log logr.Logger, config *rest.Config, options *Options) (Watch, error) {

	options = applyDefaultOptions(options)

	mapper, err := apiutil.NewDynamicRESTMapper(config)
	if err != nil {
		return nil, err
	}

	// Create the cache for the cached read client and registering informers
	cache, err := cache.New(config, cache.Options{Scheme: options.Scheme, Mapper: mapper, Resync: options.SyncPeriod, Namespace: options.Namespace})
	if err != nil {
		return nil, err
	}

	writer, err := client.New(config, client.Options{
		Scheme: options.Scheme,
		Mapper: mapper,
	})
	if err != nil {
		return nil, err
	}

	c := &client.DelegatingClient{
		Reader: &client.DelegatingReader{
			CacheReader:  cache,
			ClientReader: writer,
		},
		Writer:       writer,
		StatusClient: writer,
	}

	return &watch{
		log:      log,
		cache:    cache,
		client:   c,
		eventbus: NewEventBus(),
	}, nil
}

func (w *watch) Client() client.Client {
	return w.client
}

func (w *watch) WatchUntil(timeout time.Duration, namespace, name string, f WatchFunc) error {
	namespacedName := types.NamespacedName{Namespace: namespace, Name: name}

	ch := make(TestrunChannel)
	h := w.eventbus.Subscribe(namespacedName.String(), ch)

	// remove the watch from the list of watches to not leak watching channels
	defer w.eventbus.Unsubscribe(namespacedName.String(), h)

	var (
		errs  error
		after <-chan time.Time
	)

	if timeout != 0 {
		// time.After is more convenient, but it
		// potentially leaves timers around much longer
		// than necessary if we exit early.
		timer := time.NewTimer(timeout)
		after = timer.C
		defer timer.Stop()
	}

	for {
		select {
		case tr := <-ch:
			done, err := f(tr)
			if err != nil {
				if done {
					return multierror.Append(errs, err)
				}

				errs = multierror.Append(errs, err)
			} else if done {
				return nil
			}
		case <-after:
			return wait.ErrWaitTimeout
		}
	}
}

func (w *watch) Watch(namespace, name string, f WatchFunc) error {
	return w.WatchUntil(0, namespace, name, f)
}

func (w *watch) Start(stop <-chan struct{}) error {

	w.queue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "watch")
	defer w.queue.ShutDown()

	i, err := w.cache.GetInformer(&tmv1beta1.Testrun{})
	if err != nil {
		if kindMatchErr, ok := err.(*meta.NoKindMatchError); ok {
			w.log.Error(err, "if kind is a CRD, it should be installed before calling Start",
				"kind", kindMatchErr.GroupKind)
		}
		return err
	}
	i.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		AddFunc:    w.addItemToQueue,
		UpdateFunc: func(old, new interface{}) { w.addItemToQueue(new) },
		DeleteFunc: w.addItemToQueue,
	})

	go func() {
		if err := w.cache.Start(stop); err != nil {
			w.log.Error(err, "unable to start client cache")
			//return errors.Wrap(err, "unable to start client cache")
		}
	}()

	if ok := w.cache.WaitForCacheSync(stop); !ok {
		return errors.New("error while waiting for cache")
	}

	const jitterPeriod = 1 * time.Second

	// we only need one worker here
	go wait.Until(w.worker, jitterPeriod, stop)

	<-stop
	return nil
}

func (w *watch) WaitForCacheSync(stop <-chan struct{}) bool {
	return w.cache.WaitForCacheSync(stop)
}

// addItemToQueue adds the given object to the queue if it is applicable
func (w *watch) addItemToQueue(obj interface{}) {
	// try to cast to testrun ignore the event if this is not possible
	tr, ok := obj.(*tmv1beta1.Testrun)
	if !ok {
		return
	}

	if !w.eventbus.Has(keyOfTestrun(tr)) {
		return
	}

	w.queue.AddRateLimited(tr)
}

func (w *watch) worker() {
	for w.processQueue() {
	}
}

// processQueue takes the next item from the queue
func (w *watch) processQueue() bool {
	obj, shutdown := w.queue.Get()
	if shutdown {
		return false
	}
	defer w.queue.Done(obj)

	tr, ok := obj.(*tmv1beta1.Testrun)
	if !ok {
		// As the item in the workqueue is actually invalid, so remove it directly form the queue
		w.queue.Forget(obj)
		w.log.V(5).Info("watch queue item was not a testrun", "type", fmt.Sprintf("%T", obj), "value", obj)
		// Return true, don't take a break
		return true
	}

	w.eventbus.Publish(keyOfTestrun(tr), tr)

	w.queue.Forget(obj)
	return true
}
