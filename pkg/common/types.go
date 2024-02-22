// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package common

// CloudProvider is a string alias.
type CloudProvider string

const (
	// CloudProviderAWS is a constant for the AWS cloud provider.
	CloudProviderAWS CloudProvider = "aws"
	// CloudProviderAzure is a constant for the Azure cloud provider.
	CloudProviderAzure CloudProvider = "azure"
	// CloudProviderGCP is a constant for the GCP cloud provider.
	CloudProviderGCP CloudProvider = "gcp"
	// CloudProviderOpenStack is a constant for the OpenStack cloud provider.
	CloudProviderOpenStack CloudProvider = "openstack"
	// CloudProviderAlicloud is a constant for the Alibaba cloud provider.
	CloudProviderAlicloud CloudProvider = "alicloud"
	// CloudProviderPacket is a constant for the Packet cloud provider.
	CloudProviderPacket CloudProvider = "packet"
)
