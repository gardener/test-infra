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

package server

import (
	"context"
	"flag"
	"github.com/go-logr/logr"
	"net/http"
	"os"
	"time"

	"github.com/gardener/test-infra/pkg/controller/admission/server/webhooks"
)

var (
	listenAddressHTTP  string
	listenAddressHTTPS string

	serverCertFile string
	serverKeyFile  string
)

// Serve starts the webhook server for testrun validation
func Serve(ctx context.Context, log logr.Logger) {
	serverMuxHTTP := http.NewServeMux()
	serverMuxHTTPS := http.NewServeMux()

	serverHTTP := &http.Server{Addr: listenAddressHTTP, Handler: serverMuxHTTP}
	serverHTTPS := &http.Server{Addr: listenAddressHTTPS, Handler: serverMuxHTTPS}

	serverMuxHTTP.HandleFunc("/healthz", healthz)

	go func() {
		log.Info("starting HTTP server", "port", listenAddressHTTP)
		if err := serverHTTP.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(err, "unable to start HTTP server")
		}
	}()

	trWebhook, err := webhooks.NewTestrunWebhook(log)
	if err != nil {
		log.Error(err, "unable to create webhook for testrun")
		os.Exit(1)
	}

	serverMuxHTTPS.HandleFunc("/webhooks/validate-testrun", trWebhook.Validate)

	go func() {
		log.Info("starting HTTPS server", "port", listenAddressHTTPS)
		if err := serverHTTPS.ListenAndServeTLS(serverCertFile, serverKeyFile); err != nil && err != http.ErrServerClosed {
			log.Error(err, "unable to start HTTPS server")
		}
	}()

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

func InitFlags(flagset *flag.FlagSet) {
	if flagset == nil {
		flagset = flag.CommandLine
	}

	flagset.StringVar(&listenAddressHTTP, "webhook-http-address", ":80",
		"Webhook HTTP address to bind")
	flagset.StringVar(&listenAddressHTTPS, "webhook-https-address", ":443",
		"Webhook HTTPS address to bind")

	flagset.StringVar(&serverCertFile, "cert-file", os.Getenv("WEBHOOK_CERT_FILE"),
		"Path to server certificate")
	flagset.StringVar(&serverKeyFile, "key-file", os.Getenv("WEBHOOK_KEY_FILE"),
		"Path to private key")
}
