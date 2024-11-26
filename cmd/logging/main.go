// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

//TODO remove

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	kubernetesclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/pkg/util/gardener"
	kutil "github.com/gardener/test-infra/pkg/util/kubernetes"
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
	if err := run(ctx); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	if kubeconfigPath == "" {
		if os.Getenv("KUBECONFIG") == "" {
			return errors.New("Kubeconfig is neither defined by commandline --kubeconfig neither by environment variable 'KUBECONFIG'")
		}
		kubeconfigPath = os.Getenv("KUBECONFIG")
	}

	// if file does not exist we exit with 0 as this means that gardener wasn't deployed
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return fmt.Errorf("host kubeconfig at %s does not exists", kubeconfigPath)
	}

	kubeconfigBytes, err := os.ReadFile(filepath.Clean(kubeconfigPath))
	if err != nil {
		return fmt.Errorf("unable to read host kubeconfig: %w", err)
	}
	config, err := clientcmd.NewClientConfigFromBytes(kubeconfigBytes)
	if err != nil {
		return fmt.Errorf("unable to read k8s config: %w", err)
	}
	restClient, err := config.ClientConfig()
	if err != nil {
		return fmt.Errorf("unable to build rest config: %w", err)
	}
	k8sClient, err := kutil.NewClientFromBytes(kubeconfigBytes, client.Options{
		Scheme: gardener.ShootScheme,
	})
	if err != nil {
		return fmt.Errorf("cannot build config from path %s: %s", kubeconfigPath, err.Error())
	}
	k8sClientset, err := kubernetesclientset.NewForConfig(restClient)
	if err != nil {
		return fmt.Errorf("cannot build kubernetes clientset from path %s: %s", kubeconfigPath, err.Error())
	}

	pods := &corev1.PodList{}
	if err := k8sClient.List(ctx, pods, client.InNamespace(namespace), client.MatchingLabels(GardenerComponentLabels)); err != nil {
		return fmt.Errorf("unable to list pods in %s: %s", namespace, err.Error())
	}

	var result *multierror.Error
	for _, pod := range pods.Items {
		logs, err := getPodLogs(ctx, k8sClientset, pod)
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
	return nil
}

func getPodLogs(ctx context.Context, client kubernetesclientset.Interface, pod corev1.Pod) ([]byte, error) {
	req := client.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})
	logs, err := req.Stream(ctx)
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
	if err := os.MkdirAll(outputPath, 0750); err != nil {
		return err
	}
	fileName := fmt.Sprintf("%s.log", podName)

	return os.WriteFile(path.Join(outputPath, fileName), logs, 0600)
}
