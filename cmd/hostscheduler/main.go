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

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/gardener/test-infra/pkg/hostscheduler"
	"github.com/gardener/test-infra/pkg/hostscheduler/gardenerscheduler"
	"github.com/gardener/test-infra/pkg/hostscheduler/gkescheduler"
	"os"

	log "github.com/sirupsen/logrus"
)

var (
	registration    hostscheduler.Registrations
	schedulerLogger *log.Logger

	flagset *flag.FlagSet
	clean   bool
	debug   bool
)

const (
	cmdLock    = "lock"
	cmdRelease = "release"
)

func init() {
	// register hostscheduler provider
	registration = make(hostscheduler.Registrations)
	gkescheduler.Register(registration)
	gardenerscheduler.Register(registration)

	flagset = flag.NewFlagSet("all", flag.ContinueOnError)
	flagset.BoolVar(&clean, "clean", false, "cleanup cluster")
	flagset.BoolVar(&debug, "debug", false, "debug output")
}

func main() {
	ctx := context.Background()
	defer ctx.Done()

	if len(os.Args) < 3 {
		schedulerLogger.Fatal("list or count subcommand is required")
	}

	cmd := os.Args[1]
	schedulerName := os.Args[2]
	args := os.Args[3:]

	if cmd != cmdLock && cmd != cmdRelease {
		fmt.Printf("%s is not a valid command. Allowed commands: 'lock', 'release'", cmd)
		os.Exit(1)
	}

	// Add the flags correspnding to the choosing hostscheduler to the flagset of the command
	if err := registration.ApplyFlags(schedulerName, flagset); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// parse the arguments and initialize common standard objects like logging
	if err := parseAndSetupStdParameters(args); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Initialize the requested hostscheduler
	hostScheduler, err := registration.GetInterface(schedulerName, ctx, schedulerLogger, flagset, args)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	switch cmd {
	case cmdLock:
		lockCmd(ctx, hostScheduler)
	case cmdRelease:
		releaseCmd(ctx, hostScheduler, clean)
	default:
		fmt.Printf("%s is not a valid command. Allowed commands: 'lock', 'release'", cmd)
	}
}

func lockCmd(ctx context.Context, hostScheduler hostscheduler.Interface) {
	err := hostScheduler.Lock(ctx)
	if err != nil {
		schedulerLogger.Fatal(err)
	}
	schedulerLogger.Infof("Cluster successfully locked and ready")
}

func releaseCmd(ctx context.Context, hostScheduler hostscheduler.Interface, clean bool) {
	if clean {
		err := hostScheduler.Cleanup(ctx)
		if err != nil {
			schedulerLogger.Fatal(err)
		}
	}

	err := hostScheduler.Release(ctx)
	if err != nil {
		schedulerLogger.Fatal(err)
	}
	schedulerLogger.Infof("Successfully released cluster")
}

func parseAndSetupStdParameters(args []string) error {
	err := flagset.Parse(args)
	if err != nil {
		return fmt.Errorf("flags cannot be parsed: %s", err.Error())
	}
	schedulerLogger = log.StandardLogger()
	formatter := &log.TextFormatter{
		FullTimestamp: true,
		DisableColors: true,
	}
	schedulerLogger.SetFormatter(formatter)
	schedulerLogger.SetOutput(os.Stderr)
	if debug {
		schedulerLogger.SetLevel(log.DebugLevel)
		schedulerLogger.Warn("Set debug log level")
	}
	return nil
}
