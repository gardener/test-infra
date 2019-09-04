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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Apply registers all Scheduler Registrations as cobra commands to the rootCmd.
func (r *Registrations) Apply(rootCmd *cobra.Command) error {
	for _, registration := range r.list {
		cmd, err := CommandFromRegistration(registration)
		if err != nil {
			return errors.Wrapf(err, "unable to register %s", registration.Name())
		}
		if err := AddSchedulerCommandsFromScheduler(cmd, registration.Interface()); err != nil {
			return errors.Wrapf(err, "unable to add scheduler commands to %s", registration.Name())
		}

		rootCmd.AddCommand(cmd)
	}
	return nil
}

// Add adds a scheduler registration to the registrations list
func (r *Registrations) Add(registration Registration) {
	if r.list == nil {
		r.list = make([]Registration, 0)
	}
	r.list = append(r.list, registration)
}
