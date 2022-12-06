// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"
	"os"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewClientFromSecret creates a new Client struct for a given kubeconfig stored as a
// Secret in an existing Kubernetes cluster. This cluster will be accessed by the <k8sClient>. It will
// read the Secret <secretName> in <namespace>. The Secret must contain a field "kubeconfig" which will
// be used.
func NewClientFromSecret(ctx context.Context, c client.Client, namespace, secretName string, opts client.Options) (client.Client, error) {
	secret := &corev1.Secret{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: secretName}, secret); err != nil {
		return nil, err
	}
	return NewClientFromSecretObject(secret, opts)
}

// NewClientFromSecretObject creates a new Client struct for a given Kubernetes Secret object. The Secret must
// contain a field "kubeconfig" which will be used.
func NewClientFromSecretObject(secret *corev1.Secret, opts client.Options) (client.Client, error) {
	if kubeconfig, ok := secret.Data["kubeconfig"]; ok {
		if len(kubeconfig) == 0 {
			return nil, errors.New("the secret's field 'kubeconfig' is empty")
		}
		return NewClientFromBytes(kubeconfig, opts)
	}
	return nil, errors.New("the secret does not contain a field with name 'kubeconfig'")
}

// NewClientFromFile creates a new Client struct from a kubconfig file.
func NewClientFromFile(kubeconfigPath string, opts client.Options) (client.Client, error) {
	data, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return NewClientFromBytes(data, opts)
}

// NewClientFromBytes creates a new client from kubeconfig.
func NewClientFromBytes(data []byte, opts client.Options) (client.Client, error) {
	config, err := clientcmd.NewClientConfigFromBytes(data)
	if err != nil {
		return nil, err
	}
	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, err
	}
	return client.New(restConfig, opts)
}
