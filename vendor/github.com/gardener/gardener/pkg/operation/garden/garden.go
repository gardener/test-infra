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

package garden

import (
	"context"
	"fmt"
	"strings"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	gardencoreinformers "github.com/gardener/gardener/pkg/client/core/informers/externalversions"
	gardencorelisters "github.com/gardener/gardener/pkg/client/core/listers/core/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/logger"
	"github.com/gardener/gardener/pkg/operation/common"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	secretutils "github.com/gardener/gardener/pkg/utils/secrets"
	"github.com/gardener/gardener/pkg/utils/version"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kubeinformers "k8s.io/client-go/informers"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// New creates a new Garden object (based on a Shoot object).
func New(projectLister gardencorelisters.ProjectLister, namespace string, secrets map[string]*corev1.Secret) (*Garden, error) {
	project, err := common.ProjectForNamespace(projectLister, namespace)
	if err != nil {
		return nil, err
	}

	internalDomain, err := GetInternalDomain(secrets)
	if err != nil {
		return nil, err
	}

	defaultDomains, err := GetDefaultDomains(secrets)
	if err != nil {
		return nil, err
	}

	return &Garden{
		Project:        project,
		InternalDomain: internalDomain,
		DefaultDomains: defaultDomains,
	}, nil
}

// GetDefaultDomains finds all the default domain secrets within the given map and returns a list of
// objects that contains all relevant information about the default domains.
func GetDefaultDomains(secrets map[string]*corev1.Secret) ([]*Domain, error) {
	var defaultDomains []*Domain

	for key, secret := range secrets {
		if strings.HasPrefix(key, common.GardenRoleDefaultDomain) {
			domain, err := constructDomainFromSecret(secret)
			if err != nil {
				return nil, fmt.Errorf("error getting information out of default domain secret: %+v", err)
			}
			defaultDomains = append(defaultDomains, domain)
		}
	}

	return defaultDomains, nil
}

// GetInternalDomain finds the internal domain secret within the given map and returns the object
// that contains all relevant information about the internal domain.
func GetInternalDomain(secrets map[string]*corev1.Secret) (*Domain, error) {
	internalDomainSecret, ok := secrets[common.GardenRoleInternalDomain]
	if !ok {
		return nil, nil
	}

	return constructDomainFromSecret(internalDomainSecret)
}

func constructDomainFromSecret(secret *corev1.Secret) (*Domain, error) {
	provider, domain, includeZones, excludeZones, err := common.GetDomainInfoFromAnnotations(secret.Annotations)
	if err != nil {
		return nil, err
	}

	return &Domain{
		Domain:       domain,
		Provider:     provider,
		SecretData:   secret.Data,
		IncludeZones: includeZones,
		ExcludeZones: excludeZones,
	}, nil
}

// DomainIsDefaultDomain identifies whether the given domain is a default domain.
func DomainIsDefaultDomain(domain string, defaultDomains []*Domain) *Domain {
	for _, defaultDomain := range defaultDomains {
		if strings.HasSuffix(domain, "."+defaultDomain.Domain) {
			return defaultDomain
		}
	}
	return nil
}

// ReadGardenSecrets reads the Kubernetes Secrets from the Garden cluster which are independent of Shoot clusters.
// The Secret objects are stored on the Controller in order to pass them to created Garden objects later.
func ReadGardenSecrets(k8sInformers kubeinformers.SharedInformerFactory, k8sGardenCoreInformers gardencoreinformers.SharedInformerFactory) (map[string]*corev1.Secret, error) {
	var (
		secretsMap                          = make(map[string]*corev1.Secret)
		numberOfInternalDomainSecrets       = 0
		numberOfOpenVPNDiffieHellmanSecrets = 0
		numberOfAlertingSecrets             = 0
	)

	selector, err := labels.Parse(v1beta1constants.DeprecatedGardenRole)
	if err != nil {
		return nil, err
	}
	secrets, err := k8sInformers.Core().V1().Secrets().Lister().Secrets(v1beta1constants.GardenNamespace).List(selector)
	if err != nil {
		return nil, err
	}

	for _, obj := range secrets {
		secret := obj.DeepCopy()

		// Retrieving default domain secrets based on all secrets in the Garden namespace which have
		// a label indicating the Garden role default-domain.
		if secret.Labels[v1beta1constants.DeprecatedGardenRole] == common.GardenRoleDefaultDomain {
			_, domain, _, _, err := common.GetDomainInfoFromAnnotations(secret.Annotations)
			if err != nil {
				logger.Logger.Warnf("error getting information out of default domain secret %s: %+v", secret.Name, err)
				continue
			}
			defaultDomainSecret := secret
			secretsMap[fmt.Sprintf("%s-%s", common.GardenRoleDefaultDomain, domain)] = defaultDomainSecret
			logger.Logger.Infof("Found default domain secret %s for domain %s.", secret.Name, domain)
		}

		// Retrieving internal domain secrets based on all secrets in the Garden namespace which have
		// a label indicating the Garden role internal-domain.
		if secret.Labels[v1beta1constants.DeprecatedGardenRole] == common.GardenRoleInternalDomain {
			_, domain, _, _, err := common.GetDomainInfoFromAnnotations(secret.Annotations)
			if err != nil {
				logger.Logger.Warnf("error getting information out of internal domain secret %s: %+v", secret.Name, err)
				continue
			}
			internalDomainSecret := secret
			secretsMap[common.GardenRoleInternalDomain] = internalDomainSecret
			logger.Logger.Infof("Found internal domain secret %s for domain %s.", secret.Name, domain)
			numberOfInternalDomainSecrets++
		}

		// Retrieving Diffie-Hellman secret for OpenVPN based on all secrets in the Garden namespace which have
		// a label indicating the Garden role openvpn-diffie-hellman.
		if secret.Labels[v1beta1constants.DeprecatedGardenRole] == common.GardenRoleOpenVPNDiffieHellman {
			openvpnDiffieHellman := secret
			key := "dh2048.pem"
			if _, ok := secret.Data[key]; !ok {
				return nil, fmt.Errorf("cannot use OpenVPN Diffie Hellman secret '%s' as it does not contain key '%s' (whose value should be the actual Diffie Hellman key)", secret.Name, key)
			}
			secretsMap[common.GardenRoleOpenVPNDiffieHellman] = openvpnDiffieHellman
			logger.Logger.Infof("Found OpenVPN Diffie Hellman secret %s.", secret.Name)
			numberOfOpenVPNDiffieHellmanSecrets++
		}

		// Retrieving basic auth secret for aggregate monitoring with a label
		// indicating the Garden role global-monitoring.
		if secret.Labels[v1beta1constants.DeprecatedGardenRole] == common.GardenRoleGlobalMonitoring {
			monitoringSecret := secret
			secretsMap[common.GardenRoleGlobalMonitoring] = monitoringSecret
			logger.Logger.Infof("Found monitoring basic auth secret %s.", secret.Name)
		}
	}

	selectorGardenRole, err := labels.Parse(v1beta1constants.GardenRole)
	if err != nil {
		return nil, err
	}

	secretsGardenRole, err := k8sInformers.Core().V1().Secrets().Lister().Secrets(v1beta1constants.GardenNamespace).List(selectorGardenRole)
	if err != nil {
		return nil, err
	}

	for _, secret := range secretsGardenRole {

		// Retrieve the alerting secret to configure alerting. Either in cluster email alerting or
		// external alertmanager configuration.
		if secret.Labels[v1beta1constants.GardenRole] == common.GardenRoleAlerting {
			authType := string(secret.Data["auth_type"])
			if authType != "smtp" && authType != "none" && authType != "basic" && authType != "certificate" {
				return nil, fmt.Errorf("invalid or missing field 'auth_type' in secret %s", secret.Name)
			}
			alertingSecret := secret
			secretsMap[common.GardenRoleAlerting] = alertingSecret
			logger.Logger.Infof("Found alerting secret %s.", secret.Name)
			numberOfAlertingSecrets++
		}
	}

	// Check if an internal domain secret is required
	seeds, err := k8sGardenCoreInformers.Core().V1beta1().Seeds().Lister().List(labels.Everything())
	if err != nil {
		return nil, err
	}
	for _, seed := range seeds {
		if gardencorev1beta1helper.TaintsHave(seed.Spec.Taints, gardencorev1beta1.SeedTaintDisableDNS) {
			continue
		}

		// For each Shoot we create a LoadBalancer(LB) pointing to the API server of the Shoot. Because the technical address
		// of the LB (ip or hostname) can change we cannot directly write it into the kubeconfig of the components
		// which talk from outside (kube-proxy, kubelet etc.) (otherwise those kubeconfigs would be broken once ip/hostname
		// of LB changed; and we don't have means to exchange kubeconfigs currently).
		// Therefore, to have a stable endpoint, we create a DNS record pointing to the ip/hostname of the LB. This DNS record
		// is used in all kubeconfigs. With that we have a robust endpoint stable against underlying ip/hostname changes.
		// And there can only be one of this internal domain secret because otherwise the gardener would not know which
		// domain it should use.
		if numberOfInternalDomainSecrets != 1 {
			return nil, fmt.Errorf("require exactly ONE internal domain secret, but found %d", numberOfInternalDomainSecrets)
		}
	}

	// The VPN bridge from a Shoot's control plane running in the Seed cluster to the worker nodes of the Shoots is based
	// on OpenVPN. It requires a Diffie Hellman key. If no such key is explicitly provided as secret in the garden namespace
	// then the Gardener will use a default one (not recommended, but useful for local development). If a secret is specified
	// its key will be used for all Shoots. However, at most only one of such a secret is allowed to be specified (otherwise,
	// the Gardener cannot determine which to choose).
	if numberOfOpenVPNDiffieHellmanSecrets > 1 {
		return nil, fmt.Errorf("can only accept at most one OpenVPN Diffie Hellman secret, but found %d", numberOfOpenVPNDiffieHellmanSecrets)
	}

	// Operators can configure gardener to send email alerts or send the alerts to an external alertmanager. If no configuration
	// is provided then no alerts will be sent.
	if numberOfAlertingSecrets > 1 {
		return nil, fmt.Errorf("can only accept at most one alerting secret, but found %d", numberOfAlertingSecrets)
	}

	return secretsMap, nil
}

// VerifyInternalDomainSecret verifies that the internal domain secret matches to the internal domain secret used for
// existing Shoot clusters. It is not allowed to change the internal domain secret if there are existing Shoot clusters.
func VerifyInternalDomainSecret(k8sGardenClient kubernetes.Interface, numberOfShoots int, internalDomainSecret *corev1.Secret) error {
	_, currentDomain, _, _, err := common.GetDomainInfoFromAnnotations(internalDomainSecret.Annotations)
	if err != nil {
		return fmt.Errorf("error getting information out of current internal domain secret: %+v", err)
	}

	internalConfigMap := &corev1.ConfigMap{}
	err = k8sGardenClient.Client().Get(context.TODO(), kutil.Key(v1beta1constants.GardenNamespace, common.ControllerManagerInternalConfigMapName), internalConfigMap)
	if apierrors.IsNotFound(err) || numberOfShoots == 0 {
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      common.ControllerManagerInternalConfigMapName,
				Namespace: v1beta1constants.GardenNamespace,
			},
		}

		_, err := controllerutil.CreateOrUpdate(context.TODO(), k8sGardenClient.Client(), configMap, func() error {
			configMap.Data = map[string]string{
				common.GardenRoleInternalDomain: currentDomain,
			}
			return nil
		})
		return err
	}
	if err != nil {
		return err
	}

	oldDomain := internalConfigMap.Data[common.GardenRoleInternalDomain]
	if oldDomain != currentDomain {
		return fmt.Errorf("cannot change internal domain from '%s' to '%s' unless there are no more Shoots", oldDomain, currentDomain)
	}

	return nil
}

// BootstrapCluster bootstraps the Garden cluster and deploys various required manifests.
func BootstrapCluster(k8sGardenClient kubernetes.Interface, gardenNamespace string, secrets map[string]*corev1.Secret) error {
	// Check whether the Kubernetes version of the Garden cluster is at least 1.10 (least supported K8s version of Gardener).
	minGardenVersion := "1.10"
	gardenVersionOK, err := version.CompareVersions(k8sGardenClient.Version(), ">=", minGardenVersion)
	if err != nil {
		return err
	}
	if !gardenVersionOK {
		return fmt.Errorf("the Kubernetes version of the Garden cluster must be at least %s", minGardenVersion)
	}
	if secrets[common.GardenRoleGlobalMonitoring] == nil {
		var secret *corev1.Secret
		if secret, err = generateMonitoringSecret(k8sGardenClient, gardenNamespace); err != nil {
			return err
		}
		secrets[common.GardenRoleGlobalMonitoring] = secret
	}

	return nil
}

func generateMonitoringSecret(k8sGardenClient kubernetes.Interface, gardenNamespace string) (*corev1.Secret, error) {
	basicAuthSecret := &secretutils.BasicAuthSecretConfig{
		Name:   "monitoring-ingress-credentials",
		Format: secretutils.BasicAuthFormatNormal,

		Username:       "admin",
		PasswordLength: 32,
	}
	basicAuth, err := basicAuthSecret.Generate()
	if err != nil {
		return nil, err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      basicAuthSecret.Name,
			Namespace: gardenNamespace,
		},
	}
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), k8sGardenClient.Client(), secret, func() error {
		secret.Labels = map[string]string{
			v1beta1constants.DeprecatedGardenRole: common.GardenRoleGlobalMonitoring,
		}
		secret.Type = corev1.SecretTypeOpaque
		secret.Data = basicAuth.SecretData()
		return nil
	}); err != nil {
		return nil, err
	}
	return secret, nil
}
