// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
	serverHTTP := &http.Server{Addr: fmt.Sprintf(":%d", cfg.HTTPPort), Handler: router, ReadHeaderTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, ReadTimeout: 10 * time.Second}
	serverHTTPS := &http.Server{Addr: fmt.Sprintf(":%d", cfg.HTTPSPort), Handler: router, ReadHeaderTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, ReadTimeout: 10 * time.Second}
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
