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

package gkescheduler

import (
	"cloud.google.com/go/container/apiv1"
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/gardener/test-infra/pkg/hostscheduler"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

const (
	Name = "gke"
)

var (
	gcloudKeyfilePath string
	project           string
	zone              string

	ID string
)

var Register hostscheduler.Register = func(m hostscheduler.Registrations) {
	if m == nil {
		m = make(hostscheduler.Registrations)
	}
	m[Name] = &hostscheduler.Registration{
		Interface: registerScheduler,
		Flags:     registerFlags,
	}
}

var registerFlags hostscheduler.RegisterFlagsFunc = func(fs *flag.FlagSet) {
	fs.StringVar(&gcloudKeyfilePath, "key", "", "Path to the gardener cluster gcloudKeyfilePath")
	fs.StringVar(&project, "project", "", "gcp project name")
	fs.StringVar(&zone, "zone", "", "gcp zone name")
	fs.StringVar(&ID, "id", "", "unique id to identify the cluster")
}

var registerScheduler hostscheduler.RegisterInterfaceFromArgsFunc = func(ctx context.Context, logger *logrus.Logger) (hostscheduler.Interface, error) {
	if gcloudKeyfilePath == "" {
		return nil, errors.New("no gcloud keyfile is specified")
	}
	if project == "" {
		return nil, errors.New("no project is specified")
	}
	if zone == "" {
		return nil, errors.New("no zone is specified")
	}

	logger.Debugf("GCloud Secret Path: %s", gcloudKeyfilePath)
	logger.Debugf("Project: %s", project)
	logger.Debugf("Zone: %s", zone)
	logger.Debugf("ID: %s", ID)

	return New(ctx, logger, project, zone, gcloudKeyfilePath, ID)
}

func New(ctx context.Context, logger *logrus.Logger, project, zone string, gcloudKeyfilePath, id string) (hostscheduler.Interface, error) {
	c, err := container.NewClusterManagerClient(ctx, option.WithCredentialsFile(gcloudKeyfilePath))
	if err != nil {
		return nil, err
	}
	return &gkescheduler{client: c, logger: logger, id: id, project: project, zone: zone}, nil
}

func (s gkescheduler) getParentName() string {
	return fmt.Sprintf("projects/%s/locations/%s", s.project, s.zone)
}

func (s gkescheduler) getClusterName(name string) string {
	return fmt.Sprintf("%s/clusters/%s", s.getParentName(), name)
}

func (s gkescheduler) getNodePoolName(clusterName, nodePoolName string) string {
	return fmt.Sprintf("%s/nodePools/%s", s.getClusterName(clusterName), nodePoolName)
}

func (s gkescheduler) getOperationName(name string) string {
	return fmt.Sprintf("%s/operations/%s", s.getParentName(), name)
}

var _ hostscheduler.Interface = &gkescheduler{}
