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

package main

import (
	"context"
	"fmt"
	"github.com/gardener/test-infra/pkg/logger"
	tm_bot "github.com/gardener/test-infra/pkg/tm-bot"
	vh "github.com/gardener/test-infra/pkg/util/cmdutil/viper"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
)

func init() {
	viperHelper := vh.NewViperHelper(nil, "config", "$HOME/.tm-bot", ".")
	vh.SetViper(viperHelper)
	viperHelper.InitFlags(nil)
	tm_bot.InitFlags(nil)
	logger.InitFlags(nil)
	viperHelper.BindPFlags(flag.CommandLine, "")
}

func main() {
	flag.Parse()
	if err := vh.ViperHelper.ReadInConfig(); err != nil {
		fmt.Printf(err.Error())
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			break
		default:
			os.Exit(1)
		}
	}
	ctx := context.Background()
	log, err := logger.New(nil)
	if err != nil {
		fmt.Printf(err.Error())
		os.Exit(1)
	}
	logger.SetLogger(log)
	tm_bot.Serve(ctx, log)
}
