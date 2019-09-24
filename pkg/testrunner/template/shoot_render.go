package template

import (
	"fmt"
	"github.com/gardener/gardener/pkg/chartrenderer"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"io/ioutil"
	"k8s.io/helm/pkg/strvals"
)

// RenderChart renders the provided helm chart with testruns, adds the testrun parameters and returns the templated files.
func RenderChart(log logr.Logger, tmClient kubernetes.Interface, parameters *ShootTestrunParameters, versions []string) ([]*RenderedTestrun, error) {
	log.V(3).Info(fmt.Sprintf("Parameters: %+v", util.PrettyPrintStruct(parameters)))
	log.V(3).Info("RenderShootTestruns chart", "chart", parameters.TestrunChartPath)

	tmChartRenderer, err := chartrenderer.NewForConfig(tmClient.RESTConfig())
	if err != nil {
		return nil, errors.Wrap(err, "cannot create chartrenderer for gardener")
	}

	gardenKubeconfig, err := ioutil.ReadFile(parameters.GardenKubeconfigPath)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot read gardener kubeconfig %s", parameters.GardenKubeconfigPath)
	}

	renderedFiles := []*RenderedTestrun{}
	for _, version := range versions {
		files, err := RenderSingleChart(log, tmChartRenderer, parameters, gardenKubeconfig, version)
		if err != nil {
			return nil, err
		}
		renderedFiles = append(renderedFiles, files...)
	}
	return renderedFiles, nil
}

func RenderSingleChart(log logr.Logger, renderer chartrenderer.Interface, parameters *ShootTestrunParameters, gardenKubeconfig []byte, version string) ([]*RenderedTestrun, error) {
	newParameters := *parameters
	newParameters.ShootName = fmt.Sprintf("%s-%s", parameters.ShootName, util.RandomString(5))
	newParameters.Namespace = fmt.Sprintf("garden-%s", parameters.ProjectName)
	values := map[string]interface{}{
		"shoot": map[string]interface{}{
			"name":             newParameters.ShootName,
			"projectNamespace": newParameters.Namespace,
			"cloudprovider":    parameters.Cloudprovider,
			"cloudprofile":     parameters.Cloudprofile,
			"secretBinding":    parameters.SecretBinding,
			"region":           parameters.Region,
			"zone":             parameters.Zone,
			"k8sVersion":       version,
			"machine": map[string]interface{}{
				"type":         parameters.MachineType,
				"image":        parameters.MachineImage,
				"imageVersion": parameters.MachineImageVersion,
			},
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
	log.V(3).Info(fmt.Sprintf("Values: \n%s \n", util.PrettyPrintStruct(values)))

	chart, err := renderer.Render(parameters.TestrunChartPath, "", parameters.Namespace, values)
	if err != nil {
		return nil, err
	}

	testruns := ParseTestrunsFromChart(log, chart)
	renderedTestruns := make([]*RenderedTestrun, len(testruns))
	for i, tr := range testruns {
		renderedTestruns[i] = &RenderedTestrun{
			testrun:    tr,
			Parameters: newParameters,
			Metadata: TestrunFileMetadata{
				KubernetesVersion: version,
			},
		}
	}
	return renderedTestruns, nil
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

func ParseTestrunsFromChart(log logr.Logger, chart *chartrenderer.RenderedChart) []*v1beta1.Testrun {
	testruns := make([]*v1beta1.Testrun, 0)
	for _, file := range chart.Files() {
		tr, err := util.ParseTestrun([]byte(file))
		if err != nil {
			log.Info(fmt.Sprintf("cannot parse rendered file: %s", err.Error()))
			continue
		}
		testruns = append(testruns, &tr)
	}
	return testruns
}
