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

package cmdvalues

import (
	"github.com/gardener/test-infra/pkg/hostscheduler"
	"github.com/gardener/test-infra/pkg/hostscheduler/gardenerscheduler"
	"github.com/spf13/pflag"
)

type HostProviderValue struct {
	provider *hostscheduler.Provider
}

func NewHostProviderValue(value *hostscheduler.Provider, defaultValue hostscheduler.Provider) pflag.Value {
	*value = defaultValue
	return &HostProviderValue{provider: value}
}

var _ pflag.Value = &HostProviderValue{}

func (v *HostProviderValue) String() string {
	if v.provider == nil {
		return string(gardenerscheduler.Name)
	}
	return string(*v.provider)
}

func (v *HostProviderValue) Type() string {
	return "HostProvider"
}

func (v *HostProviderValue) Set(value string) error {
	provider := hostscheduler.Provider(value)
	// add validation
	*v.provider = provider
	return nil
}
