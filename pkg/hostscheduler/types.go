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

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

type Provider string

const (
	ProviderUnknown Provider = "Unknown"
)

// Interface is the hostscheduler interface.
// Hostscheduler functions are designed to register their function scoped flags
// and return a SchedulerFunc that is executed with corresponding subcommand.
type Interface interface {
	Lock(*flag.FlagSet) (SchedulerFunc, error)
	Release(*flag.FlagSet) (SchedulerFunc, error)
	Cleanup(*flag.FlagSet) (SchedulerFunc, error)

	List(*flag.FlagSet) (SchedulerFunc, error)
	Recreate(*flag.FlagSet) (SchedulerFunc, error)
}

// SchedulerFunc is the default hostscheduler functions which is called by the framework
// in the corresponding cobra subcommand.
type SchedulerFunc = func(ctx context.Context) error

// Registration represents the registration with metadata of a hostscheduler
type Registration interface {
	Name() Provider
	Description() string
	PreRun(cmd *cobra.Command, args []string) error
	RegisterFlags(flagset *flag.FlagSet)

	Interface() Interface
}

// Registrations holds a map of hostscheduler names and their corresponding Registration implementation.
type Registrations struct {
	list []Registration
}

// Register should be implemented by the hostscheduler and add themselves to the registrations.
type Register = func(registration *Registrations)
