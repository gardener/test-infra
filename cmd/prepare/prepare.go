// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	prepare "github.com/gardener/test-infra/pkg/testmachinery/prepare"
	"github.com/gardener/test-infra/pkg/util"
)

const bloblessClone = "--filter=blob:none"

func runPrepare(log logr.Logger, cfg *prepare.Config, repoBasePath string) error {
	fmt.Printf("\n%s\n\n", util.PrettyPrintStruct(cfg))

	if err := createDirectories(log.WithName("create-directories"), cfg.Directories); err != nil {
		return err
	}

	for _, repo := range cfg.Repositories {
		if err := cloneRepository(log.WithName("clone-repositories"), repo, repoBasePath); err != nil {
			return err
		}
	}

	if err := createTMKubeconfigFile(log.WithName("create-tm-kubeconfig")); err != nil {
		return err
	}

	return nil
}

func createTMKubeconfigFile(log logr.Logger) error {
	log.Info("Creating TestMachinery kubeconfig file from pod service account")
	tmFilePath := filepath.Join(os.Getenv(testmachinery.TM_KUBECONFIG_PATH_NAME), tmv1beta1.TestMachineryKubeconfigName)
	kubeconfig, err := util.CreateKubeconfigFromInternal()
	if err != nil {
		return err
	}

	log.Info("Write TestMachinery kubeconfig to file", "file", tmFilePath)
	return os.WriteFile(tmFilePath, kubeconfig, 0600)
}

func createDirectories(log logr.Logger, directories []string) error {
	for _, dir := range directories {
		log.Info("create directory", "dir", dir)
		if err := os.MkdirAll(dir, 0750); err != nil {
			return err
		}
	}
	return nil
}

func cloneRepository(log logr.Logger, repo *prepare.Repository, repoBasePath string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	repoPath := path.Join(repoBasePath, repo.Name)
	log.Info("Clone repo", "repo", repo.URL, "revision", repo.Revision, "path", repoPath)

	if err := runCommand(log, cwd, "git", "clone", bloblessClone, "-v", repo.URL, repoPath); err != nil {
		// do some checks to diagnose why git clone fails
		if addrs, e := net.LookupHost("github.com"); e == nil {
			fmt.Printf("LookupHost github.com: %v\n", addrs)
		}
		if addrs, e := net.LookupHost("google.com"); e == nil {
			fmt.Printf("LookupHost google.com: %v\n", addrs)
		}
		if addrs, e := net.LookupHost("kubernetes.default.svc.cluster.local"); e == nil {
			fmt.Printf("LookupHost kubernetes.default.svc.cluster.local: %v\n", addrs)
		}
		_ = runCommand(log, cwd, "nslookup", "github.com")
		_ = runCommand(log, cwd, "nslookup", "google.com")
		_ = runCommand(log, cwd, "nslookup", "kubernetes.default.svc.cluster.local")
		// for whatever reason, git clone sometimes fails to resolve github.com => workaround by retrying
		log.Info("git clone failed => retrying once")
		if err := runCommand(log, cwd, "git", "clone", bloblessClone, "-v", repo.URL, repoPath); err != nil {
			return err
		}
	}

	if err := runCommand(log, repoPath, "git", "fetch", "origin", repo.Revision); err != nil {
		return err
	}

	if err := runCommand(log, repoPath, "git", "checkout", repo.Revision, "--"); err != nil {
		return err
	}

	if err := os.RemoveAll(path.Join(repoPath, ".git")); err != nil {
		return err
	}
	return nil
}

func readConfigFile(file string) (*prepare.Config, error) {
	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, err
	}

	cfg := &prepare.Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func runCommand(log logr.Logger, dir string, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Error(err, fmt.Sprintf("Couldn't execute command '%s' with args '%s' in dir '%s'", command, strings.Join(args, " "), dir))
		return err
	}
	return nil
}
