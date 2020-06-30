// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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
	"sync"

	"k8s.io/apimachinery/pkg/types"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

type TestrunChannel chan *tmv1beta1.Testrun

type EventBusHandle *int

type EventBus interface {
	Subscribe(key string, ch TestrunChannel) EventBusHandle
	Unsubscribe(key string, h EventBusHandle)
	Publish(key string, tr *tmv1beta1.Testrun)
	Has(key string) bool
}

type eventbus struct {
	watches map[string]map[EventBusHandle]TestrunChannel
	mux     sync.RWMutex
}

func NewEventBus() EventBus {
	return &eventbus{
		watches: make(map[string]map[EventBusHandle]TestrunChannel),
		mux:     sync.RWMutex{},
	}
}

func (e *eventbus) Publish(key string, tr *tmv1beta1.Testrun) {
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

// has returns whether a the requested is watched
func (e *eventbus) Has(key string) bool {
	e.mux.RLock()
	defer e.mux.RUnlock()
	bus, ok := e.watches[key]
	if !ok {
		return false
	}
	return len(bus) != 0
}

func keyOfTestrun(tr *tmv1beta1.Testrun) string {
	return types.NamespacedName{Name: tr.Name, Namespace: tr.Namespace}.String()
}
