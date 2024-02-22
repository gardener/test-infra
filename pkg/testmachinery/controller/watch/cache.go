// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package watch

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
	ctrlcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

// CachedInformerType specifies the cached informer type.
const CachedInformerType InformerType = "cached"

type cachedInformer struct {
	log      logr.Logger
	client   client.Client
	cache    ctrlcache.Cache
	eventbus EventBus
}

func newCachedInformer(log logr.Logger, config *rest.Config, options *Options) (Informer, error) {
	httpClient, err := rest.HTTPClientFor(config)
	if err != nil {
		return nil, err
	}
	mapper, err := apiutil.NewDynamicRESTMapper(config, httpClient)
	if err != nil {
		return nil, err
	}

	// Create the informer for the cached read client and registering informers
	cache, err := ctrlcache.New(config, ctrlcache.Options{Scheme: options.Scheme, Mapper: mapper, SyncPeriod: options.SyncPeriod, DefaultNamespaces: map[string]ctrlcache.Config{
		options.Namespace: {},
	}})
	if err != nil {
		return nil, err
	}

	clientOpts := client.Options{}
	clientOpts.Scheme = options.Scheme
	clientOpts.Mapper = mapper
	clientOpts.Cache = &client.CacheOptions{}
	clientOpts.Cache.Reader = cache

	cachedClient, err := client.New(config, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("unable to create cached client: %w", err)
	}
	return &cachedInformer{
		log:    log,
		client: cachedClient,
		cache:  cache,
	}, nil
}

func (c *cachedInformer) Start(ctx context.Context) error {
	i, err := c.cache.GetInformer(ctx, &tmv1beta1.Testrun{})
	if err != nil {
		if kindMatchErr, ok := err.(*meta.NoKindMatchError); ok {
			c.log.Error(err, "if kind is a CRD, it should be installed before calling Start",
				"kind", kindMatchErr.GroupKind)
		}
		return err
	}
	_, err = i.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		AddFunc:    c.addItemToQueue,
		UpdateFunc: func(old, new interface{}) { c.addItemToQueue(new) },
		DeleteFunc: c.addItemToQueue,
	})
	if err != nil {
		return err
	}

	return c.cache.Start(ctx)
}

func (c *cachedInformer) WaitForCacheSync(ctx context.Context) bool {
	return c.cache.WaitForCacheSync(ctx)
}

func (c *cachedInformer) InjectEventBus(eb EventBus) {
	c.eventbus = eb
}

func (c *cachedInformer) Client() client.Client {
	return c.client
}

func (c *cachedInformer) addItemToQueue(obj interface{}) {
	// try to cast to testrun ignore the event if this is not possible
	tr, ok := obj.(*tmv1beta1.Testrun)
	if !ok {
		return
	}
	c.eventbus.Publish(keyOfTestrun(tr), tr)
}
