package template

import (
	"fmt"
	"io/ioutil"

	"github.com/gardener/gardener/pkg/chartrenderer"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/util"
	log "github.com/sirupsen/logrus"
)

// RenderChart renders the provided helm chart with testruns, adds the testrun parameters and returns the templated files.
func RenderChart(tmClient kubernetes.Interface, parameters *TestrunParameters, versions []string) ([]string, error) {
	log.Debugf("Parameters: %+v", util.PrettyPrintStruct(parameters))
	log.Debugf("Render chart from %s", parameters.TestrunChartPath)

	tmChartRenderer, err := chartrenderer.New(tmClient.Kubernetes())
	if err != nil {
		return nil, fmt.Errorf("Cannot create chartrenderer for gardener: %s", err.Error())
	}

	gardenKubeconfig, err := ioutil.ReadFile(parameters.GardenKubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("Cannot read gardener kubeconfig %s, Error: %s", parameters.GardenKubeconfigPath, err.Error())
	}

	renderedFiles := []string{}
	for _, version := range versions {
		files, err := renderSingleChart(tmChartRenderer, parameters, gardenKubeconfig, version)
		if err != nil {
			return nil, err
		}
		renderedFiles = append(renderedFiles, files...)
	}
	return renderedFiles, nil
}

func renderSingleChart(renderer chartrenderer.ChartRenderer, parameters *TestrunParameters, gardenKubeconfig []byte, version string) ([]string, error) {
	chart, err := renderer.Render(parameters.TestrunChartPath, "", parameters.Namespace, map[string]interface{}{
		"shoot": map[string]interface{}{
			"name":             fmt.Sprintf("%s-%s", parameters.ShootName, util.RandomString(5)),
			"projectNamespace": fmt.Sprintf("garden-%s", parameters.ProjectName),
			"cloudprovider":    parameters.Cloudprovider,
			"cloudprofile":     parameters.Cloudprofile,
			"secretBinding":    parameters.SecretBinding,
			"region":           parameters.Region,
			"zone":             parameters.Zone,
			"k8sVersion":       version,
			"machinetype":      parameters.MachineType,
			"autoscalerMin":    parameters.AutoscalerMin,
			"autoscalerMax":    parameters.AutoscalerMax,
			"floatingPoolName": parameters.FloatingPoolName,
		},
		"kubeconfigs": map[string]interface{}{
			"gardener": string(gardenKubeconfig),
		},
	})

	if err != nil {
		return nil, err
	}

	files := []string{}
	for _, file := range chart.Files {
		files = append(files, file)
	}

	return files, nil
}
