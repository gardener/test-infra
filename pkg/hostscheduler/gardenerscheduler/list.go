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

package gardenerscheduler

import (
	"context"
	"fmt"
	"os"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"

	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/pkg/hostscheduler"
	"github.com/gardener/test-infra/pkg/util/cmdutil"
)

func (s *gardenerscheduler) List(_ *flag.FlagSet) (hostscheduler.SchedulerFunc, error) {
	return func(ctx context.Context) error {

		shoots := &gardencorev1beta1.ShootList{}
		selector := labels.SelectorFromSet(map[string]string{
			ShootLabel: "true",
		})
		err := s.client.List(ctx, shoots, &client.ListOptions{
			LabelSelector: selector,
			Namespace:     s.namespace,
		})
		if err != nil {
			return fmt.Errorf("shoots cannot be listed: %s", err.Error())
		}

		headers := []string{"NAME", "STATUS", "ID", "LOCKED", "MESSAGE"}
		content := make([][]string, 0)
		for _, shoot := range shoots.Items {
			var (
				status  = shoot.Labels[ShootLabelStatus]
				message string
			)
			if s.cloudprovider != CloudProviderAll {
				if shoot.Spec.Provider.Type != string(s.cloudprovider) {
					continue
				}
			}

			if err := shootReady(&shoot); err != nil {
				status = "notReady"
				message = err.Error()
			}

			content = append(content, []string{
				shoot.Name,
				status,
				shoot.Annotations[ShootAnnotationID],
				shoot.Annotations[ShootAnnotationLockedAt],
				message,
			})
		}

		cmdutil.PrintTable(os.Stdout, headers, content)
		return nil
	}, nil
}
