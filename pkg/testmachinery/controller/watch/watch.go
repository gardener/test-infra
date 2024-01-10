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
	"time"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

type WatchFunc func(*tmv1beta1.Testrun) (bool, error)

// InformerType is the type of informer to be used.
type InformerType string

type Watch interface {
	// Watch registers a watch for a testrun resource until the testrun is completed
	Watch(namespace, name string, f WatchFunc) error

	WatchUntil(timeout time.Duration, namespace, name string, f WatchFunc) error

	Client() client.Client

	// Start starts watcher and its informer
	Start(ctx context.Context) error

	// WaitForCacheSync waits for all the caches to sync.  Returns false if it could not sync a informer.
	WaitForCacheSync(ctx context.Context) bool
}

// Informer is the internal watch that interacts with the cluster and publishes events to the event bus.
type Informer interface {
	// Start starts watcher and its informer
	Start(ctx context.Context) error

	// WaitForCacheSync waits for all the caches to sync.  Returns false if it could not sync a informer.
	WaitForCacheSync(ctx context.Context) bool

	// InjectEventBus injects the the event bus instance of the watch
	InjectEventBus(eb EventBus)

	// Client returns the used k8s client.
	Client() client.Client
}

type Options struct {
	// Scheme is the scheme
	Scheme *runtime.Scheme

	// InformerType is the type of the informer that should be used.
	InformerType InformerType

	// SyncPeriod is the minimum time where resources are reconciled
	// Only relevant if the cached informer should be used
	SyncPeriod *time.Duration

	// PollInterval is the time where informer polls the apiserver for an update.
	// Only relevant if the polling informer is used.
	PollInterval *time.Duration

	// Namespace restrict the namespace to watch.
	// Leave this value empty to watch all namespaces.
	Namespace string
}

type watch struct {
	log      logr.Logger
	informer Informer
	eventbus EventBus
}

// WaitForCacheSyncWithTimeout waits for all the caches to sync. Returns false if it could not sync a informer.
func WaitForCacheSyncWithTimeout(w Watch, d time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	if ok := w.WaitForCacheSync(ctx); !ok {
		return errors.New("error while waiting for informer")
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

	var inf Informer
	switch options.InformerType {
	case CachedInformerType:
		cInf, err := newCachedInformer(log, config, options)
		if err != nil {
			return nil, err
		}
		inf = cInf
	case PollingInformerType:
		pInf, err := newPollingInformer(log, config, options)
		if err != nil {
			return nil, err
		}
		inf = pInf
	default:
		return nil, errors.Errorf("unknown infromer type %s", options.InformerType)
	}

	return &watch{
		log:      log,
		informer: inf,
		eventbus: NewEventBus(log),
	}, nil
}

func (w *watch) Client() client.Client {
	return w.informer.Client()
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
			return wait.ErrorInterrupted(errors.New("timed out waiting for the condition"))
		}
	}
}

func (w *watch) Watch(namespace, name string, f WatchFunc) error {
	return w.WatchUntil(0, namespace, name, f)
}

func (w *watch) Start(ctx context.Context) error {

	go func() {
		if err := w.eventbus.Start(ctx.Done()); err != nil {
			w.log.Error(err, "unable to start the event bus")
		}
	}()

	w.informer.InjectEventBus(w.eventbus)

	go func() {
		if err := w.informer.Start(ctx); err != nil {
			w.log.Error(err, "unable to start informer")
		}
	}()

	<-ctx.Done()
	return nil
}

func (w *watch) WaitForCacheSync(ctx context.Context) bool {
	return w.informer.WaitForCacheSync(ctx)
}
