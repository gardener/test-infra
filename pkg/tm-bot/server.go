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

package tm_bot

import (
	"context"
	"fmt"
	"github.com/gardener/test-infra/pkg/apis/config"
	"k8s.io/client-go/rest"
	"net/http"
	"os"
	"time"

	"github.com/go-logr/logr"
)

// Serve starts the webhook server for testrun validation
func Serve(ctx context.Context, log logr.Logger, restConfig *rest.Config, cfg *config.BotConfiguration) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	opts := &options{
		log:        log,
		restConfig: restConfig,
		cfg:        cfg,
	}

	r, err := opts.Complete(stopCh)
	if err != nil {
		log.Error(err, "unable to setup components")
		os.Exit(1)
	}

	serverHTTP := &http.Server{Addr: fmt.Sprintf(":%d", cfg.Webserver.HTTPPort), Handler: r}
	serverHTTPS := &http.Server{Addr: fmt.Sprintf(":%d", cfg.Webserver.HTTPSPort), Handler: r}
	go func() {
		log.Info("starting HTTP server", "port", cfg.Webserver.HTTPPort)
		if err := serverHTTP.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(err, "unable to start HTTP server")
			os.Exit(1)
		}
	}()

	go func() {
		log.Info("starting HTTPS server", "port", cfg.Webserver.HTTPSPort)
		if err := serverHTTPS.ListenAndServeTLS(cfg.Webserver.Certificate.Cert, cfg.Webserver.Certificate.PrivateKey); err != nil && err != http.ErrServerClosed {
			log.Error(err, "unable to start HTTPS server")
		}
	}()

	UpdateHealth(true)
	<-ctx.Done()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := serverHTTP.Shutdown(ctx); err != nil {
		log.Error(err, "unable to shut down HTTP server")
	}
	if err := serverHTTPS.Shutdown(ctx); err != nil {
		log.Error(err, "unable to shut down HTTPS server")
	}
	log.Info("HTTP(S) servers stopped.")
}
