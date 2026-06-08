// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"encoding/json"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
)

// ArchsByVersion maps a machine-image version to its supported architectures.
type ArchsByVersion = map[string][]string

// ArchsByImage maps a machine-image name to its per-version architectures.
type ArchsByImage = map[string]ArchsByVersion

// BuildCapabilityProviderConfig builds a *runtime.RawExtension carrying
// machine-image architecture data in the capability-flavor format. It is
// exposed for cross-package tests that need to populate
// cloudprofile.Spec.ProviderConfig in the schema this package consumes.
//
// One capability flavor is emitted per version, listing all archs.
//
// This helper is test-only: it asserts via Gomega that JSON marshaling
// succeeds, so it must be called from within a Ginkgo spec.
func BuildCapabilityProviderConfig(imagesByName ArchsByImage) *runtime.RawExtension {
	images := make([]providerConfigMachineImage, 0, len(imagesByName))
	for imageName, archsByVersion := range imagesByName {
		versions := make([]providerConfigMachineImageVersion, 0, len(archsByVersion))
		for version, archs := range archsByVersion {
			versions = append(versions, providerConfigMachineImageVersion{
				Version: version,
				CapabilityFlavors: []providerConfigCapabilityFlavor{
					{Capabilities: providerConfigCapabilities{Architecture: archs}},
				},
			})
		}
		images = append(images, providerConfigMachineImage{Name: imageName, Versions: versions})
	}
	raw, err := json.Marshal(providerConfigMachineImages{MachineImages: images})
	Expect(err).ToNot(HaveOccurred())
	return &runtime.RawExtension{Raw: raw}
}
