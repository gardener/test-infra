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
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/gardener/test-infra/pkg/tm-bot/hook"
	"net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	flag "github.com/spf13/pflag"
)

var (
	listenAddressHTTP  string
	listenAddressHTTPS string

	serverCertFile string
	serverKeyFile  string

	githubAppID        int
	githubKeyFile      string
	webhookSecretToken string
	repoConfigFile     string

	kubeconfigPath string
)

// Serve starts the webhook server for testrun validation
func Serve(ctx context.Context, log logr.Logger) {

	k8sClient, err := kubernetes.NewClientFromFile("", kubeconfigPath, kubernetes.WithClientOptions(client.Options{
		Scheme: testmachinery.TestMachineryScheme,
	}))
	if err != nil {
		log.Error(err, "unable to initialize kubernetes client")
		os.Exit(1)
	}

	ghClient, err := github.NewManager(log.WithName("github"), githubAppID, githubKeyFile, repoConfigFile)
	if err != nil {
		log.Error(err, "unable to initialize github client")
		os.Exit(1)
	}
	hooks := hook.New(log.WithName("hooks"), ghClient, webhookSecretToken, k8sClient)

	serverMuxHTTP := http.NewServeMux()
	serverMuxHTTPS := http.NewServeMux()
	serverHTTP := &http.Server{Addr: listenAddressHTTP, Handler: serverMuxHTTP}
	serverHTTPS := &http.Server{Addr: listenAddressHTTPS, Handler: serverMuxHTTPS}
	r := mux.NewRouter()

	r.HandleFunc("/healthz", healthz(log.WithName("health"))).Methods(http.MethodGet)
	r.HandleFunc("/event/handler", hooks.HandleWebhook).Methods(http.MethodPost)

	serverMuxHTTP.Handle("/", r)
	serverMuxHTTPS.Handle("/", r)

	go func() {
		log.Info("starting HTTP server", "port", listenAddressHTTP)
		if err := serverHTTP.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(err, "unable to start HTTP server")
		}
	}()

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

	flagset.IntVar(&githubAppID, "github-app-id", 0, "GitHub app installation id")
	flagset.StringVar(&githubKeyFile, "github-key-file", "", "GitHub app private key file path")
	flagset.StringVar(&webhookSecretToken, "webhook-secret-token", "testing", "GitHub webhook secret to verify payload")
	flagset.StringVar(&repoConfigFile, "config-file-path", ".ci/tm-bot", "Path the bot configuration in the repository")

	flagset.StringVar(&kubeconfigPath, "kubeconfig", os.Getenv("KUBECONFIG"), "Kubeconfig path to a testmachinery cluster")
}
