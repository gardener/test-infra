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
	"net/http"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"k8s.io/client-go/rest"

	"github.com/gardener/test-infra/pkg/apis/config"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/watch"
	"github.com/gardener/test-infra/pkg/tm-bot/tests"
)

// Serve starts the webhook server for testrun validation
func Serve(ctx context.Context, log logr.Logger, restConfig *rest.Config, cfg *config.BotConfiguration) error {
	o := NewOptions(log, restConfig, cfg)
	log.Info("Start TM Bot")

	var (
		err        error
		syncPeriod = 10 * time.Minute
	)
	o.w, err = watch.New(o.log, o.restConfig, &watch.Options{
		SyncPeriod: &syncPeriod,
	})
	if err != nil {
		return err
	}

	go func() {
		if err := o.w.Start(ctx); err != nil {
			o.log.Error(err, "error while starting watch")
		}
	}()

	if err := watch.WaitForCacheSyncWithTimeout(o.w, 2*time.Minute); err != nil {
		return err
	}

	runs := tests.NewRuns(o.w)

	r := mux.NewRouter()
	r.Use(loggingMiddleware(o.log.WithName("trace")))
	r.HandleFunc("/healthz", healthz(o.log.WithName("health"))).Methods(http.MethodGet)

	if err := o.setupGitHubBot(r, runs); err != nil {
		return err
	}

	if err := o.setupDashboard(r, runs); err != nil {
		return err
	}

	return o.startWebserver(r, ctx.Done())
}

func (o *options) startWebserver(router *mux.Router, stop <-chan struct{}) error {
	cfg := o.cfg.Webserver
	serverHTTP := &http.Server{Addr: fmt.Sprintf(":%d", cfg.HTTPPort), Handler: router}
	serverHTTPS := &http.Server{Addr: fmt.Sprintf(":%d", cfg.HTTPSPort), Handler: router}
	go func() {
		o.log.Info("starting HTTP server", "port", cfg.HTTPPort)
		if err := serverHTTP.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			o.log.Error(err, "unable to start HTTP server")
			os.Exit(1)
		}
	}()

	go func() {
		o.log.Info("starting HTTPS server", "port", cfg.HTTPSPort)
		if err := serverHTTPS.ListenAndServeTLS(o.cfg.Webserver.Certificate.Cert, cfg.Certificate.PrivateKey); err != nil && err != http.ErrServerClosed {
			o.log.Error(err, "unable to start HTTPS server")
		}
	}()

	UpdateHealth(true)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := serverHTTP.Shutdown(ctx); err != nil {
		o.log.Error(err, "unable to shut down HTTP server")
	}
	if err := serverHTTPS.Shutdown(ctx); err != nil {
		o.log.Error(err, "unable to shut down HTTPS server")
	}
	o.log.Info("HTTP(S) servers stopped.")
	return nil
}
