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

package main

/*
FILEPATH="$1"

for d in $(cat "$FILEPATH" | jq -c '.directories[]' ); do
    mkdir -p $d
done

for repo in $(cat "$FILEPATH" | jq -c '.repositories[]' ); do
    url=$( echo $repo | jq -r '.url')
    revision=$( echo $repo | jq -r '.revision')
    name=$( echo $repo | jq -r '.name')

    echo "Clone repo $url with revision $revision to $TM_REPO_PATH/$name \n"
    git clone -v $url $TM_REPO_PATH/$name;

    pushd .
    cd $TM_REPO_PATH/$name
    git fetch origin $revision
    git checkout $revision
    rm -rf .git
    popd
done
 */

import (
	"encoding/json"
	"fmt"
	prepare "github.com/gardener/test-infra/pkg/testmachinery/prepare"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

func runPrepare(log logr.Logger, cfg *prepare.Config, repoBasePath string) error {
	fmt.Printf("\n%s\n\n", util.PrettyPrintStruct(cfg))

	for _, dir := range cfg.Directories {
		log.Info("create directory", "dir", dir)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}
	}

	for _, repo := range cfg.Repositories {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		repoPath := path.Join(repoBasePath, repo.Name)
		log.Info("Clone repo", "repo", repo.URL, "revision", repo.Revision, "path", repoPath)

		if err := runGit(cwd, "clone", "-v",  repo.URL, repoPath); err != nil {
			return err
		}

		if err := runGit(repoPath, "fetch", "origin", repo.Revision); err != nil {
			return err
		}

		if err := runGit(repoPath, "checkout", repo.Revision); err != nil {
			return err
		}

		if err := os.RemoveAll(path.Join(repoPath, ".git")); err != nil {
			return err
		}

	}

	return nil
}

func readConfigFile(file string) (*prepare.Config, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	cfg := &prepare.Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func runGit(pwd string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = pwd

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err :=  cmd.Run(); err != nil {
		return errors.Wrap(err, fmt.Sprintf("Command: %s", strings.Join(args, " ")))
	}
	return nil
}
