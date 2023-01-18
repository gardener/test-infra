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

package plugins_test

import (
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"

	"github.com/gardener/test-infra/pkg/tm-bot/github"
	mock_github "github.com/gardener/test-infra/pkg/tm-bot/github/mocks"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins"
	mock_plugins "github.com/gardener/test-infra/pkg/tm-bot/plugins/mocks"
)

var _ = Describe("plugins", func() {
	var (
		mockPersistence *mock_plugins.MockPersistence
		mockPlugin      *mock_plugins.MockPlugin
		mockGHMgr       *mock_github.MockManager
		mockGHClient    *mock_github.MockClient

		plCtrl    *gomock.Controller
		persCtrl  *gomock.Controller
		ghMgrCtrl *gomock.Controller
		ghCtrl    *gomock.Controller
	)
	BeforeEach(func() {
		persCtrl = gomock.NewController(GinkgoT())
		plCtrl = gomock.NewController(GinkgoT())
		ghMgrCtrl = gomock.NewController(GinkgoT())
		ghCtrl = gomock.NewController(GinkgoT())
	})
	AfterEach(func() {
		persCtrl.Finish()
		plCtrl.Finish()
		ghMgrCtrl.Finish()
		ghCtrl.Finish()
	})

	Context("Register", func() {
		BeforeEach(func() {
			mockPersistence = mock_plugins.NewMockPersistence(persCtrl)
			mockPlugin = mock_plugins.NewMockPlugin(plCtrl)
			mockGHMgr = mock_github.NewMockManager(ghMgrCtrl)
			mockGHClient = mock_github.NewMockClient(ghCtrl)
		})
		It("should register a plugin and retrieve it", func() {
			mockPlugin.EXPECT().Command().Return("test").AnyTimes()
			mockPlugin.EXPECT().New(gomock.Any()).Return(mockPlugin).Times(1)
			p := plugins.New(logr.Discard(), mockPersistence)

			p.Register(mockPlugin)

			_, pl, err := p.Get("test")
			Expect(err).ToNot(HaveOccurred())
			Expect(pl).To(Equal(mockPlugin))
		})

		It("should throw an error if a plugin is not defined", func() {
			p := plugins.New(logr.Discard(), mockPersistence)
			_, _, err := p.Get("test")
			Expect(err).To(HaveOccurred())
		})

		It("should call a registered plugin", func() {
			event := &github.GenericRequestEvent{
				Repository: nil,
				Body:       "/test",
				Author:     nil,
			}
			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

			mockGHClient.EXPECT().IsAuthorized(github.AuthorizationAll, event).Return(true).AnyTimes()

			mockPlugin.EXPECT().Command().Return("test").AnyTimes()
			mockPlugin.EXPECT().New(gomock.Any()).Return(mockPlugin).Times(1)
			mockPlugin.EXPECT().Flags().Return(fs).Times(1)
			mockPlugin.EXPECT().Authorization().Return(github.AuthorizationAll).Times(1)
			mockPlugin.EXPECT().Run(gomock.Any(), mockGHClient, event).Return(nil).Times(1)

			p := plugins.New(logr.Discard(), mockPersistence)
			p.Register(mockPlugin)

			err := p.HandleRequest(mockGHClient, event)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should resume a running plugin", func() {
			var saved bool
			event := &github.GenericRequestEvent{
				Repository: nil,
				Body:       "/test",
				Author:     nil,
			}
			state := map[string]map[string]*plugins.State{
				"test": {
					"abc": &plugins.State{
						Event:  event,
						Custom: "",
					},
				},
			}

			mockPlugin.EXPECT().Command().Return("test").AnyTimes()
			mockPlugin.EXPECT().New(gomock.Any()).Return(mockPlugin).Times(2)
			mockPlugin.EXPECT().ResumeFromState(mockGHClient, event, "").Return(nil).Times(1)

			mockPersistence.EXPECT().Load().Return(state, nil).Times(1)
			mockPersistence.EXPECT().Save(gomock.Any()).DoAndReturn(func(_ interface{}) error {
				saved = true
				return nil
			})

			mockGHMgr.EXPECT().GetClient(event).Return(mockGHClient, nil).Times(1)

			p := plugins.New(logr.Discard(), mockPersistence)
			p.Register(mockPlugin)

			err := p.ResumePlugins(mockGHMgr)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				return saved
			}).Should(BeTrue())
		})
	})

})
