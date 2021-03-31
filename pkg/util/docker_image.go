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

package util

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

type dockerImages struct {
	Tags []string `json:"tags"`
}

// CheckDockerImageExists checks if a docker image exists
func CheckDockerImageExists(image, tag string) error {

	// Build hostname/v2/<image>/manifests/<tag> to directly check if the image exists
	splitImage := strings.Split(image, "/")
	tail := splitImage[1:]
	reqPath := append(append([]string{"v2"}, tail...), "manifests", tag)

	u := &url.URL{
		Scheme: "https",
		Host:   splitImage[0],
		Path:   strings.Join(reqPath, "/"),
	}
	res, err := http.Get(u.String())
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New("tag does not exist")
	}
	return nil
}

// GetDockerImageFromCommit searches all tags of a image and try to matches the commit (e.g. .10.0-dev-<commit>).
// The image tag is returned if an applicable tag can be found
// todo: use pagination if gcr will support is someday
func GetDockerImageFromCommit(image, commit string) (string, error) {

	// construct api call with the form hostname/v2/<image>/tags/list
	splitImage := strings.Split(image, "/")
	tail := splitImage[1:]
	reqPath := append(append([]string{"v2"}, tail...), "tags", "list")

	u := &url.URL{
		Scheme: "https",
		Host:   splitImage[0],
		Path:   strings.Join(reqPath, "/"),
	}
	res, err := http.Get(u.String())
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", errors.New("no tag found")
	}
	defer res.Body.Close()

	decoder := json.NewDecoder(res.Body)
	var images dockerImages
	if err := decoder.Decode(&images); err != nil {
		return "", err
	}

	for _, tag := range images.Tags {
		if strings.Contains(tag, commit) {
			return tag, nil
		}
	}

	return "", errors.New("no tag found")
}
