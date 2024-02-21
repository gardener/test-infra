// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package echo

import (
	"context"
	"fmt"

	"github.com/spf13/pflag"

	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins"
)

type echo struct {
	runID string
	value string
}

func New() plugins.Plugin {
	return &echo{}
}

func (e *echo) New(runID string) plugins.Plugin {
	return &echo{runID: runID}
}

func (_ *echo) Command() string {
	return "echo"
}

func (_ *echo) Authorization() github.AuthorizationType {
	return github.AuthorizationTeam
}

func (_ *echo) Description() string {
	return "Prints the provided value"
}

func (_ *echo) Example() string {
	return "/echo \"text to echo\""
}

func (_ *echo) Config() string {
	return ""
}

func (_ *echo) ResumeFromState(_ github.Client, _ *github.GenericRequestEvent, _ string) error {
	return nil
}

func (e *echo) Flags() *pflag.FlagSet {
	flagset := pflag.NewFlagSet(e.Command(), pflag.ContinueOnError)
	flagset.StringVarP(&e.value, "value", "v", "", "Echo value")
	return flagset
}

func (e *echo) Run(flagset *pflag.FlagSet, client github.Client, event *github.GenericRequestEvent) error {
	cfg, err := client.GetRawConfig(e.Command())
	if err == nil {
		fmt.Println(string(cfg))
	}

	var val string
	if flagset.NArg() == 0 {
		val, err = flagset.GetString("value")
		if err != nil {
			return err
		}
	} else {
		val = flagset.Arg(0)
	}

	_, err = client.Comment(context.TODO(), event, fmt.Sprintf("@%s: %s\n%s", event.GetAuthorName(), val, e.runID))
	return err
}
