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
	"flag"
	"github.com/sirupsen/logrus"
)

type Interface interface {
	Lock(context.Context) error
	Release(context.Context) error
	Cleanup(context.Context) error
}

// RegisterInterfaceFromArgsFunc initializes a new hostscheduler interface from given arguments
type RegisterInterfaceFromArgsFunc = func(ctx context.Context, logger *logrus.Logger) (Interface, error)

// RegisterFlagsFunc adds necessary flags from a specific hostscheduler to the overall command.
// In the future we may also have to have different flags for lock and release.
type RegisterFlagsFunc = func(flagset *flag.FlagSet)

type Registration struct {
	Interface RegisterInterfaceFromArgsFunc
	Flags     RegisterFlagsFunc
}

// Registrations holds a map of hostscheduler names and their corresponding Registration implementation.
type Registrations map[string]*Registration

// Register should be implemented by the hostscheduler and add themselves to the registrations.
type Register = func(registration Registrations)
