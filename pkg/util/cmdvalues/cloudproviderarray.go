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
	"fmt"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/spf13/pflag"
)

type CloudProviderArrayValue struct {
	allowedProvider map[common.CloudProvider]bool
	cloudproviders  map[common.CloudProvider]bool
	value           *[]common.CloudProvider
	changed         bool
}

func NewCloudProviderArrayValue(value *[]common.CloudProvider, allowed ...common.CloudProvider) pflag.Value {
	cpvalue := &CloudProviderArrayValue{
		allowedProvider: make(map[common.CloudProvider]bool),
		cloudproviders:  make(map[common.CloudProvider]bool),
		value:           value,
		changed:         false,
	}
	for _, cp := range allowed {
		cpvalue.allowedProvider[cp] = true
	}
	return cpvalue
}

func (v *CloudProviderArrayValue) String() string {
	// won't be implemented as we cannot user strings.Join() and
	// would need to construct the csv of casted cloudproviders ourselves
	return ""
}

func (v *CloudProviderArrayValue) Type() string {
	return "CloudProviderArray"
}

func (v *CloudProviderArrayValue) Set(value string) error {
	provider := common.CloudProvider(value)
	if _, ok := v.allowedProvider[provider]; len(v.allowedProvider) != 0 && !ok {
		return fmt.Errorf("unsupported cloudprovider %s", provider)
	}

	if _, ok := v.cloudproviders[provider]; ok {
		return fmt.Errorf("duplicated cloudprovider %s", provider)
	}

	if !v.changed {
		*v.value = []common.CloudProvider{provider}
		v.changed = true
	} else {
		*v.value = append(*v.value, provider)
	}

	v.cloudproviders[provider] = true
	return nil
}
