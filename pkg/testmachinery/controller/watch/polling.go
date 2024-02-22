// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package watch

import (
	"context"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	log "github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

// PollingInformerType specifies the polling informer type.
const PollingInformerType InformerType = "polling"

type pollingInformer struct {
	log          logr.Logger
	client       client.Client
	eventbus     EventBus
	pollInterval time.Duration

	old map[string]*tmv1beta1.Testrun
}

func newPollingInformer(log logr.Logger, config *rest.Config, options *Options) (Informer, error) {
	c, err := client.New(config, client.Options{
		Scheme: options.Scheme,
	})
	if err != nil {
		return nil, err
	}
	return &pollingInformer{
		log:          log,
		client:       c,
		pollInterval: *options.PollInterval,
		old:          make(map[string]*tmv1beta1.Testrun),
	}, nil
}

// Start starts the polling
func (p *pollingInformer) Start(ctx context.Context) error {
	return wait.PollUntilContextCancel(ctx, p.pollInterval, true, p.process)
}

func (p *pollingInformer) process(ctx context.Context) (done bool, err error) {
	defer ctx.Done()

	newOldCache := make(map[string]*tmv1beta1.Testrun)
	for _, key := range p.eventbus.Subscriptions() {
		nn, err := namespacedNameFromKey(key)
		if err != nil {
			p.log.Error(err, "invalid key identifier", "key", key)
			continue
		}

		tr := &tmv1beta1.Testrun{}
		if err := p.client.Get(ctx, nn, tr); err != nil {
			if !apierrors.IsNotFound(err) {
				log.Error(err, "unable to get tr", "key", key)
			}
			continue
		}

		if old, ok := p.old[key]; ok && reflect.DeepEqual(old, tr) {
			continue
		}
		newOldCache[key] = tr
		p.eventbus.Publish(key, tr)
	}

	p.old = newOldCache
	return false, nil
}

// WaitForCacheSync implements a noop operation for th polling informer as there are no caches to start
func (p *pollingInformer) WaitForCacheSync(_ context.Context) bool {
	return true
}

func (p *pollingInformer) InjectEventBus(eb EventBus) {
	p.eventbus = eb
}

func (p *pollingInformer) Client() client.Client {
	return p.client
}
