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
	"context"
	"fmt"
	"github.com/gardener/test-infra/pkg/hostscheduler"
	flag "github.com/spf13/pflag"
)

func (s *gkescheduler) Recreate(flagset *flag.FlagSet) (hostscheduler.SchedulerFunc, error) {
	return func(ctx context.Context) error {
		fmt.Println("recreate for gke clusters currently not implemented")
		return nil
	}, nil
}
