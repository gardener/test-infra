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

package xkcd

import (
	"encoding/json"
	"fmt"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins"
	pluginerr "github.com/gardener/test-infra/pkg/tm-bot/plugins/errors"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"strconv"
)

type xkcdInfo struct {
	Title string `json:"safe_title"`
	Img   string `json:"img"`
	Num   int    `json:"num"`
}

type xkcd struct {
	URL *url.URL
	num int
}

func New() (plugins.Plugin, error) {
	u, err := url.Parse("https://xkcd.com")
	if err != nil {
		return nil, err
	}
	return &xkcd{URL: u}, nil
}

func (x *xkcd) New(_ string) plugins.Plugin {
	p := *x
	return &p
}

func (_ *xkcd) Command() string {
	return "xkcd"
}

func (_ *xkcd) Authorization() github.AuthorizationType {
	return github.AuthorizationAll
}

func (_ *xkcd) Description() string {
	return "Adds an random image of xkcd"
}

func (_ *xkcd) Example() string {
	return "/xkcd --num 2"
}

func (_ *xkcd) ResumeFromState(_ github.Client, _ *github.GenericRequestEvent, _ string) error {
	return nil
}

func (x *xkcd) Flags() *pflag.FlagSet {
	flagset := pflag.NewFlagSet(x.Command(), pflag.ContinueOnError)
	flagset.IntVar(&x.num, "num", 0, "XKCD image number")
	return flagset
}

func (x *xkcd) Run(flagset *pflag.FlagSet, client github.Client, event *github.GenericRequestEvent) error {
	max, err := x.getCurrentMax()
	if err != nil {
		return pluginerr.New("unable to get maximum xkcd number", err.Error())
	}
	if flagset.Changed("num") {
		if x.num > max {
			return pluginerr.New(fmt.Sprintf("xkcd %d does not exist. The maximum number is currently %d", x.num, max), "")
		}
	} else {
		x.num = rand.Intn(max)
	}

	info, err := x.GetImage(x.num)
	if err != nil {
		return nil
	}

	_, err = client.Comment(event, formatResponse(info))
	return err
}

func (x *xkcd) GetImage(num int) (*xkcdInfo, error) {
	u := *x.URL
	u.Path = path.Join(u.Path, strconv.Itoa(num), "info.0.json")

	res, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	info := xkcdInfo{}
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&info); err != nil {
		return nil, err
	}

	if info.Img == "" {
		return nil, errors.New("no image could be found")
	}
	return &info, nil
}

func (x *xkcd) getCurrentMax() (int, error) {
	u := *x.URL
	u.Path = path.Join(u.Path, "info.0.json")

	res, err := http.Get(u.String())
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	info := xkcdInfo{}
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&info); err != nil {
		return 0, err
	}

	return info.Num, nil
}

func formatResponse(info *xkcdInfo) string {
	msg := `
**%s** [link](%s)
![xkcd image](%s)
`
	return fmt.Sprintf(msg, info.Title, info.Img, info.Img)
}
