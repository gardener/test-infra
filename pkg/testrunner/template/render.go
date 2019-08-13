package template

import (
	"fmt"
	"github.com/gardener/gardener/pkg/chartrenderer"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s.io/helm/pkg/strvals"
)

// RenderChart renders the provided helm chart with testruns, adds the testrun parameters and returns the templated files.
func RenderChart(tmClient kubernetes.Interface, parameters *ShootTestrunParameters, versions []string) ([]*TestrunFile, error) {
	log.Debugf("Parameters: %+v", util.PrettyPrintStruct(parameters))
	log.Debugf("RenderShootTestrun chart from %s", parameters.TestrunChartPath)

	tmChartRenderer, err := chartrenderer.NewForConfig(tmClient.RESTConfig())
	if err != nil {
		return nil, fmt.Errorf("Cannot create chartrenderer for gardener: %s", err.Error())
	}

	gardenKubeconfig, err := ioutil.ReadFile(parameters.GardenKubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("Cannot read gardener kubeconfig %s, Error: %s", parameters.GardenKubeconfigPath, err.Error())
	}

	renderedFiles := []*TestrunFile{}
	for _, version := range versions {
		files, err := RenderSingleChart(tmChartRenderer, parameters, gardenKubeconfig, version)
		if err != nil {
			return nil, err
		}
		renderedFiles = append(renderedFiles, files...)
	}
	return renderedFiles, nil
}

func RenderSingleChart(renderer chartrenderer.Interface, parameters *ShootTestrunParameters, gardenKubeconfig []byte, version string) ([]*TestrunFile, error) {
	values := map[string]interface{}{
		"shoot": map[string]interface{}{
			"name":                 fmt.Sprintf("%s-%s", parameters.ShootName, util.RandomString(5)),
			"projectNamespace":     fmt.Sprintf("garden-%s", parameters.ProjectName),
			"cloudprovider":        parameters.Cloudprovider,
			"cloudprofile":         parameters.Cloudprofile,
			"secretBinding":        parameters.SecretBinding,
			"region":               parameters.Region,
			"zone":                 parameters.Zone,
			"k8sVersion":           version,
			"machinetype":          parameters.MachineType,
			"autoscalerMin":        parameters.AutoscalerMin,
			"autoscalerMax":        parameters.AutoscalerMax,
			"floatingPoolName":     parameters.FloatingPoolName,
			"loadbalancerProvider": parameters.LoadBalancerProvider,
		},
		"gardener": map[string]interface{}{
			"version": parameters.GardenerVersion,
		},
		"kubeconfigs": map[string]interface{}{
			"gardener": string(gardenKubeconfig),
		},
	}

	values, err := determineValues(values, parameters.SetValues, parameters.FileValues)
	if err != nil {
		return nil, err
	}
	log.Debugf("Values: \n%s \n", util.PrettyPrintStruct(values))

	chart, err := renderer.Render(parameters.TestrunChartPath, "", parameters.Namespace, values)
	if err != nil {
		return nil, err
	}

	return ParseTestrunChart(chart, TestrunFileMetadata{
		KubernetesVersion: version,
	}), nil
}

// determineValues fetches values from all specified files and set values.
// The values are merged with the defaultValues whereas file values overwrite default values and set values overwrite file values.
func determineValues(defaultValues map[string]interface{}, setValues string, fileValues []string) (map[string]interface{}, error) {
	newFileValues, err := readFileValues(fileValues)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read values from file")
	}
	defaultValues = utils.MergeMaps(defaultValues, newFileValues)
	newSetValues, err := strvals.ParseString(setValues)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse set values")
	}
	defaultValues = utils.MergeMaps(defaultValues, newSetValues)

	return defaultValues, nil
}

func ParseTestrunChart(chart *chartrenderer.RenderedChart, metadata TestrunFileMetadata) []*TestrunFile {
	files := make([]*TestrunFile, 0)
	for _, file := range chart.Files() {
		files = append(files, &TestrunFile{
			File:     file,
			Metadata: metadata,
		})
	}
	return files
}
