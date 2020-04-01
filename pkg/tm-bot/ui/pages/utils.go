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

package pages

import (
	"fmt"
	"net/url"
)

func makeBaseTemplateSettings(global globalSettings) func(string, interface{}) baseTemplateSettings {
	return func(pageName string, arguments interface{}) baseTemplateSettings {
		return baseTemplateSettings{
			globalSettings: global,
			PageName:       pageName,
			Arguments:      arguments,
		}
	}
}

// addURLParams adds all parameters defined by "key" "value" to the url url
func addURLParams(baseUrl string, keysAndValues ...interface{}) string {
	u, err := url.Parse(baseUrl)
	if err != nil {
		panic(fmt.Sprintf("unable to parse url %s: %v", baseUrl, err))
	}

	if (len(keysAndValues) % 2) != 0 {
		panic("odd number of key value pairs")
	}

	q := u.Query()
	for i := 0; i < len(keysAndValues); i = i + 2 {
		key := fmt.Sprintf("%v", keysAndValues[i])
		value := fmt.Sprintf("%v", keysAndValues[i+1])

		q.Set(key, value)
	}

	u.RawQuery = q.Encode()
	return u.String()
}
