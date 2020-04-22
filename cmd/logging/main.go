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
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"

	flag "github.com/spf13/pflag"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	corev1 "k8s.io/api/core/v1"
	kubernetesclientset "k8s.io/client-go/kubernetes"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	kubeconfigPath string
	namespace      string
	output         string
)

var (
	GardenerComponentLabels = map[string]string{
		"app": "gardener",
	}
)

func init() {
	// configuration flags
	flag.StringVar(&kubeconfigPath, "kubeconfig", "", "Path to the host cluster kubeconfigPath")
	flag.StringVar(&namespace, "namespace", v1beta1constants.GardenNamespace, "Namespace of the gardener deployments")
	flag.StringVar(&output, "output", "", "Logs output directory path")
}

func main() {
	flag.Parse()
	ctx := context.Background()
	defer ctx.Done()

	if kubeconfigPath == "" {
		if os.Getenv("KUBECONFIG") == "" {
			fmt.Println("Kubeconfig is neither defined by commandline --kubeconfig neither by environment variable 'KUBECONFIG'")
			os.Exit(1)
		}
		kubeconfigPath = os.Getenv("KUBECONFIG")
	}

	// if file does not exist we exit with 0 as this means that gardener wasn't deployed
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		fmt.Printf("host kubeconfig at %s does not exists\n", kubeconfigPath)
		os.Exit(1)
	}

	k8sClient, err := kubernetes.NewClientFromFile("", kubeconfigPath, kubernetes.WithClientOptions(client.Options{
		Scheme: kubernetes.ShootScheme,
	}))
	if err != nil {
		log.Fatalf("cannot build config from path %s: %s", kubeconfigPath, err.Error())
	}

	pods := &corev1.PodList{}
	if err := k8sClient.Client().List(ctx, pods, client.InNamespace(namespace), client.MatchingLabels(GardenerComponentLabels)); err != nil {
		fmt.Printf("unable to list pods in %s: %s\n", namespace, err.Error())
		os.Exit(1)
	}

	var result *multierror.Error
	for _, pod := range pods.Items {
		logs, err := getPodLogs(k8sClient.Kubernetes(), pod)
		if err != nil {
			result = multierror.Append(result, err)
			continue
		}

		fmt.Printf("[%s]\n", pod.Name)
		fmt.Print(string(logs))
		fmt.Println("---------------------------------")

		if err := writePodLogs(pod.Name, output, logs); err != nil {
			result = multierror.Append(result, err)
		}
	}

	if result.ErrorOrNil() != nil {
		fmt.Println(result.Error())
		os.Exit(1)
	}
	fmt.Println("Successfully fetched all logs")
}

func getPodLogs(client kubernetesclientset.Interface, pod corev1.Pod) ([]byte, error) {
	req := client.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})
	logs, err := req.Stream()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get logs from pod %s in namespace %s", pod.Name, pod.Namespace)
	}
	defer logs.Close()

	buf := bytes.NewBuffer([]byte{})
	_, err = io.Copy(buf, logs)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func writePodLogs(podName, outputPath string, logs []byte) error {
	if outputPath == "" {
		return nil
	}
	if err := os.MkdirAll(outputPath, os.ModePerm); err != nil {
		return err
	}
	fileName := fmt.Sprintf("%s.log", podName)

	return ioutil.WriteFile(path.Join(outputPath, fileName), logs, os.ModePerm)
}
