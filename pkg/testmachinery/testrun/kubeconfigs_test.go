// Copyright 2021 Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package testrun

import (
	"fmt"

	"github.com/gardener/test-infra/pkg/apis/config"
	"github.com/gardener/test-infra/pkg/testmachinery"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
)

const (
	serverUrl         = "https://api.server.com"
	contextName       = "default-context"
	clusterName       = "default-cluster"
	authName          = "default-auth"
	namespace         = "default"
	testrunName       = "test-run"
	audience          = "default"
	expirationSeconds = int64(60)
)

var _ = Describe("kubeconfig tests", func() {

	Context("process kubeconfig for tokenFile", func() {
		var defaultConfig *clientcmdv1.Config

		BeforeEach(func() {
			defaultConfig = &clientcmdv1.Config{
				Clusters: []clientcmdv1.NamedCluster{
					{
						Name: clusterName,
						Cluster: clientcmdv1.Cluster{
							Server: serverUrl,
						},
					},
				},
				AuthInfos: []clientcmdv1.NamedAuthInfo{
					{
						Name:     authName,
						AuthInfo: clientcmdv1.AuthInfo{},
					},
				},
				Contexts: []clientcmdv1.NamedContext{
					{
						Name: contextName,
						Context: clientcmdv1.Context{
							Cluster:   clusterName,
							AuthInfo:  authName,
							Namespace: namespace,
						},
					},
				},
				CurrentContext: contextName,
			}

		})

		It("should return an empty map, when there is no tokenFile found", func() {
			defaultConfig.AuthInfos[0].AuthInfo.Token = "abcdefgh"
			configs := make(map[string]*clientcmdv1.Config)
			configs[gardenerKubeconfig] = defaultConfig

			projectedTokenMounts, err := processTokenFileConfigs(configs, testrunName, namespace)

			Expect(err).ToNot(HaveOccurred())
			Expect(len(projectedTokenMounts)).To(BeZero())
		})

		It("should return an error, if there is no active context in a kubeconfig", func() {
			defaultConfig.CurrentContext = ""
			configs := make(map[string]*clientcmdv1.Config)
			configs[gardenerKubeconfig] = defaultConfig

			projectedTokenMounts, err := processTokenFileConfigs(configs, testrunName, namespace)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("cannot process kubeconfig for %s due to missing currentContext field", gardenerKubeconfig)))
			Expect(len(projectedTokenMounts)).To(BeZero())
		})

		It("should return an error, when there is a tokenFile but no mapping", func() {
			defaultConfig.AuthInfos[0].AuthInfo.TokenFile = "/path/to/token"
			configs := make(map[string]*clientcmdv1.Config)
			configs[gardenerKubeconfig] = defaultConfig

			projectedTokenMounts, err := processTokenFileConfigs(configs, testrunName, namespace)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("testrun wants to use a tokenFile for kubeconfig %s, but no matching landsacpeMapping was found", gardenerKubeconfig)))
			Expect(len(projectedTokenMounts)).To(BeZero())

		})

		It("should return an error, when there is a tokenFile but a mismatch with the API server URL", func() {
			err := testmachinery.Setup(&config.Configuration{
				TestMachinery: config.TestMachinery{
					LandscapeMappings: []config.LandscapeMapping{
						{
							Namespace:           namespace,
							ApiServerUrl:        "https://wrong.api.server.com",
							Audience:            audience,
							ExpirationSeconds:   expirationSeconds,
							AllowUntrustedUsage: false,
						},
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			defaultConfig.AuthInfos[0].AuthInfo.TokenFile = "/path/to/token"
			configs := make(map[string]*clientcmdv1.Config)
			configs[gardenerKubeconfig] = defaultConfig

			projectedTokenMounts, err := processTokenFileConfigs(configs, testrunName, namespace)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("testrun wants to use a tokenFile for kubeconfig %s, but no matching landsacpeMapping was found", gardenerKubeconfig)))
			Expect(len(projectedTokenMounts)).To(BeZero())

		})

		It("should return an error, when there is a tokenFile but a mismatch with the namespace", func() {
			err := testmachinery.Setup(&config.Configuration{
				TestMachinery: config.TestMachinery{
					LandscapeMappings: []config.LandscapeMapping{
						{
							Namespace:           "wrong",
							ApiServerUrl:        serverUrl,
							Audience:            audience,
							ExpirationSeconds:   expirationSeconds,
							AllowUntrustedUsage: false,
						},
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			defaultConfig.AuthInfos[0].AuthInfo.TokenFile = "/path/to/token"
			configs := make(map[string]*clientcmdv1.Config)
			configs[gardenerKubeconfig] = defaultConfig

			projectedTokenMounts, err := processTokenFileConfigs(configs, testrunName, namespace)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("testrun wants to use a tokenFile for kubeconfig %s, but no matching landsacpeMapping was found", gardenerKubeconfig)))
			Expect(len(projectedTokenMounts)).To(BeZero())

		})

		It("should return an error, when there is a conflict in the tokenFile path", func() {
			err := testmachinery.Setup(&config.Configuration{
				TestMachinery: config.TestMachinery{
					LandscapeMappings: []config.LandscapeMapping{
						{
							Namespace:           namespace,
							ApiServerUrl:        serverUrl,
							Audience:            audience,
							ExpirationSeconds:   expirationSeconds,
							AllowUntrustedUsage: false,
						},
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			defaultConfig.AuthInfos[0].AuthInfo.TokenFile = "/path/to/token"
			configs := make(map[string]*clientcmdv1.Config)
			configs[gardenerKubeconfig] = defaultConfig

			addConfig := defaultConfig.DeepCopy()
			addConfig.AuthInfos[0].AuthInfo.TokenFile = "/path/to/token"
			configs[seedKubeconfig] = addConfig

			projectedTokenMounts, err := processTokenFileConfigs(configs, testrunName, namespace)

			Expect(err).To(HaveOccurred())
			//Expect(err.Error()).To(Equal(fmt.Sprintf("kubeconfigs for %s and %s both point to the exact same tokenFile location. Use a unique location per kubeconfig", gardenerKubeconfig, seedKubeconfig)))
			Expect(len(projectedTokenMounts)).To(BeZero())
		})

		It("should return an error, when a shoot kubeconfig has a tokenFile but untrusted usage is not allowed", func() {
			err := testmachinery.Setup(&config.Configuration{
				TestMachinery: config.TestMachinery{
					LandscapeMappings: []config.LandscapeMapping{
						{
							Namespace:           namespace,
							ApiServerUrl:        serverUrl,
							Audience:            audience,
							ExpirationSeconds:   expirationSeconds,
							AllowUntrustedUsage: false,
						},
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			defaultConfig.AuthInfos[0].AuthInfo.TokenFile = "/path/to/token"
			configs := make(map[string]*clientcmdv1.Config)
			configs[shootKubeconfig] = defaultConfig

			projectedTokenMounts, err := processTokenFileConfigs(configs, testrunName, namespace)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("untrusted usage of tokenFile for kubeconfig %s is not allowed in landscapeMapping", shootKubeconfig)))
			Expect(len(projectedTokenMounts)).To(BeZero())
		})

		It("should return a valid tokenMount, when there is one kubeconfig with a tokenFile", func() {
			err := testmachinery.Setup(&config.Configuration{
				TestMachinery: config.TestMachinery{
					LandscapeMappings: []config.LandscapeMapping{
						{
							Namespace:           namespace,
							ApiServerUrl:        serverUrl,
							Audience:            audience,
							ExpirationSeconds:   expirationSeconds,
							AllowUntrustedUsage: false,
						},
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			defaultConfig.AuthInfos[0].AuthInfo.TokenFile = "/path/to/token"
			configs := make(map[string]*clientcmdv1.Config)
			configs[gardenerKubeconfig] = defaultConfig

			projectedTokenMounts, err := processTokenFileConfigs(configs, testrunName, namespace)

			Expect(err).ToNot(HaveOccurred())
			Expect(len(projectedTokenMounts)).To(Equal(1))
		})

		It("should return a valid tokenMount for each kubeconfig with a tokenFile", func() {
			err := testmachinery.Setup(&config.Configuration{
				TestMachinery: config.TestMachinery{
					LandscapeMappings: []config.LandscapeMapping{
						{
							Namespace:           namespace,
							ApiServerUrl:        serverUrl,
							Audience:            audience,
							ExpirationSeconds:   expirationSeconds,
							AllowUntrustedUsage: false,
						},
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			defaultConfig.AuthInfos[0].AuthInfo.TokenFile = "/path/to/garden-token"
			configs := make(map[string]*clientcmdv1.Config)
			configs[gardenerKubeconfig] = defaultConfig

			addConfig := defaultConfig.DeepCopy()
			addConfig.AuthInfos[0].AuthInfo.TokenFile = "/path/to/seed-token"
			configs[seedKubeconfig] = addConfig

			projectedTokenMounts, err := processTokenFileConfigs(configs, testrunName, namespace)

			Expect(err).ToNot(HaveOccurred())
			Expect(len(projectedTokenMounts)).To(Equal(2))
		})

		It("should return a valid tokenMount , when a shoot kubeconfig has a tokenFile and untrusted usage is allowed", func() {
			err := testmachinery.Setup(&config.Configuration{
				TestMachinery: config.TestMachinery{
					LandscapeMappings: []config.LandscapeMapping{
						{
							Namespace:           namespace,
							ApiServerUrl:        serverUrl,
							Audience:            audience,
							ExpirationSeconds:   expirationSeconds,
							AllowUntrustedUsage: true,
						},
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			defaultConfig.AuthInfos[0].AuthInfo.TokenFile = "/path/to/token"
			configs := make(map[string]*clientcmdv1.Config)
			configs[shootKubeconfig] = defaultConfig

			projectedTokenMounts, err := processTokenFileConfigs(configs, testrunName, namespace)

			Expect(err).ToNot(HaveOccurred())
			Expect(len(projectedTokenMounts)).To(Equal(1))
		})
	})
})
