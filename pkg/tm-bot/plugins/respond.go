// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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

// FormatUnauthorizedResponse returns the user not authorized response
func FormatUnauthorizedResponse(to, name string) string {
	format := ":construction: @%s you are not allowed to use the command `%s`"
	return fmt.Sprintf(format, to, name)
}
