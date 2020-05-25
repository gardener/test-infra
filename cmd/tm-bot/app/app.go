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

package app

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	tm_bot "github.com/gardener/test-infra/pkg/tm-bot"
	"github.com/gardener/test-infra/pkg/version"
)

func NewTestMachineryBotCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:   "testmachinery-bot",
		Short: "TestMachinery bot hosts a github bot to interact with github and start tests and hosts the TestMachinery Dashbaord",

		Run: func(cmd *cobra.Command, args []string) {
			if err := options.Complete(); err != nil {
				fmt.Print(err)
				os.Exit(1)
			}
			options.run(ctx)
		},
	}

	options.AddFlags(cmd.Flags())

	return cmd
}

func (o *options) run(ctx context.Context) {
	o.log.Info(fmt.Sprintf("start Test Machinery Bot with version %s", version.Get().String()))

	tm_bot.Serve(ctx, o.log, o.restConfig, o.config)
}
