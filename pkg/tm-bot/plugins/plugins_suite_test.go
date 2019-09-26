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

package plugins_test

import (
	"github.com/gardener/test-infra/pkg/tm-bot/plugins"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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

	It("should parse multiple commands", func() {
		input := `/test --arg1=asdf 
/cmd2 --test		
`
		expect := [][]string{
			{"test", "--arg1=asdf"},
			{"cmd2", "--test"},
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
