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
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

type WatchFunc func(*v1beta1.Testrun) (bool, error)

type Watch interface {
	reconcile.Reconciler

	// Watch registers a watch for a testrun resource until the testrun is completed
	Watch(namespace, name string, f WatchFunc) error

	WatchUntil(timeout time.Duration, namespace, name string, f WatchFunc) error

	Client() client.Client
}

func New(log logr.Logger, c client.Client) (Watch, error) {
	return &watch{
		log:     log,
		client:  c,
		watches: make(map[string]chan *v1beta1.Testrun),
	}, nil
}

type watch struct {
	log     logr.Logger
	client  client.Client
	watches map[string]chan *v1beta1.Testrun
}

func (w *watch) Client() client.Client {
	return w.client
}

func (w *watch) WatchUntil(timeout time.Duration, namespace, name string, f WatchFunc) error {
	namespacedName := types.NamespacedName{Namespace: namespace, Name: name}
	if _, ok := w.watches[namespacedName.String()]; !ok {
		w.watches[namespacedName.String()] = make(chan *v1beta1.Testrun)
	}

	var (
		errs  error
		done  = w.watches[namespacedName.String()]
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
		case tr := <-done:
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

func (w *watch) Reconcile(r reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	defer ctx.Done()
	ch, ok := w.watches[r.String()]
	if !ok {
		return reconcile.Result{}, nil
	}

	tr := &v1beta1.Testrun{}
	if err := w.client.Get(ctx, r.NamespacedName, tr); err != nil {
		w.log.Error(err, "unable to get testrun", "namespacedName", r.String())
		return reconcile.Result{Requeue: true}, nil
	}

	ch <- tr

	// close the channel if the testrun is completed
	if util.Completed(tr.Status.Phase) {
		close(ch)
		delete(w.watches, r.String())
	}

	return reconcile.Result{}, nil
}
