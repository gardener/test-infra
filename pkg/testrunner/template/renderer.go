package template

import (
	"fmt"
	"path/filepath"

	"github.com/gardener/gardener/pkg/utils"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"helm.sh/helm/v3/pkg/strvals"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testrun_renderer"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/util"
)

// templateRenderer is the internal template templateRenderer
type templateRenderer struct {
	log      logr.Logger
	renderer *engine.Engine

	defaultValues map[string]interface{}
}

// renderState holds the state of a chart rendering.
// It implements the rerender interface to that the same chart with equal configuration can be retried.
type renderState struct {
	templateRenderer
	chartPath  string
	values     ValueRenderer
	parameters *internalParameters
}

func newTemplateRenderer(log logr.Logger, setValues, fileValues []string) (*templateRenderer, error) {
	// init engine.Engine without a restConfig and mimic helm3 defaults
	var chartRenderer engine.Engine
	chartRenderer.EnableDNS = false
	values, err := determineDefaultValues(setValues, fileValues)
	if err != nil {
		return nil, err
	}
	log.V(3).Info(fmt.Sprintf("Values: \n%s \n", util.PrettyPrintStruct(values)))
	return &templateRenderer{
		log:           log,
		renderer:      &chartRenderer,
		defaultValues: values,
	}, nil
}

// Render renders a helm chart of multiple TestRuns with values and returns a list of runs.
func (r *templateRenderer) Render(parameters *internalParameters, chartPath string, valueRenderer ValueRenderer) (testrunner.RunList, error) {
	if chartPath == "" {
		return make(testrunner.RunList, 0), nil
	}

	state := &renderState{
		templateRenderer: *r,
		chartPath:        chartPath,
		values:           valueRenderer,
		parameters:       parameters,
	}

	c, err := loader.Load(chartPath)
	if err != nil {
		return nil, err
	}

	// split all found templates into separate charts
	templates, files := splitTemplates(c.Templates)
	runs := make(testrunner.RunList, 0)
	for _, tmpl := range files {
		values, metadata, info, err := valueRenderer.Render(r.defaultValues)
		if err != nil {
			return nil, err
		}
		c.Templates = append(templates, tmpl)
		files, err := r.RenderChart(c, parameters.Namespace, values)
		if err != nil {
			return nil, err
		}

		testruns := parseTestrunsFromChart(r.log, files)

		for _, tr := range testruns {
			meta := metadata.DeepCopy()
			// Add all repositories defined in the component descriptor to the testrun locations.
			// This gives us all dependent repositories as well as there deployed version.
			if err := testrun_renderer.AddLocationsToTestrun(tr, "default", parameters.ComponentDescriptor, true, parameters.AdditionalLocations); err != nil {
				r.log.Info(fmt.Sprintf("cannot add bom locations: %s", err.Error()))
				continue
			}

			// Add runtime annotations to the testrun
			addAnnotationsToTestrun(tr, meta.CreateAnnotations())

			// add collect annotation
			metav1.SetMetaDataAnnotation(&tr.ObjectMeta, common.AnnotationCollectTestrun, "true")

			runs = append(runs, &testrunner.Run{
				Info:       info,
				Testrun:    tr,
				Metadata:   meta,
				Rerenderer: state,
			})
		}

	}

	return runs, nil
}

func (r *templateRenderer) RenderChart(chart *chart.Chart, namespace string, values map[string]interface{}) (map[string]string, error) {

	revision := 1
	options := chartutil.ReleaseOptions{
		Name:      util.RandomString(5),
		Namespace: namespace,
		Revision:  revision,
		IsInstall: true,
	}

	valuesToRender, err := chartutil.ToRenderValues(chart, values, options, nil)
	if err != nil {
		return nil, err
	}

	return r.renderer.Render(chart, valuesToRender)
}

func (s *renderState) Rerender(tr *v1beta1.Testrun) (*testrunner.Run, error) {
	runs, err := s.Render(s.parameters, s.chartPath, s.values)
	if err != nil {
		return nil, err
	}

	templateID, ok := tr.GetAnnotations()[common.AnnotationTemplateIDTestrun]
	if !ok {
		return nil, errors.Errorf("testrun %s does not have a template id", tr.GetName())
	}
	for _, run := range runs {
		if id := run.Testrun.GetAnnotations()[common.AnnotationTemplateIDTestrun]; id == templateID {
			return run, nil
		}
	}
	return nil, errors.Errorf("unable to rerender testrun for file %s", "")
}

// determineDefaultValues fetches values from all specified files and set values.
// The values are merged whereas set values overwrite file values.
func determineDefaultValues(setValues []string, fileValues []string) (map[string]interface{}, error) {
	values, err := readFileValues(fileValues)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read values from file")
	}

	for _, val := range setValues {
		newSetValues, err := strvals.ParseString(val)
		if err != nil {
			return nil, errors.Wrap(err, "unable to parse set values")
		}
		values = utils.MergeMaps(values, newSetValues)
	}
	return values, nil
}

// splitTemplates splits all found templates into 2 lists of .tpl and other files
// todo: improve to check whether the template is a testrun
func splitTemplates(all []*chart.File) ([]*chart.File, []*chart.File) {
	var (
		templates = make([]*chart.File, 0)
		others    = make([]*chart.File, 0)
	)
	for _, tmpl := range all {
		if filepath.Ext(tmpl.Name) == ".tpl" {
			templates = append(templates, tmpl)
		} else {
			others = append(others, tmpl)
		}
	}
	return templates, others
}

func parseTestrunsFromChart(log logr.Logger, files map[string]string) []*v1beta1.Testrun {
	testruns := make([]*v1beta1.Testrun, 0)
	for filename, file := range files {
		tr, err := testmachinery.ParseTestrun([]byte(file))
		if err != nil {
			log.Info(fmt.Sprintf("cannot parse rendered file %s: %s", filename, err.Error()))
			continue
		}
		metav1.SetMetaDataAnnotation(&tr.ObjectMeta, common.AnnotationTemplateIDTestrun, filename)
		testruns = append(testruns, tr)
	}
	return testruns
}
