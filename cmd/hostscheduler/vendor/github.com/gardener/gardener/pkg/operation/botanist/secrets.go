// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package botanist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"

	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	"github.com/gardener/gardener/pkg/apis/garden"
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	controllermanagerfeatures "github.com/gardener/gardener/pkg/controllermanager/features"
	"github.com/gardener/gardener/pkg/features"
	"github.com/gardener/gardener/pkg/operation/common"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/gardener/pkg/utils/secrets"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/authentication/user"
)

var wantedCertificateAuthorities = map[string]*secrets.CertificateSecretConfig{
	gardencorev1alpha1.SecretNameCACluster: {
		Name:       gardencorev1alpha1.SecretNameCACluster,
		CommonName: "kubernetes",
		CertType:   secrets.CACert,
	},
	gardencorev1alpha1.SecretNameCAETCD: {
		Name:       gardencorev1alpha1.SecretNameCAETCD,
		CommonName: "etcd",
		CertType:   secrets.CACert,
	},
	gardencorev1alpha1.SecretNameCAFrontProxy: {
		Name:       gardencorev1alpha1.SecretNameCAFrontProxy,
		CommonName: "front-proxy",
		CertType:   secrets.CACert,
	},
	gardencorev1alpha1.SecretNameCAKubelet: {
		Name:       gardencorev1alpha1.SecretNameCAKubelet,
		CommonName: "kubelet",
		CertType:   secrets.CACert,
	},
	gardencorev1alpha1.SecretNameCAMetricsServer: {
		Name:       gardencorev1alpha1.SecretNameCAMetricsServer,
		CommonName: "metrics-server",
		CertType:   secrets.CACert,
	},
}

const (
	certificateETCDServer = "etcd-server-tls"
	certificateETCDClient = "etcd-client-tls"
)

// generateWantedSecrets returns a list of Secret configuration objects satisfying the secret config intface,
// each containing their specific configuration for the creation of certificates (server/client), RSA key pairs, basic
// authentication credentials, etc.
func (b *Botanist) generateWantedSecrets(basicAuthAPIServer *secrets.BasicAuth, certificateAuthorities map[string]*secrets.Certificate) ([]secrets.ConfigInterface, error) {
	var (
		alertManagerHost = b.Seed.GetIngressFQDN("a", b.Shoot.Info.Name, b.Garden.Project.Name)
		grafanaHost      = b.Seed.GetIngressFQDN("g", b.Shoot.Info.Name, b.Garden.Project.Name)
		prometheusHost   = b.ComputePrometheusIngressFQDN()

		apiServerIPAddresses = []net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP(common.ComputeClusterIP(b.Shoot.GetServiceNetwork(), 1)),
		}
		apiServerCertDNSNames = append([]string{
			"kube-apiserver",
			fmt.Sprintf("kube-apiserver.%s", b.Shoot.SeedNamespace),
			fmt.Sprintf("kube-apiserver.%s.svc", b.Shoot.SeedNamespace),
			b.Shoot.InternalClusterDomain,
		}, dnsNamesForService("kubernetes", "default")...)

		cloudControllerManagerCertDNSNames = dnsNamesForService("cloud-controller-manager", b.Shoot.SeedNamespace)
		kubeControllerManagerCertDNSNames  = dnsNamesForService("kube-controller-manager", b.Shoot.SeedNamespace)
		kubeSchedulerCertDNSNames          = dnsNamesForService("kube-scheduler", b.Shoot.SeedNamespace)

		etcdCertDNSNames = dnsNamesForEtcd(b.Shoot.SeedNamespace)
	)

	if len(certificateAuthorities) != len(wantedCertificateAuthorities) {
		return nil, fmt.Errorf("missing certificate authorities")
	}

	if b.Shoot.ExternalClusterDomain != nil {
		apiServerCertDNSNames = append(apiServerCertDNSNames, *(b.Shoot.Info.Spec.DNS.Domain), *(b.Shoot.ExternalClusterDomain))
	}

	secretList := []secrets.ConfigInterface{
		// Secret definition for kube-apiserver
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: "kube-apiserver",

				CommonName:   user.APIServerUser,
				Organization: nil,
				DNSNames:     apiServerCertDNSNames,
				IPAddresses:  apiServerIPAddresses,

				CertType:  secrets.ServerCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
			},
		},

		// Secret definition for kube-apiserver to kubelets communication
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: "kube-apiserver-kubelet",

				CommonName:   "system:kube-apiserver:kubelet",
				Organization: nil,
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCAKubelet],
			},
		},

		// Secret definition for kube-aggregator
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: "kube-aggregator",

				CommonName:   "system:kube-aggregator",
				Organization: nil,
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCAFrontProxy],
			},
		},

		// Secret definition for kube-controller-manager
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: "kube-controller-manager",

				CommonName:   user.KubeControllerManager,
				Organization: nil,
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
			},
			KubeConfigRequest: &secrets.KubeConfigRequest{
				ClusterName:  b.Shoot.SeedNamespace,
				APIServerURL: b.Shoot.ComputeAPIServerURL(true, false),
			},
		},

		// Secret definition for kube-controller-manager server
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: common.KubeControllerManagerServerName,

				CommonName:   common.KubeControllerManagerDeploymentName,
				Organization: nil,
				DNSNames:     kubeControllerManagerCertDNSNames,
				IPAddresses:  nil,

				CertType:  secrets.ServerCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
			},
		},

		// Secret definition for cloud-controller-manager
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: "cloud-controller-manager",

				CommonName:   "system:cloud-controller-manager",
				Organization: []string{user.SystemPrivilegedGroup},
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
			},

			KubeConfigRequest: &secrets.KubeConfigRequest{
				ClusterName:  b.Shoot.SeedNamespace,
				APIServerURL: b.Shoot.ComputeAPIServerURL(true, false),
			},
		},

		// Secret definition for cloud-controller-manager server
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: common.CloudControllerManagerServerName,

				CommonName:   common.CloudControllerManagerDeploymentName,
				Organization: nil,
				DNSNames:     cloudControllerManagerCertDNSNames,
				IPAddresses:  nil,

				CertType:  secrets.ServerCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
			},
		},

		// Secret definition for the aws-lb-readvertiser
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: "aws-lb-readvertiser",

				CommonName:   "aws-lb-readvertiser",
				Organization: []string{user.SystemPrivilegedGroup},
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
			},

			KubeConfigRequest: &secrets.KubeConfigRequest{
				ClusterName:  b.Shoot.SeedNamespace,
				APIServerURL: b.Shoot.ComputeAPIServerURL(true, false),
			},
		},

		// Secret definition for kube-scheduler
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: "kube-scheduler",

				CommonName:   user.KubeScheduler,
				Organization: nil,
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
			},

			KubeConfigRequest: &secrets.KubeConfigRequest{
				ClusterName:  b.Shoot.SeedNamespace,
				APIServerURL: b.Shoot.ComputeAPIServerURL(true, false),
			},
		},

		// Secret definition for kube-scheduler server
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: common.KubeSchedulerServerName,

				CommonName:   common.KubeSchedulerDeploymentName,
				Organization: nil,
				DNSNames:     kubeSchedulerCertDNSNames,
				IPAddresses:  nil,

				CertType:  secrets.ServerCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
			},
		},

		// Secret definition for cluster-autoscaler
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: gardencorev1alpha1.DeploymentNameClusterAutoscaler,

				CommonName:   "system:cluster-autoscaler",
				Organization: nil,
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
			},

			KubeConfigRequest: &secrets.KubeConfigRequest{
				ClusterName:  b.Shoot.SeedNamespace,
				APIServerURL: b.Shoot.ComputeAPIServerURL(true, false),
			},
		},

		// Secret definition for kube-addon-manager
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: "kube-addon-manager",

				CommonName:   "system:kube-addon-manager",
				Organization: []string{user.SystemPrivilegedGroup},
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
			},

			KubeConfigRequest: &secrets.KubeConfigRequest{
				ClusterName:  b.Shoot.SeedNamespace,
				APIServerURL: b.Shoot.ComputeAPIServerURL(true, false),
			},
		},

		// Secret definition for csi-attacher
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: "csi-attacher",

				CommonName:   "system:csi-attacher",
				Organization: []string{user.SystemPrivilegedGroup},
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
			},

			KubeConfigRequest: &secrets.KubeConfigRequest{
				ClusterName:  b.Shoot.SeedNamespace,
				APIServerURL: b.Shoot.ComputeAPIServerURL(true, false),
			},
		},

		// Secret definition for csi-provisioner
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: "csi-provisioner",

				CommonName:   "system:csi-provisioner",
				Organization: []string{user.SystemPrivilegedGroup},
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
			},

			KubeConfigRequest: &secrets.KubeConfigRequest{
				ClusterName:  b.Shoot.SeedNamespace,
				APIServerURL: b.Shoot.ComputeAPIServerURL(true, false),
			},
		},

		// Secret definition for csi-snapshotter
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: "csi-snapshotter",

				CommonName:   "system:csi-snapshotter",
				Organization: []string{user.SystemPrivilegedGroup},
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
			},

			KubeConfigRequest: &secrets.KubeConfigRequest{
				ClusterName:  b.Shoot.SeedNamespace,
				APIServerURL: b.Shoot.ComputeAPIServerURL(true, false),
			},
		},
		// Secret definition for kube-proxy
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: "kube-proxy",

				CommonName:   user.KubeProxy,
				Organization: nil,
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
			},

			KubeConfigRequest: &secrets.KubeConfigRequest{
				ClusterName:  b.Shoot.SeedNamespace,
				APIServerURL: b.Shoot.ComputeAPIServerURL(false, true),
			},
		},

		// Secret definition for kube-state-metrics
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: "kube-state-metrics",

				CommonName:   fmt.Sprintf("%s:monitoring:kube-state-metrics", garden.GroupName),
				Organization: []string{fmt.Sprintf("%s:monitoring", garden.GroupName)},
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
			},

			KubeConfigRequest: &secrets.KubeConfigRequest{
				ClusterName:  b.Shoot.SeedNamespace,
				APIServerURL: b.Shoot.ComputeAPIServerURL(true, false),
			},
		},

		// Secret definition for prometheus
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: "prometheus",

				CommonName:   fmt.Sprintf("%s:monitoring:prometheus", garden.GroupName),
				Organization: []string{fmt.Sprintf("%s:monitoring", garden.GroupName)},
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
			},

			KubeConfigRequest: &secrets.KubeConfigRequest{
				ClusterName:  b.Shoot.SeedNamespace,
				APIServerURL: b.Shoot.ComputeAPIServerURL(true, false),
			},
		},

		// Secret definition for prometheus to kubelets communication
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: "prometheus-kubelet",

				CommonName:   fmt.Sprintf("%s:monitoring:prometheus", garden.GroupName),
				Organization: []string{fmt.Sprintf("%s:monitoring", garden.GroupName)},
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCAKubelet],
			},
		},

		// Secret definition for kubecfg
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: "kubecfg",

				CommonName:   "system:cluster-admin",
				Organization: []string{user.SystemPrivilegedGroup},
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
			},

			BasicAuth: basicAuthAPIServer,

			KubeConfigRequest: &secrets.KubeConfigRequest{
				ClusterName:  b.Shoot.SeedNamespace,
				APIServerURL: b.Shoot.ComputeAPIServerURL(false, false),
			},
		},

		// Secret definition for gardener
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: gardencorev1alpha1.SecretNameGardener,

				CommonName:   gardenv1beta1.GardenerName,
				Organization: []string{user.SystemPrivilegedGroup},
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
			},

			KubeConfigRequest: &secrets.KubeConfigRequest{
				ClusterName:  b.Shoot.SeedNamespace,
				APIServerURL: b.Shoot.ComputeAPIServerURL(false, true),
			},
		},

		// Secret definition for cloud-config-downloader
		&secrets.ControlPlaneSecretConfig{
			CertificateSecretConfig: &secrets.CertificateSecretConfig{
				Name: "cloud-config-downloader",

				CommonName:   "cloud-config-downloader",
				Organization: nil,
				DNSNames:     nil,
				IPAddresses:  nil,

				CertType:  secrets.ClientCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
			},

			KubeConfigRequest: &secrets.KubeConfigRequest{
				ClusterName:  b.Shoot.SeedNamespace,
				APIServerURL: b.Shoot.ComputeAPIServerURL(false, true),
			},
		},

		// Secret definition for monitoring
		&secrets.BasicAuthSecretConfig{
			Name:   "monitoring-ingress-credentials",
			Format: secrets.BasicAuthFormatNormal,

			Username:       "admin",
			PasswordLength: 32,
		},

		// Secret definition for ssh-keypair
		&secrets.RSASecretConfig{
			Name:       gardencorev1alpha1.SecretNameSSHKeyPair,
			Bits:       4096,
			UsedForSSH: true,
		},

		// Secret definition for service-account-key
		&secrets.RSASecretConfig{
			Name:       "service-account-key",
			Bits:       4096,
			UsedForSSH: false,
		},

		// Secret definition for vpn-shoot (OpenVPN server side)
		&secrets.CertificateSecretConfig{
			Name: "vpn-shoot",

			CommonName:   "vpn-shoot",
			Organization: nil,
			DNSNames:     []string{},
			IPAddresses:  []net.IP{},

			CertType:  secrets.ServerCert,
			SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
		},

		// Secret definition for vpn-seed (OpenVPN client side)
		&secrets.CertificateSecretConfig{
			Name: "vpn-seed",

			CommonName:   "vpn-seed",
			Organization: nil,
			DNSNames:     []string{},
			IPAddresses:  []net.IP{},

			CertType:  secrets.ClientCert,
			SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
		},

		// Secret definition for etcd server
		&secrets.CertificateSecretConfig{
			Name: certificateETCDServer,

			CommonName:   "etcd-server",
			Organization: nil,
			DNSNames:     etcdCertDNSNames,
			IPAddresses:  nil,

			CertType:  secrets.ServerClientCert,
			SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCAETCD],
		},

		// Secret definition for etcd server
		&secrets.CertificateSecretConfig{
			Name: certificateETCDClient,

			CommonName:   "etcd-client",
			Organization: nil,
			DNSNames:     nil,
			IPAddresses:  nil,

			CertType:  secrets.ClientCert,
			SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCAETCD],
		},

		// Secret definition for metrics-server
		&secrets.CertificateSecretConfig{
			Name: "metrics-server",

			CommonName:   "metrics-server",
			Organization: nil,
			DNSNames: []string{
				"metrics-server",
				fmt.Sprintf("metrics-server.%s", metav1.NamespaceSystem),
				fmt.Sprintf("metrics-server.%s.svc", metav1.NamespaceSystem),
			},
			IPAddresses: nil,

			CertType:  secrets.ServerClientCert,
			SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCAMetricsServer],
		},

		// Secret definition for alertmanager (ingress)
		&secrets.CertificateSecretConfig{
			Name: "alertmanager-tls",

			CommonName:   "alertmanager",
			Organization: []string{fmt.Sprintf("%s:monitoring:ingress", garden.GroupName)},
			DNSNames:     []string{alertManagerHost},
			IPAddresses:  nil,

			CertType:  secrets.ServerCert,
			SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
		},

		// Secret definition for grafana (ingress)
		&secrets.CertificateSecretConfig{
			Name: "grafana-tls",

			CommonName:   "grafana",
			Organization: []string{fmt.Sprintf("%s:monitoring:ingress", garden.GroupName)},
			DNSNames:     []string{grafanaHost},
			IPAddresses:  nil,

			CertType:  secrets.ServerCert,
			SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
		},

		// Secret definition for prometheus (ingress)
		&secrets.CertificateSecretConfig{
			Name: "prometheus-tls",

			CommonName:   "prometheus",
			Organization: []string{fmt.Sprintf("%s:monitoring:ingress", garden.GroupName)},
			DNSNames:     []string{prometheusHost},
			IPAddresses:  nil,

			CertType:  secrets.ServerCert,
			SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
		},
	}

	loggingEnabled := controllermanagerfeatures.FeatureGate.Enabled(features.Logging)
	if loggingEnabled {
		kibanaHost := b.Seed.GetIngressFQDN("k", b.Shoot.Info.Name, b.Garden.Project.Name)
		secretList = append(secretList,
			&secrets.CertificateSecretConfig{
				Name: "kibana-tls",

				CommonName:   "kibana",
				Organization: []string{fmt.Sprintf("%s:logging:ingress", garden.GroupName)},
				DNSNames:     []string{kibanaHost},
				IPAddresses:  nil,

				CertType:  secrets.ServerCert,
				SigningCA: certificateAuthorities[gardencorev1alpha1.SecretNameCACluster],
			},
			// Secret definition for logging
			&secrets.BasicAuthSecretConfig{
				Name:   "logging-ingress-credentials",
				Format: secrets.BasicAuthFormatNormal,

				Username:       "admin",
				PasswordLength: 32,
			},
		)
	}

	return secretList, nil
}

// DeploySecrets creates a CA certificate for the Shoot cluster and uses it to sign the server certificate
// used by the kube-apiserver, and all client certificates used for communcation. It also creates RSA key
// pairs for SSH connections to the nodes/VMs and for the VPN tunnel. Moreover, basic authentication
// credentials are computed which will be used to secure the Ingress resources and the kube-apiserver itself.
// Server certificates for the exposed monitoring endpoints (via Ingress) are generated as well.
func (b *Botanist) DeploySecrets() error {
	existingSecretsMap, err := b.fetchExistingSecrets()
	if err != nil {
		return err
	}

	if err := b.deleteOldETCDServerCertificate(existingSecretsMap); err != nil {
		return err
	}

	certificateAuthorities, err := b.generateCertificateAuthorities(existingSecretsMap)
	if err != nil {
		return err
	}

	basicAuthAPIServer, err := b.generateBasicAuthAPIServer(existingSecretsMap)
	if err != nil {
		return err
	}

	if err := b.deployOpenVPNTLSAuthSecret(existingSecretsMap); err != nil {
		return err
	}

	wantedSecretsList, err := b.generateWantedSecrets(basicAuthAPIServer, certificateAuthorities)
	if err != nil {
		return err
	}

	if err := b.generateShootSecrets(existingSecretsMap, wantedSecretsList); err != nil {
		return err
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()

	for name, secret := range b.Secrets {
		b.CheckSums[name] = computeSecretCheckSum(secret.Data)
	}

	return nil
}

// DeployCloudProviderSecret creates or updates the cloud provider secret in the Shoot namespace
// in the Seed cluster.
func (b *Botanist) DeployCloudProviderSecret() error {
	var (
		checksum = computeSecretCheckSum(b.Shoot.Secret.Data)
		secret   = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      gardencorev1alpha1.SecretNameCloudProvider,
				Namespace: b.Shoot.SeedNamespace,
				Annotations: map[string]string{
					"checksum/data": checksum,
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: b.Shoot.Secret.Data,
		}
	)

	if _, err := b.K8sSeedClient.CreateSecretObject(secret, true); err != nil {
		return err
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.Secrets[gardencorev1alpha1.SecretNameCloudProvider] = b.Shoot.Secret
	b.CheckSums[gardencorev1alpha1.SecretNameCloudProvider] = checksum

	return nil
}

// DeleteGardenSecrets deletes the Shoot-specific secrets from the project namespace in the Garden cluster.
// TODO: https://github.com/gardener/gardener/pull/353: This can be removed in a future version as we are now using owner
// references for the Garden secrets (also remove the actual invocation of the function in the deletion flow of a Shoot).
func (b *Botanist) DeleteGardenSecrets() error {
	if err := b.K8sGardenClient.DeleteSecret(b.Shoot.Info.Namespace, generateGardenSecretName(b.Shoot.Info.Name, "kubeconfig")); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := b.K8sGardenClient.DeleteSecret(b.Shoot.Info.Namespace, generateGardenSecretName(b.Shoot.Info.Name, gardencorev1alpha1.SecretNameSSHKeyPair)); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func (b *Botanist) fetchExistingSecrets() (map[string]*corev1.Secret, error) {
	secretList, err := b.K8sSeedClient.ListSecrets(b.Shoot.SeedNamespace, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	existingSecretsMap := make(map[string]*corev1.Secret, len(secretList.Items))
	for _, secret := range secretList.Items {
		secretObj := secret
		existingSecretsMap[secret.Name] = &secretObj
	}

	return existingSecretsMap, nil
}

// Delete the etcd server certificate if it has been generated by an old version
// of Gardener (not enough SANS).
func (b *Botanist) deleteOldETCDServerCertificate(existingSecretsMap map[string]*corev1.Secret) error {
	secret, ok := existingSecretsMap[certificateETCDServer]
	if !ok {
		return nil
	}

	certificate, err := secrets.LoadCertificate(certificateETCDServer, secret.Data[secrets.DataKeyPrivateKey], secret.Data[secrets.DataKeyCertificate])
	if err != nil {
		return err
	}

	if crt := certificate.Certificate; crt != nil {
		old := sets.NewString(crt.DNSNames...)
		new := sets.NewString(dnsNamesForEtcd(b.Shoot.SeedNamespace)...)

		if old.Equal(new) {
			return nil
		}
	}

	b.Logger.Infof("Will recreate secret %s", certificateETCDServer)
	if err := b.K8sSeedClient.DeleteSecret(b.Shoot.SeedNamespace, certificateETCDServer); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	delete(existingSecretsMap, certificateETCDServer)

	return nil
}

func (b *Botanist) generateCertificateAuthorities(existingSecretsMap map[string]*corev1.Secret) (map[string]*secrets.Certificate, error) {
	generatedSecrets, certificateAuthorities, err := secrets.GenerateCertificateAuthorities(b.K8sSeedClient, existingSecretsMap, wantedCertificateAuthorities, b.Shoot.SeedNamespace)
	if err != nil {
		return nil, err
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()

	for secretName, caSecret := range generatedSecrets {
		b.Secrets[secretName] = caSecret
	}

	return certificateAuthorities, nil
}

func (b *Botanist) generateBasicAuthAPIServer(existingSecretsMap map[string]*corev1.Secret) (*secrets.BasicAuth, error) {
	basicAuthSecretAPIServer := &secrets.BasicAuthSecretConfig{
		Name:           "kube-apiserver-basic-auth",
		Format:         secrets.BasicAuthFormatCSV,
		Username:       "admin",
		PasswordLength: 32,
	}

	if existingSecret, ok := existingSecretsMap[basicAuthSecretAPIServer.Name]; ok {
		basicAuth, err := secrets.LoadBasicAuthFromCSV(basicAuthSecretAPIServer.Name, existingSecret.Data[secrets.DataKeyCSV])
		if err != nil {
			return nil, err
		}

		b.mutex.Lock()
		defer b.mutex.Unlock()

		b.Secrets[basicAuthSecretAPIServer.Name] = existingSecret

		return basicAuth, nil
	}

	basicAuth, err := basicAuthSecretAPIServer.Generate()
	if err != nil {
		return nil, err
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.Secrets[basicAuthSecretAPIServer.Name], err = b.K8sSeedClient.CreateSecret(b.Shoot.SeedNamespace, basicAuthSecretAPIServer.Name, corev1.SecretTypeOpaque, basicAuth.SecretData(), false)
	if err != nil {
		return nil, err
	}

	return basicAuth.(*secrets.BasicAuth), nil
}

func (b *Botanist) generateShootSecrets(existingSecretsMap map[string]*corev1.Secret, wantedSecretsList []secrets.ConfigInterface) error {
	deployedClusterSecrets, err := secrets.GenerateClusterSecrets(b.K8sSeedClient, existingSecretsMap, wantedSecretsList, b.Shoot.SeedNamespace)
	if err != nil {
		return err
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()

	for secretName, secret := range deployedClusterSecrets {
		b.Secrets[secretName] = secret
	}

	return nil
}

// SyncShootCredentialsToGarden copies the kubeconfig generated for the user as well as the SSH keypair to
// the project namespace in the Garden cluster.
func (b *Botanist) SyncShootCredentialsToGarden() error {
	for key, value := range map[string]string{"kubeconfig": "kubecfg", gardencorev1alpha1.SecretNameSSHKeyPair: gardencorev1alpha1.SecretNameSSHKeyPair} {
		secretObj := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s.%s", b.Shoot.Info.Name, key),
				Namespace: b.Shoot.Info.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(b.Shoot.Info, gardenv1beta1.SchemeGroupVersion.WithKind("Shoot")),
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: b.Secrets[value].Data,
		}
		if _, err := b.K8sGardenClient.CreateSecretObject(secretObj, true); err != nil {
			return err
		}
	}

	return nil
}

func (b *Botanist) deployOpenVPNTLSAuthSecret(existingSecretsMap map[string]*corev1.Secret) error {
	name := "vpn-seed-tlsauth"
	if tlsAuthSecret, ok := existingSecretsMap[name]; ok {
		b.mutex.Lock()
		defer b.mutex.Unlock()

		b.Secrets[name] = tlsAuthSecret
		return nil
	}

	tlsAuthKey, err := generateOpenVPNTLSAuth()
	if err != nil {
		return fmt.Errorf("error while creating openvpn tls auth secret: %v", err)
	}

	data := map[string][]byte{
		"vpn.tlsauth": tlsAuthKey,
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.Secrets[name], err = b.K8sSeedClient.CreateSecret(b.Shoot.SeedNamespace, name, corev1.SecretTypeOpaque, data, false)
	return err
}

func generateOpenVPNTLSAuth() ([]byte, error) {
	var (
		out bytes.Buffer
		cmd = exec.Command("openvpn", "--genkey", "--secret", "/dev/stdout")
	)

	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func computeSecretCheckSum(data map[string][]byte) string {
	jsonString, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return utils.ComputeSHA256Hex(jsonString)
}

func generateGardenSecretName(shootName, secretName string) string {
	return fmt.Sprintf("%s.%s", shootName, secretName)
}

func dnsNamesForService(name, namespace string) []string {
	return []string{
		name,
		fmt.Sprintf("%s.%s", name, namespace),
		fmt.Sprintf("%s.%s.svc", name, namespace),
		fmt.Sprintf("%s.%s.svc.%s", name, namespace, gardenv1beta1.DefaultDomain),
	}
}

func dnsNamesForEtcd(namespace string) []string {
	names := []string{
		fmt.Sprintf("%s-0", common.EtcdMainStatefulSetName),
		fmt.Sprintf("%s-0", common.EtcdEventsStatefulSetName),
	}
	names = append(names, dnsNamesForService(fmt.Sprintf("%s-client", common.EtcdMainStatefulSetName), namespace)...)
	names = append(names, dnsNamesForService(fmt.Sprintf("%s-client", common.EtcdEventsStatefulSetName), namespace)...)
	return names
}
