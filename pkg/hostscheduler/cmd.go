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

package hostscheduler

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
)

var lockCmd = cobra.Command{
	Use:   "lock",
	Short: "Select an available host and lock it",
}

var releaseCmd = cobra.Command{
	Use:   "release",
	Short: "Free a locked cluster",
}

var cleanupCmd = cobra.Command{
	Use:     "cleanup",
	Aliases: []string{"clean"},
	Short:   "Clean a kubernetes cluster",
	Long: `Cleans a kubernetes cluster by
- deleting all crds
- deleting all webhooks
- deleting all default resources (deployments, statefulsets, pods, namspaces, etc.)

NOTE: Secrets and configmaps are not cleanup explicitly
			`,
}

var listCmd = cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List host cluster",
}

var recreateCmd = cobra.Command{
	Use:     "recreate",
	Aliases: []string{"rc"},
	Short:   "Recreate a given host cluster",
	Long:    "Deletes the old cluster if it is not currently used and recreate it with the same spec",
	Example: "recreate --name [cluster identifier]",
}

func copyCommand(cmd cobra.Command) *cobra.Command {
	newCommand := cmd
	return &newCommand
}

// CommandFromRegistration generates the command of a scheduler registration with all its subcommands.
func CommandFromRegistration(r Registration) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   string(r.Name()),
		Short: r.Description(),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if cmd.Parent() != nil && cmd.Parent().PersistentPreRun != nil {
				cmd.Parent().PersistentPreRun(cmd.Parent(), args)
			}
			if err := r.PreRun(cmd, args); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		},
		RunE: nil,
	}

	r.RegisterFlags(cmd.PersistentFlags())
	return cmd, nil
}

// AddSchedulerCommandsFromScheduler genereates and adds all interface functions of the scheduler interface
// as cobra subcommands to the provided rootCmd.
func AddSchedulerCommandsFromScheduler(rootCmd *cobra.Command, scheduler Interface) error {
	lockCmd := copyCommand(lockCmd)
	lockFunc, err := scheduler.Lock(lockCmd.Flags())
	if err != nil {
		return errors.Wrap(err, "unable to register lock")
	}
	lockCmd.Run = func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		defer ctx.Done()
		if err := lockFunc(ctx); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	releaseCmd := copyCommand(releaseCmd)
	releaseFunc, err := scheduler.Release(releaseCmd.Flags())
	if err != nil {
		return errors.Wrap(err, "unable to register release cmd")
	}
	releaseCleanupFunc, err := scheduler.Cleanup(releaseCmd.Flags())
	if err != nil {
		return errors.Wrap(err, "unable to register cleanup cmd")
	}
	releaseCmd.Run = func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		defer ctx.Done()

		err := releaseCleanupFunc(ctx)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if err := releaseFunc(ctx); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	cleanupCmd := copyCommand(cleanupCmd)
	cleanupFunc, err := scheduler.Cleanup(cleanupCmd.Flags())
	if err != nil {
		return errors.Wrap(err, "unable to register cleanup cmd")
	}
	cleanupCmd.Run = func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		defer ctx.Done()
		if err := cleanupFunc(ctx); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	listCmd := copyCommand(listCmd)
	listFunc, err := scheduler.List(listCmd.Flags())
	if err != nil {
		return errors.Wrap(err, "unable to register lock")
	}
	listCmd.Run = func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		defer ctx.Done()
		if err := listFunc(ctx); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	recreateCmd := copyCommand(recreateCmd)
	recreateFunc, err := scheduler.Recreate(recreateCmd.Flags())
	if err != nil {
		return errors.Wrap(err, "unable to register recreate")
	}
	recreateCmd.Run = func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		defer ctx.Done()
		if err := recreateFunc(ctx); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	rootCmd.AddCommand(lockCmd, releaseCmd, cleanupCmd, listCmd, recreateCmd)
	return nil
}
