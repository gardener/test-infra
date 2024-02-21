// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package watch

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

type TestrunChannel chan *tmv1beta1.Testrun

type EventBusHandle *int

type EventBus interface {
	// Start starts watcher and its informer
	Start(<-chan struct{}) error
	// Subscribe registers a subscription to receive vents published for a specific key.
	Subscribe(key string, ch TestrunChannel) EventBusHandle
	// Unsubscribe removes the subscription from a key with the unique handle.
	Unsubscribe(key string, h EventBusHandle)
	// Publish sends a update to all registered subscription for the specific key.
	Publish(key string, tr *tmv1beta1.Testrun)
	// Has checks if the given is subscribed by anyone.
	Has(key string) bool
	// Subscriptions returns a list of all watched keys.
	Subscriptions() []string
}

type eventbus struct {
	log     logr.Logger
	queue   workqueue.RateLimitingInterface
	watches map[string]map[EventBusHandle]TestrunChannel
	mux     sync.RWMutex
}

func NewEventBus(log logr.Logger) EventBus {
	return &eventbus{
		log:     log,
		watches: make(map[string]map[EventBusHandle]TestrunChannel),
		mux:     sync.RWMutex{},
	}
}

func (e *eventbus) Start(stop <-chan struct{}) error {
	e.queue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "watch")
	defer e.queue.ShutDown()

	const jitterPeriod = 1 * time.Second

	// we only need one worker here
	go wait.Until(e.worker, jitterPeriod, stop)

	<-stop
	return nil
}

func (e *eventbus) Publish(key string, tr *tmv1beta1.Testrun) {
	e.mux.RLock()
	_, ok := e.watches[key]
	defer e.mux.RUnlock()
	if !ok {
		return
	}

	e.queue.Add(tr)
}

// publish should be only called from the rate limited workqueue and does the actual publish of events
func (e *eventbus) publish(key string, tr *tmv1beta1.Testrun) {
	e.mux.RLock()
	defer e.mux.RUnlock()
	bus, ok := e.watches[key]
	if !ok {
		return
	}

	for _, ch := range bus {
		go func(ch TestrunChannel) {
			select {
			case ch <- tr:
			default:
			}
		}(ch)
	}
}

func (e *eventbus) Subscribe(key string, ch TestrunChannel) EventBusHandle {
	h := EventBusHandle(new(int))

	e.mux.Lock()
	defer e.mux.Unlock()

	bus, ok := e.watches[key]
	if !ok {
		bus = make(map[EventBusHandle]TestrunChannel, 1)
		e.watches[key] = bus
	}

	bus[h] = ch

	return h
}

func (e *eventbus) Unsubscribe(key string, h EventBusHandle) {
	e.mux.Lock()
	defer e.mux.Unlock()

	bus, ok := e.watches[key]
	if !ok {
		return
	}

	ch, ok := bus[h]
	if !ok {
		return
	}

	close(ch)
	delete(bus, h)
}

// Has returns whether a the requested is watched
func (e *eventbus) Has(key string) bool {
	e.mux.RLock()
	defer e.mux.RUnlock()
	bus, ok := e.watches[key]
	if !ok {
		return false
	}
	return len(bus) != 0
}

// Subscriptions returns all subscribed testruns
func (e *eventbus) Subscriptions() []string {
	e.mux.RLock()
	defer e.mux.RUnlock()
	watches := make([]string, len(e.watches))
	i := 0
	for key := range e.watches {
		watches[i] = key
		i++
	}
	return watches
}

func (e *eventbus) worker() {
	for e.processQueue() {
	}
}

// processQueue takes the next item from the queue
func (e *eventbus) processQueue() bool {
	obj, shutdown := e.queue.Get()
	if shutdown {
		return false
	}
	defer e.queue.Done(obj)

	tr, ok := obj.(*tmv1beta1.Testrun)
	if !ok {
		// As the item in the workqueue is actually invalid, so remove it directly form the queue
		e.queue.Forget(obj)
		e.log.V(5).Info("watch queue item was not a testrun", "type", fmt.Sprintf("%T", obj), "value", obj)
		// Return true, don't take a break
		return true
	}

	e.publish(keyOfTestrun(tr), tr)

	e.queue.Forget(obj)
	return true
}

func keyOfTestrun(tr *tmv1beta1.Testrun) string {
	return types.NamespacedName{Name: tr.Name, Namespace: tr.Namespace}.String()
}

func namespacedNameFromKey(key string) (types.NamespacedName, error) {
	splitKey := strings.Split(key, string(types.Separator))
	if len(splitKey) != 2 {
		return types.NamespacedName{}, errors.New("invalid namespaced name")
	}
	return types.NamespacedName{
		Name:      splitKey[1],
		Namespace: splitKey[0],
	}, nil
}
