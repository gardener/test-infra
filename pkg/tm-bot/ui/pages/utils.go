// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
