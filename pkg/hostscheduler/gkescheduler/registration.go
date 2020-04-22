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

package gkescheduler

import (
	"cloud.google.com/go/container/apiv1"
	"context"
	"fmt"
	"github.com/gardener/test-infra/pkg/hostscheduler"
	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/util/cmdutil/viper"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"google.golang.org/api/option"
	"os"
)

const (
	Name hostscheduler.Provider = "gke"
)

var Register hostscheduler.Register = func(m *hostscheduler.Registrations) {
	m.Add(&registration{
		scheduler: &gkescheduler{},
	})
}

func (r *registration) Name() hostscheduler.Provider {
	return Name
}
func (r *registration) Description() string {
	return ""
}
func (r *registration) Interface() hostscheduler.Interface {
	return r.scheduler
}
func (r *registration) RegisterFlags(flagset *flag.FlagSet) {
	if flagset == nil {
		flagset = flag.CommandLine
	}

	flagset.StringVar(&r.gcloudkeyFile, "key", "", "Path to the gardener cluster gcloudKeyfilePath")
	flagset.StringVar(&r.scheduler.hostname, "name", "", "Name of the target gke cluster. Optional")
	flagset.StringVar(&r.scheduler.project, "project", "", "gcp project name")
	flagset.StringVar(&r.scheduler.zone, "zone", "", "gcp zone name")

	viper.BindPFlagFromFlagSet(flagset, "key", "gke.gcloudKeyPath")
	viper.BindPFlagFromFlagSet(flagset, "project", "gke.project")
	viper.BindPFlagFromFlagSet(flagset, "zone", "gke.zone")
}

func (r *registration) PreRun(cmd *cobra.Command, args []string) error {
	r.scheduler.log = logger.Log.WithName(string(r.Name()))

	if r.gcloudkeyFile == "" {
		return errors.New("No gcloud key file defined")
	}
	if _, err := os.Stat(r.gcloudkeyFile); err != nil {
		return fmt.Errorf("GCloud json at %s cannot be found", r.gcloudkeyFile)
	}

	c, err := container.NewClusterManagerClient(context.TODO(), option.WithCredentialsFile(r.gcloudkeyFile))
	if err != nil {
		return err
	}
	r.scheduler.client = c
	return nil
}
