// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
	if err := tm_bot.Serve(ctx, o.log, o.restConfig, o.config); err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
}
