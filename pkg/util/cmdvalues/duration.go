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
	"time"

	"github.com/spf13/pflag"
)

type DurationValue struct {
	duration *time.Duration
}

func NewDurationValue(value *time.Duration, defaultValue time.Duration) pflag.Value {
	*value = defaultValue
	return &DurationValue{duration: value}
}

func (v *DurationValue) String() string {
	return v.duration.String()
}

func (v *DurationValue) Type() string {
	return "duration"
}

func (v *DurationValue) Set(value string) error {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return err
	}
	*v.duration = duration
	return nil
}
