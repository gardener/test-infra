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
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math"
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

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v50/github"
	"github.com/pkg/errors"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
	clientv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/util/elasticsearch"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyz1234567890"

// MaxTimeExceeded checks if the max time is exceeded.
func MaxTimeExceeded(startTime time.Time, maxWaitTime int64) bool {
	maxTime := startTime.Add(time.Duration(maxWaitTime) * time.Second)
	return maxTime.Before(time.Now())
}

// CompletedStep checks if the teststep is in a completed phase
func CompletedStep(phase argov1.NodePhase) bool {
	if phase == tmv1beta1.StepPhaseSuccess || phase == tmv1beta1.StepPhaseFailed || phase == tmv1beta1.StepPhaseError || phase == tmv1beta1.StepPhaseSkipped || phase == tmv1beta1.StepPhaseTimeout {
		return true
	}
	return false
}

// CompletedRun checks if the testrun is in a completed phase
func CompletedRun(phase argov1.WorkflowPhase) bool {
	if phase == tmv1beta1.RunPhaseSuccess || phase == tmv1beta1.RunPhaseFailed || phase == tmv1beta1.RunPhaseError || phase == tmv1beta1.RunPhaseTimeout {
		return true
	}
	return false
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

	data, err := io.ReadAll(resp.Body)
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

// DomainMatches returns true if one of the given domains or subdomains match.
func DomainMatches(s string, domains ...string) bool {
	normalizedDomain := strings.ToUpper(s)
	for _, d := range domains {
		if strings.HasSuffix(normalizedDomain, strings.ToUpper(d)) {
			return true
		}
	}
	return false
}

// HasLabel returns a bool if passed in label exists
func HasLabel(obj metav1.ObjectMeta, label string) bool {
	_, found := obj.Labels[label]
	return found
}

// SetMetaDataLabel sets the label and value
func SetMetaDataLabel(obj *metav1.ObjectMeta, label string, value string) {
	if obj.Labels == nil {
		obj.Labels = make(map[string]string)
	}
	obj.Labels[label] = value
}

// GitHub helper functions

// ParseRepoURLFromString returns the repository owner and name of a github repo url
func ParseRepoURLFromString(repoURL string) (repoOwner, repoName string, err error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", "", err
	}

	repoOwner, repoName = ParseRepoURL(u)
	return repoOwner, repoName, nil
}

// ParseRepoURL returns the repository owner and name of a github repo url
func ParseRepoURL(url *url.URL) (repoOwner, repoName string) {
	repoNameComponents := strings.Split(url.Path, "/")
	repoOwner = repoNameComponents[1]
	repoName = strings.Replace(repoNameComponents[2], ".git", "", 1)
	return
}

// GetGitHubClient returns a new github enterprise client with basic auth and optional tls verification
func GetGitHubClient(apiURL, username, password, uploadURL string, skipTLS bool) (*github.Client, error) {
	ghClient, err := github.NewEnterpriseClient(apiURL, uploadURL, GetHTTPClient(username, password, skipTLS))
	if err != nil {
		return nil, err
	}
	return ghClient, nil
}

// GetHTTPClient returns a new http client with basic auth and optional tls verification
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

// CreateKubeconfigFromClient creates a new kubeocnfig file from a resclient config
func CreateKubeconfigFromInternal() ([]byte, error) {
	config, err := restclient.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get in-cluster kubeconfig")
	}

	rootCAFile := "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	caCert, err := os.ReadFile(rootCAFile)
	if err != nil {
		return nil, err
	}

	kubeconfig := clientv1.Config{
		CurrentContext: "default",
		Contexts: []clientv1.NamedContext{
			{
				Name: "default",
				Context: clientv1.Context{
					Cluster:   "default",
					AuthInfo:  "default",
					Namespace: "default",
				},
			},
		},
		Clusters: []clientv1.NamedCluster{
			{
				Name: "default",
				Cluster: clientv1.Cluster{
					Server:                   config.Host,
					InsecureSkipTLSVerify:    config.Insecure,
					CertificateAuthorityData: caCert,
				},
			},
		},
		AuthInfos: []clientv1.NamedAuthInfo{
			{
				Name: "default",
				AuthInfo: clientv1.AuthInfo{
					Token: config.BearerToken,
				},
			},
		},
	}

	return yaml.Marshal(kubeconfig)
}

// Unzip unpacks the given archive to the specified target path
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
			if err := os.MkdirAll(path, file.Mode()); err != nil {
				return err
			}
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

// IsIsEndOfBucket returns if the current value is the last integer value of its bucket
// Examples (bucket size: 3):
// 0: false
// 1: false
// 2: true
// 3: false
// 5: true
func IsLastElementOfBucket(value, bucketSize int) bool {
	if bucketSize == 0 {
		return true
	}
	mod := float64(value+1) / float64(bucketSize)
	return mod == math.Trunc(mod)
}

// Zipit zips a source file or directory and writes the archive to the specified target path
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

// DocExists checks whether an elasticsearch doc of a testrun exists in the testmachinery-* index
func DocExists(log logr.Logger, esClient elasticsearch.Client, testrunID, testrunStartTime string) (docExists bool) {
	if esClient == nil {
		return false
	}
	log.V(2).Info("check if docs have already been ingested")

	payload := fmt.Sprintf(`{
			"size": 0,
			"query": {
				"bool": {
					"must": [
						{ "match": { "tm.tr.id.keyword": "%s" } },
						{ "match": { "tm.tr.startTime": "%s" } }
					]
				}
			}
		}`, testrunID, testrunStartTime)

	responseBytes, err := esClient.Request(http.MethodGet, "/testmachinery-*/_search", strings.NewReader(payload))
	if err != nil {
		log.Error(err, "elasticsearch request failed")
		return false
	}
	var esHits ESHits
	if err = json.Unmarshal(responseBytes, &esHits); err != nil {
		log.Error(err, fmt.Sprintf("elasticsearch hits response unmarshal failed for payload: '%s'", string(responseBytes)))
	}
	return esHits.Hits.Total.Value > 0
}

type ESHits struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
	} `json:"hits"`
}

// GetClusterDomainURL tries to derive the cluster domain url from a grafana ingress if possible. Returns an error if the ingress cannot be found or is in unexpected form.
func GetClusterDomainURL(tmClient client.Client) (string, error) {
	// try to derive the cluster domain url from grafana ingress if possible
	// return err if the ingress cannot be found
	if tmClient == nil {
		return "", nil
	}
	ingress := &netv1.Ingress{}
	err := tmClient.Get(context.TODO(), client.ObjectKey{Namespace: "monitoring", Name: "grafana"}, ingress)
	if err != nil {
		return "", fmt.Errorf("cannot get grafana ingress: %v", err)
	}
	if len(ingress.Spec.Rules) == 0 {
		return "", fmt.Errorf("cannot get ingress rule from ingress %v", ingress)
	}
	host := ingress.Spec.Rules[0].Host
	return parseDomain(host)
}

func parseDomain(hostname string) (string, error) {
	r, _ := regexp.Compile(`[a-z]+\.ingress\.(.+)$`)
	matches := r.FindStringSubmatch(hostname)
	if len(matches) < 2 {
		return "", fmt.Errorf("cannot regex cluster domain from hostname %v", hostname)
	}
	return matches[1], nil
}

// MarshalMap encodes the given map into a string similar to kubectl --selectors: map '{a:b,c:d}' becomes string 'a=b,c=d'
func MarshalMap(annotations map[string]string) string {
	var buf []string
	for k, v := range annotations {
		buf = append(buf, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(buf, ",")
}

// UnmarshalMap does the opposite of MarshalMap. It decodes the given string into a map, return an error if the string is not in the expected format 'key1=value1,key2=value2'
func UnmarshalMap(cfg string) (map[string]string, error) {
	if len(cfg) == 0 {
		return nil, nil
	}
	result := make(map[string]string)
	annotations := strings.Split(cfg, ",")
	for _, annotation := range annotations {
		annotation = strings.TrimSpace(annotation)
		if len(annotation) == 0 {
			continue
		}
		keyValue := strings.Split(annotation, "=")
		if len(keyValue) != 2 {
			return nil, fmt.Errorf("annotation %s could not be parsed into key and value", annotation)
		}
		result[keyValue[0]] = keyValue[1]
	}

	return result, nil
}
