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
	"fmt"

	"github.com/gardener/test-infra/pkg/hostscheduler"
)

func (s *gkescheduler) getParentName() string {
	return fmt.Sprintf("projects/%s/locations/%s", s.project, s.zone)
}

func (s *gkescheduler) getClusterName(name string) string {
	return fmt.Sprintf("%s/clusters/%s", s.getParentName(), name)
}

func (s *gkescheduler) getNodePoolName(clusterName, nodePoolName string) string {
	return fmt.Sprintf("%s/nodePools/%s", s.getClusterName(clusterName), nodePoolName)
}

func (s *gkescheduler) getOperationName(name string) string {
	return fmt.Sprintf("%s/operations/%s", s.getParentName(), name)
}

var _ hostscheduler.Interface = &gkescheduler{}
