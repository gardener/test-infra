// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package plugins_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/test-infra/pkg/tm-bot/plugins"
)

var _ = Describe("Commands", func() {
	It("should parse one line command", func() {
		input := "/test --arg1=asdf --args2 2"
		expect := [][]string{
			{"test", "--arg1=asdf", "--args2", "2"},
		}

		actual, err := plugins.ParseCommands(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(actual).To(Equal(expect))
	})

	It("should parse command with string flag", func() {
		input := "/test --arg1=\"hello world\" --args2 2"
		expect := [][]string{
			{"test", "--arg1=hello world", "--args2", "2"},
		}

		actual, err := plugins.ParseCommands(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(actual).To(Equal(expect))
	})

	It("should parse multiple commands", func() {
		input := "/test --arg1=asdf\n/cmd2 --test\r\n/cmd3 --test"
		expect := [][]string{
			{"test", "--arg1=asdf"},
			{"cmd2", "--test"},
			{"cmd3", "--test"},
		}

		actual, err := plugins.ParseCommands(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(actual).To(Equal(expect))
	})

	It("should parse multi line commands", func() {
		input := `/test --arg1=asdf 
--args2 2
--args3		
`
		expect := [][]string{
			{"test", "--arg1=asdf"},
		}

		actual, err := plugins.ParseCommands(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(actual).To(Equal(expect))
	})

	It("should ignore non command text", func() {
		input := `/test --arg1=asdf 
/cmd2 --test
this is a example text
of multiple lines
`
		expect := [][]string{
			{"test", "--arg1=asdf"},
			{"cmd2", "--test"},
		}

		actual, err := plugins.ParseCommands(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(actual).To(Equal(expect))
	})
})
