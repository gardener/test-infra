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
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gardener/test-infra/pkg/server/webhooks"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	log "github.com/sirupsen/logrus"
)

// Serve starts the webhook server for testrun validation
func Serve(ctx context.Context, mgr manager.Manager) {

	listenAddressHTTP := fmt.Sprintf("%s:%s", os.Getenv("WEBHOOK_HTTP_BINDADDRESS"), os.Getenv("WEBHOOK_HTTP_PORT"))
	listenAddressHTTPS := fmt.Sprintf("%s:%s", os.Getenv("WEBHOOK_HTTPS_BINDADDRESS"), os.Getenv("WEBHOOK_HTTPS_PORT"))

	serverCertFile := os.Getenv("WEBHOOK_CERT_FILE")
	serverKeyFile := os.Getenv("WEBHOOK_KEY_FILE")

	serverMuxHTTP := http.NewServeMux()
	serverMuxHTTPS := http.NewServeMux()

	serverHTTP := &http.Server{Addr: listenAddressHTTP, Handler: serverMuxHTTP}
	serverHTTPS := &http.Server{Addr: listenAddressHTTPS, Handler: serverMuxHTTPS}

	serverMuxHTTP.HandleFunc("/healthz", healthz)

	go func() {
		log.Infof("Starting HTTP server on %s", listenAddressHTTP)
		if err := serverHTTP.ListenAndServe(); err != http.ErrServerClosed {
			log.Errorf("Could not start HTTP server: %s", err.Error())
		}
	}()

	trWebhook := webhooks.NewTestrunWebhook(mgr.GetAdmissionDecoder())

	serverMuxHTTPS.HandleFunc("/webhooks/validate-testrun", trWebhook.Validate)

	go func() {
		log.Infof("Starting HTTPS server on %s", listenAddressHTTPS)
		if err := serverHTTPS.ListenAndServeTLS(serverCertFile, serverKeyFile); err != http.ErrServerClosed {
			log.Errorf("Could not start HTTPS server: %s", err.Error())
		}
	}()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := serverHTTP.Shutdown(ctx); err != nil {
		log.Errorf("Error when shutting down HTTP server: %v", err)
	}
	if err := serverHTTPS.Shutdown(ctx); err != nil {
		log.Errorf("Error when shutting down HTTPS server: %v", err)
	}
	log.Info("HTTP(S) servers stopped.")

}
