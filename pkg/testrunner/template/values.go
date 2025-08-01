// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"fmt"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/shootflavors"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/pkg/util"
)

func NewDefaultValueRenderer(parameters *internalParameters) ValueRenderer {
	return &defaultValueRenderer{
		parameters: parameters,
	}
}

type defaultValueRenderer struct {
	parameters *internalParameters
}

func (r *defaultValueRenderer) GetValues(defaultValues map[string]interface{}) (map[string]interface{}, error) {
	values := map[string]interface{}{
		"gardener": map[string]interface{}{
			"version": r.parameters.GardenerVersion,
		},
		"kubeconfigs": map[string]interface{}{
			"gardener": string(r.parameters.GardenerKubeconfig),
		},
	}
	return utils.MergeMaps(defaultValues, values), nil
}

func (r *defaultValueRenderer) Render(defaultValues map[string]interface{}) (map[string]interface{}, *metadata.Metadata, interface{}, error) {
	values, err := r.GetValues(defaultValues)
	if err != nil {
		return nil, nil, nil, err
	}
	metadata, err := r.GetMetadata()
	if err != nil {
		return nil, nil, nil, err
	}
	return values, metadata, nil, nil
}

func (r *defaultValueRenderer) GetMetadata() (*metadata.Metadata, error) {
	return &metadata.Metadata{
		Landscape:           r.parameters.Landscape,
		ComponentDescriptor: r.parameters.ComponentDescriptor.JSON(),
	}, nil
}

func NewShootValueRenderer(log logr.Logger, shoot *shootflavors.ExtendedFlavorInstance, parameters *internalParameters) ValueRenderer {
	return &shootValueRenderer{
		log:        log,
		shoot:      shoot,
		parameters: parameters,
	}
}

type shootValueRenderer struct {
	log   logr.Logger
	shoot *shootflavors.ExtendedFlavorInstance

	parameters *internalParameters
}

func (r *shootValueRenderer) Render(defaultValues map[string]interface{}) (map[string]interface{}, *metadata.Metadata, interface{}, error) {
	shoot := r.shoot.New()
	values, err := r.GetValues(shoot, defaultValues)
	if err != nil {
		return nil, nil, nil, err
	}
	metadata, err := r.GetMetadata(shoot)
	if err != nil {
		return nil, nil, nil, err
	}
	return values, metadata, shoot, nil
}

func (r *shootValueRenderer) GetValues(shoot *common.ExtendedShoot, defaultValues map[string]interface{}) (map[string]interface{}, error) {
	workers, err := encodeRawObject(shoot.Workers)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse worker config")
	}
	r.log.V(3).Info(fmt.Sprintf("Workers: \n%s \n", util.PrettyPrintStruct(workers)))

	infrastructure, err := encodeRawObject(shoot.InfrastructureConfig)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse infrastructure config")
	}
	r.log.V(3).Info(fmt.Sprintf("Infrastructure: \n%s \n", util.PrettyPrintStruct(infrastructure)))

	networkingConfig, err := encodeRawObject(shoot.NetworkingConfig)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse networking config")
	}
	r.log.V(3).Info(fmt.Sprintf("networking: \n%s \n", util.PrettyPrintStruct(networkingConfig)))

	controlplane, err := encodeRawObject(shoot.ControlPlaneConfig)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse infrastructure config")
	}
	r.log.V(3).Info(fmt.Sprintf("Controlplane: \n%s \n", util.PrettyPrintStruct(controlplane)))

	prevPrePatchVersion, prevPatchVersion, err := util.GetPreviousKubernetesVersions(shoot.Cloudprofile, shoot.KubernetesVersion)
	if err != nil {
		r.log.Info("unable to get previous versions", "error", err.Error())
	}

	var (
		hasInPlaceWorker                                  bool
		currentMachineImage                               gardencorev1beta1.ShootMachineImage
		prevVersionThatCanBeInPlaceUpdatedToCurrent, arch string
	)

	for _, worker := range shoot.Workers {
		if v1beta1helper.IsUpdateStrategyInPlace(worker.UpdateStrategy) {
			hasInPlaceWorker = true
			arch = *worker.Machine.Architecture
			currentMachineImage = *worker.Machine.Image
			break
		}
	}

	if hasInPlaceWorker {
		prevVersionThatCanBeInPlaceUpdatedToCurrent, err = util.GetLatestPreviousVersionForInPlaceUpdate(shoot.Cloudprofile, currentMachineImage, arch)
		if err != nil {
			r.log.Info("unable to get previous version", "error", err.Error())
		}
	}

	values := map[string]interface{}{
		"shoot": map[string]interface{}{
			"name":                         shoot.Name,
			"projectNamespace":             shoot.Namespace,
			"cloudprovider":                shoot.Provider,
			"cloudprofile":                 shoot.CloudprofileName,
			"secretBinding":                shoot.SecretBinding,
			"credentialsBinding":           shoot.CredentialsBinding,
			"region":                       shoot.Region,
			"zone":                         shoot.Zone,
			"workers":                      workers,
			"k8sVersion":                   shoot.KubernetesVersion.Version,
			"k8sPrevPrePatchVersion":       prevPrePatchVersion.Version,
			"k8sPrevPatchVersion":          prevPatchVersion.Version,
			"controlPlaneFailureTolerance": shoot.ControlPlaneFailureTolerance,
			"floatingPoolName":             shoot.FloatingPoolName,
			"networkingType":               shoot.NetworkingType,
			"ipFamilies":                   shoot.IpFamilies,
			"loadbalancerProvider":         shoot.LoadbalancerProvider,
			"infrastructureConfig":         infrastructure,
			"networkingConfig":             networkingConfig,
			"controlplaneConfig":           controlplane,
		},
		"gardener": map[string]interface{}{
			"version": r.parameters.GardenerVersion,
		},
		"kubeconfigs": map[string]interface{}{
			"gardener": string(r.parameters.GardenerKubeconfig),
		},
	}
	if shoot.AdditionalAnnotations != nil {
		values["shoot"].(map[string]interface{})["shootAnnotations"] = util.MarshalMap(shoot.AdditionalAnnotations)
	}
	if hasInPlaceWorker {
		values["shoot"].(map[string]any)["machineImagePrevVersion"] = prevVersionThatCanBeInPlaceUpdatedToCurrent
		values["shoot"].(map[string]any)["machine"] = map[string]any{
			"imageversion": *currentMachineImage.Version,
		}
	}
	return utils.MergeMaps(defaultValues, values), nil
}

func (r *shootValueRenderer) GetMetadata(shoot *common.ExtendedShoot) (*metadata.Metadata, error) {
	operatingsystemversion := "latest"
	if shoot.Workers[0].Machine.Image.Version != nil {
		operatingsystemversion = *shoot.Workers[0].Machine.Image.Version
	}
	containerRuntime := ""
	if shoot.Workers[0].CRI != nil {
		containerRuntime = string(shoot.Workers[0].CRI.Name)
	}
	return &metadata.Metadata{
		FlavorDescription:      shoot.Description,
		Landscape:              r.parameters.Landscape,
		ComponentDescriptor:    r.parameters.ComponentDescriptor.JSON(),
		CloudProvider:          string(shoot.Provider),
		KubernetesVersion:      shoot.KubernetesVersion.Version,
		Region:                 shoot.Region,
		Zone:                   shoot.Zone,
		OperatingSystem:        shoot.Workers[0].Machine.Image.Name, // todo: check if there a possible multiple workerpools with different images
		OperatingSystemVersion: operatingsystemversion,
		ContainerRuntime:       containerRuntime,
		Annotations:            shoot.AdditionalAnnotations,
	}, nil
}
