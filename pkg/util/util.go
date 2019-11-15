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
	"archive/zip"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/google/go-github/v27/github"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"sigs.k8s.io/yaml"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz1234567890"

// MaxTimeExceeded checks if the max time is exceeded.
func MaxTimeExceeded(startTime time.Time, maxWaitTime int64) bool {
	maxTime := startTime.Add(time.Duration(maxWaitTime) * time.Second)
	return maxTime.Before(time.Now())
}

// Completed checks if the testrun is in a completed phase
func Completed(phase argov1.NodePhase) bool {
	if phase == tmv1beta1.PhaseStatusSuccess || phase == tmv1beta1.PhaseStatusFailed || phase == tmv1beta1.PhaseStatusError || phase == tmv1beta1.PhaseStatusSkipped || phase == tmv1beta1.PhaseStatusTimeout {
		return true
	}
	return false
}

// ParseTestrunFromFile reads a testrun.yaml file from filePath and parses the yaml.
func ParseTestrunFromFile(filePath string) (tmv1beta1.Testrun, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return tmv1beta1.Testrun{}, err
	}
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return tmv1beta1.Testrun{}, err
	}

	return ParseTestrun(data)
}

// ParseTestrun parses testrun.
func ParseTestrun(data []byte) (tmv1beta1.Testrun, error) {
	if len(data) == 0 {
		return tmv1beta1.Testrun{}, errors.New("empty data")
	}
	jsonBody, err := yaml.YAMLToJSON(data)
	if err != nil {
		return tmv1beta1.Testrun{}, err
	}

	var testrun tmv1beta1.Testrun
	err = json.Unmarshal(jsonBody, &testrun)
	if err != nil {
		return tmv1beta1.Testrun{}, err
	}
	return testrun, nil
}

// ParseTestDef parses a file into a TestDefinition.
func ParseTestDef(data []byte) (tmv1beta1.TestDefinition, error) {
	jsonBody, err := yaml.YAMLToJSON(data)
	if err != nil {
		return tmv1beta1.TestDefinition{}, err
	}

	var testDef tmv1beta1.TestDefinition
	err = json.Unmarshal(jsonBody, &testDef)
	if err != nil {
		return tmv1beta1.TestDefinition{}, err
	}

	return testDef, nil
}

// DownloadFile downloads a file from the given url and return the content.
func DownloadFile(client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Cannot downdload file from %s: \n %v", url, err.Error())
	}
	return data, nil
}

// Getenv returns the string value of the environment variable with the provided key if the env var exists.
// Otherwise the default value is returned
func Getenv(key, defaultValue string) string {
	if os.Getenv(key) != "" {
		return os.Getenv(key)
	}
	return defaultValue
}

// GetenvBool returns the boolean value of the environment variable with the provided key if the env var exists and can be parsed.
// Otherwise the default value is returned
func GetenvBool(key string, defaultValue bool) bool {
	env := os.Getenv(key)
	if env != "" {
		if b, err := strconv.ParseBool(env); err == nil {
			return b
		}
	}
	return defaultValue
}

// RandomString generates a random string out of "abcdefghijklmnopqrstuvwxyz1234567890" with a given length.RandomString
// The generated string is compatible to k8s name conventions
func RandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

// IsAnnotationSubset checks if all items in target are deep equal in src with the same key
func IsAnnotationSubset(src, target map[string]string) bool {
	for key, value := range target {
		if !reflect.DeepEqual(value, src[key]) {
			return false
		}
	}

	return true
}

// FormatArtifactName replaces all invalid artifact name characters.
// It replaces everything that is not an alphan-numeric character or "-" with a "-".
func FormatArtifactName(name string) string {
	reg := regexp.MustCompile(`[^[a-zA-Z0-9-]]*`)
	return reg.ReplaceAllString(name, "-")
}

// PrettyPrintStruct returns an obj as pretty printed yaml.
func PrettyPrintStruct(obj interface{}) string {
	str, err := yaml.Marshal(obj)
	if err != nil {
		return ""
	}
	return string(str)
}

// MarshalNoHTMLEscape is nearly same as json.Marshal but does NOT HTLM-escape <, > or &
// However it does add a newline char at the end (as done by json.Encoder.Encode)
func MarshalNoHTMLEscape(v interface{}) ([]byte, error) {
	buffer := bytes.NewBuffer([]byte{})
	enc := json.NewEncoder(buffer)
	enc.SetEscapeHTML(false)
	err := enc.Encode(v)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// StringArrayContains checks if a string array contains the string elem
func StringArrayContains(ar []string, elem string) bool {
	for _, val := range ar {
		if val == elem {
			return true
		}
	}
	return false
}

// StringDefault checks if a string is defined.
// If the value is emtpy the default string is returned
func StringDefault(value, def string) string {
	if value == "" {
		return def
	}
	return value
}

// GitHub helper functions
func ParseRepoURL(url *url.URL) (repoOwner, repoName string) {
	repoNameComponents := strings.Split(url.Path, "/")
	repoOwner = repoNameComponents[1]
	repoName = strings.Replace(repoNameComponents[2], ".git", "", 1)
	return
}

func GetGitHubClient(apiURL, username, password, uploadURL string, skipTLS bool) (*github.Client, error) {
	client, err := github.NewEnterpriseClient(apiURL, uploadURL, GetHTTPClient(username, password, skipTLS))
	if err != nil {
		return nil, err
	}
	return client, nil
}

func GetHTTPClient(username, password string, skipTLS bool) *http.Client {
	if username != "" && password != "" {
		basicAuth := github.BasicAuthTransport{
			Username: username,
			Password: password,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: skipTLS},
			},
		}
		return basicAuth.Client()
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}
func Unzip(archive, target string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}

	for _, file := range reader.File {
		path := filepath.Join(target, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer targetFile.Close()

		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return err
		}
	}

	return nil
}

func Zipit(source, target string) error {
	zipfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	info, err := os.Stat(source)
	if err != nil {
		return nil
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	if err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		if baseDir != "" {
			header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
		}

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	}); err != nil {
		return err
	}

	return err
}