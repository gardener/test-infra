// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package configwatcher

import (
	"context"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/gardener/test-infra/pkg/apis/config"
	"github.com/gardener/test-infra/pkg/testmachinery"
)

type NotifyFunc = func(ctx context.Context, configuration *config.Configuration) error

// ConfigWatcher watches changes to the configuration at the given path.
// It reads and parses the config on update to the file
type ConfigWatcher struct {
	sync.Mutex
	log logr.Logger

	watcher *fsnotify.Watcher
	decoder runtime.Decoder

	notify NotifyFunc

	configpath    string
	currentConfig *config.Configuration
}

// New returns a new watcher that watches the given config
func New(log logr.Logger, configpath string) (*ConfigWatcher, error) {
	var err error
	cw := &ConfigWatcher{
		log:        log,
		configpath: configpath,
		decoder:    serializer.NewCodecFactory(testmachinery.ConfigScheme).UniversalDecoder(),
	}

	if err := cw.ReadConfiguration(); err != nil {
		return nil, errors.Wrap(err, "error re-reading configuration")
	}

	cw.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return cw, nil
}

// InjectReconciler injects a reconciler to the watches that is called when the config changes
func (cw *ConfigWatcher) InjectNotifyFunc(f NotifyFunc) {
	cw.notify = f
}

// Start starts the watch on the certificate and key files.
func (cw *ConfigWatcher) Start(stopCh <-chan struct{}) error {
	if err := cw.watcher.Add(cw.configpath); err != nil {
		return err
	}

	go cw.Watch()
	// Block until the stop channel is closed.
	<-stopCh

	return cw.watcher.Close()
}

// Watch reads events from the watcher's channel and reacts to changes.
func (cw *ConfigWatcher) Watch() {
	for {
		select {
		case event, ok := <-cw.watcher.Events:
			// Channel is closed.
			if !ok {
				return
			}

			cw.handleEvent(event)

		case err, ok := <-cw.watcher.Errors:
			// Channel is closed.
			if !ok {
				return
			}

			cw.log.Error(err, "config watch error")
		}
	}
}

func (cw *ConfigWatcher) handleEvent(event fsnotify.Event) {
	// Only care about events which may modify the contents of the file.
	if !(isWrite(event) || isRemove(event) || isCreate(event)) {
		return
	}
	ctx := context.Background()
	defer ctx.Done()

	cw.log.V(3).Info("config event", "event", event)

	// If the file was removed, re-add the watch.
	if isRemove(event) {
		if err := cw.watcher.Add(event.Name); err != nil {
			cw.log.Error(err, "error re-watching file")
		}
	}

	if err := cw.ReadConfiguration(); err != nil {
		cw.log.Error(err, "error re-reading configuration")
	}

	// call reconciler to notify for changes
	if cw.notify == nil {
		return
	}
	if err := cw.notify(ctx, cw.currentConfig); err != nil {
		// todo: schrodit - maybe we should hard exit we are unable to reconcile
		cw.log.Error(err, "unable to reconcile with new config")
	}
}

func (cw *ConfigWatcher) GetConfiguration() *config.Configuration {
	cw.Lock()
	defer cw.Unlock()
	return cw.currentConfig
}

// ReadConfiguration reads the configuration from the file system
func (cw *ConfigWatcher) ReadConfiguration() error {
	file, err := os.ReadFile(cw.configpath)
	if err != nil {
		return err
	}

	cfg := &config.Configuration{}
	if _, _, err := cw.decoder.Decode(file, nil, cfg); err != nil {
		return err
	}

	cw.Lock()
	if cw.currentConfig == nil {
		cw.currentConfig = cfg
	} else {
		*cw.currentConfig = *cfg
	}
	cw.Unlock()

	return nil
}

func isWrite(event fsnotify.Event) bool {
	return event.Op&fsnotify.Write == fsnotify.Write
}

func isCreate(event fsnotify.Event) bool {
	return event.Op&fsnotify.Create == fsnotify.Create
}

func isRemove(event fsnotify.Event) bool {
	return event.Op&fsnotify.Remove == fsnotify.Remove
}
