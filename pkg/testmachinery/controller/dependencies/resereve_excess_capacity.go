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

package dependencies

import (
	"context"
	"fmt"
	"github.com/gardener/gardener/pkg/utils/chart"
	"github.com/gardener/test-infra/pkg/testmachinery/imagevector"

	intconfig "github.com/gardener/test-infra/pkg/apis/config"
)

func (e *DependencyEnsurer) ensureReserveExcessCapacityPods(ctx context.Context, namespace string, config *intconfig.ReservedExcessCapacity) error {
	e.log.Info("Ensuring reserve excess capacity pods")

	if config == nil {
		e.log.Info("Reserve excess capacity pods were not deployed as no configuration was provided")
		return nil
	}

	values := map[string]interface{}{
		"replicas":  config.Replicas,
		"resources": config.Resources,
	}

	values, err := chart.InjectImages(values, imagevector.ImageVector(), []string{
		intconfig.ReserveExcessCapacityImageName,
	})
	if err != nil {
		return fmt.Errorf("failed to find image version %v", err)
	}

	err = e.createManagedResource(ctx, namespace, intconfig.ReserveExcessCapacityManagedResourceName, e.renderer,
		intconfig.ReserveExcessCapacityChartName, values, nil)
	if err != nil {
		e.log.Error(err, "unable to create managed resource")
		return err
	}
	return nil
}
