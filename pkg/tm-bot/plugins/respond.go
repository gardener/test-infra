// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package plugins

import (
	"fmt"
)

const AboutThisBotWithoutCommands = "Instructions for interacting with me using PR comments are available <a href=\"https://tm.gardener.cloud/command-help\">here</a>"

// FormatSimpleResponse formats a response that does not warrant additional explanation in the
// details section.
func FormatSimpleResponse(to, message string) string {
	format := `@%s: %s
<details>
%s
</details>`

	return fmt.Sprintf(format, to, message, AboutThisBotWithoutCommands)
}

// FormatResponseWithReason formats a response with additional explanation in the
// details section.
func FormatResponseWithReason(to, message, reason string) string {
	format := `
@%s: %s
<details>
%s

%s
</details>`

	return fmt.Sprintf(format, to, message, reason, AboutThisBotWithoutCommands)
}

// FormatSimpleErrorResponse formats a response that does not warrant additional explanation in the
// details section.
func FormatSimpleErrorResponse(to, message string) string {
	format := `:fire: Oops, something went wrong @%s
%s

>%s`

	return fmt.Sprintf(format, to, message, AboutThisBotWithoutCommands)
}

// FormatErrorResponse formats a response that does not warrant additional explanation in the
// details section.
func FormatErrorResponse(to, message, reason string) string {
	format := `:fire: Oops, something went wrong @%s
%s
<details>
%s

</details>

>%s`

	return fmt.Sprintf(format, to, message, reason, AboutThisBotWithoutCommands)
}

// FormatUsageError formats Usage of a command
func FormatUsageError(name, description, example, usage string) string {
	format := `
<pre>
Command %s
%s

Example: %s

Usage:
%s
</pre>
`
	return fmt.Sprintf(format, name, description, example, usage)
}
