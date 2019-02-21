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
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"time"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	yaml "k8s.io/apimachinery/pkg/util/yaml"
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
	if phase == argov1.NodeSucceeded || phase == argov1.NodeFailed || phase == argov1.NodeError || phase == argov1.NodeSkipped {
		return true
	}
	return false
}

// ParseTestrun reads the testrun.yaml file from filePath and parses the yaml.
func ParseTestrun(filePath string) (tmv1beta1.Testrun, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return tmv1beta1.Testrun{}, err
	}

	jsonBody, err := yaml.ToJSON(data)
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
	jsonBody, err := yaml.ToJSON(data)
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
